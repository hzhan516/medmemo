# ADR-004: Adopting Hugot + ONNX Runtime for Local NER Inference

> 🌐 [中文版本](../i18n/zh-Hans-CN/adr/004-onnx-integration.md)

- **Status**: Accepted
- **Date**: 2026-05
- **Deciders**: Backend Tech Lead, AI Engineer

## Context

MedMemo's **three-tier de-identification pipeline** (L1 Rules → L2 NER → L3 Keywords) requires a local Named Entity Recognition (NER) model to identify sensitive entities in user health text before any data leaves the device. The L2 NER stage is the accuracy-critical layer — it must recognize:

- Person names (patients, doctors, family members)
- Organization names (hospitals, clinics, insurance companies)
- Disease names and medical conditions
- Medication names and drug aliases

The NER engine must satisfy four hard constraints:

1. **Privacy zero-outbound**: All inference happens locally; no user text is sent to any external service for entity recognition.
2. **Offline availability**: Must function without any network connection.
3. **Latency budget**: <200ms for short text (<500 chars), <500ms for long text (<2000 chars) — measured at p95.
4. **Size budget**: Model file + runtime libraries must fit within the ~100MB resource directory budget.

Before project kickoff, the team evaluated three local inference approaches:

| Approach | Pros | Cons | Suitability |
|:---------|:-----|:-----|:------------|
| **Hugot + ONNX Runtime** | Mature Go binding (`github.com/knights-analytics/hugot`); broad model ecosystem (BERT, DistilBERT, RoBERTa); int8 quantization reduces model to ~50MB; CPU inference fast enough for NER | CGO cross-compilation complexity; ONNX Runtime session is **not thread-safe**, requiring serialization | ✅ Best fit for accuracy + privacy + size |
| **Remote NER API** (e.g., cloud NLP service) | Highest accuracy; zero local model size | Violates privacy-first principle; requires network; adds latency (RTT + server time) | ❌ Violates zero-outbound constraint |
| **Pure Go NLP** (e.g., `prose` v3, rule-based) | Zero CGO; trivial cross-compilation; tiny footprint | Accuracy far below deep-learning NER for medical domain; cannot recognize rare disease names or medication aliases | ❌ Does not meet accuracy requirements |

## Decision

Adopt **Hugot v0.7+ + ONNX Runtime 1.17+** for local NER inference, using an **int8-quantized DistilBERT token-classification model** (~50MB). Inference is orchestrated through a **2-worker serial execution model** to handle ONNX Runtime's thread-unsafe session constraint.

### Architecture

```
User Input Text
    │
    ▼
┌─────────────────────────────────────────┐
│  L2 NER Stage (internal/application/)   │
│  ┌─────────────────────────────────┐   │
│  │  PipelineInput (text + metadata)│   │
│  └─────────────┬───────────────────┘   │
│                │                        │
│  ┌─────────────▼───────────────────┐   │
│  │  2-Worker ONNX Inference Pool   │   │
│  │  ┌─────────┐    ┌─────────┐    │   │
│  │  │ Worker 0│    │ Worker 1│    │   │
│  │  │ Session │    │ Session │    │   │
│  │  │ (Mutex) │    │ (Mutex) │    │   │
│  │  └────┬────┘    └────┬────┘    │   │
│  │       └──────────────┘         │   │
│  │          (Buffered Chan)        │   │
│  └─────────────┬───────────────────┘   │
│                │                        │
│  ┌─────────────▼───────────────────┐   │
│  │  EntityList → Sensitivity Grading│   │
│  │  (P1 Public / P2 Internal / P3) │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

### Key Technical Choices

1. **Model**: DistilBERT fine-tuned for token classification (BiLSTM-CRF head), exported to ONNX, then int8-quantized. Quantization reduces model size from ~250MB (fp32) to ~50MB with <2% accuracy loss on medical entity benchmarks.

2. **Worker model**: 2 fixed inference workers, each holding an independent ONNX Session (~80–100MB RAM per session, total ~200MB). Tasks are dispatched via a buffered channel (capacity 16). Each worker serializes `Session.Run()` calls because ONNX Runtime sessions are **not thread-safe**.

3. **Go binding**: `github.com/knights-analytics/hugot` provides a high-level pipeline API (`TokenClassificationPipeline`) that abstracts ONNX Runtime C API calls into idiomatic Go:
   ```go
   pipeline, err := hugot.NewTokenClassificationPipeline(
       sessionOptions, hugot.TokenClassificationConfig{
           ModelPath: modelPath,
           MaxLength: 512,
           Truncation: true,
       })
   ```

4. **Platform-specific library distribution**: ONNX Runtime dynamic libraries are bundled per platform:
   - macOS: `libonnxruntime.dylib` (~15MB)
   - Windows: `onnxruntime.dll` (~15MB)
   - Linux: `libonnxruntime.so` (~15MB)
   The `internal/infrastructure/onnx/` package selects the correct library path at runtime via `runtime.GOOS`.

5. **Initialization strategy**: ONNX Session initialization happens **asynchronously in the background** during app startup. Target: <400ms initialization delay, non-blocking for the first UI paint (PER-03).

### Integration with De-Identification Pipeline

The L2 NER stage sits between L1 (rule engine, <1ms) and L3 (keyword dictionary, <5ms):

```
L1 Rule Engine ──(missed spans)──► L2 NER ONNX ──(low-confidence spans)──► L3 Keywords
         │                                    │
         └───── P3 hard-mask ────────────────┘
```

- Spans identified by L1 with `Confidence == 1.0` bypass L2 entirely (pipeline short-circuit).
- L2 NER outputs are graded into P1/P2/P3 sensitivity levels. P3 spans are **hard-replaced** with irreversible masks before any cloud API call.
- Low-confidence L2 outputs (<0.7) are forwarded to L3 keyword dictionary matching as a fallback.

## Consequences

### Positive Impacts

- **Privacy guarantee**: User health text never leaves the device for entity recognition; the NER model runs entirely in-process.
- **Offline resilience**: De-identification works without any network, fulfilling the offline-first principle.
- **Accuracy**: DistilBERT NER significantly outperforms rule-only or pure-Go approaches on medical entity recognition benchmarks.
- **Model upgradability**: Users can swap the ONNX model file in `resources/models/` without recompiling the application.

### Negative Impacts

- **CGO cross-compilation burden**: Building for three platforms requires platform-specific ONNX Runtime libraries and CGO toolchain setup, complicating CI.
- **Memory pressure**: 2 ONNX Sessions consume ~200MB RAM. On systems with <8GB RAM, this competes with DuckDB, local LLM (Ollama), and the OS.
- **Non-thread-safe session constraint**: The 2-worker serial model limits NER throughput. Under burst input scenarios (e.g., bulk health record import), requests queue in the channel.
- **Model size**: 50MB is the largest single file in the resource directory; slow initial download may degrade first-launch experience on slow connections.

## Alternatives Considered

| Alternative | Rejection Reason |
|:------------|:-----------------|
| Remote NER API (cloud NLP service) | Violates the core privacy pillar; health text would leave the device before de-identification, defeating the purpose of the pipeline |
| Pure Go token classification (`prose` v3 + custom rules) | Insufficient accuracy for medical domain; cannot recognize rare disease names, medication aliases, or hospital names with acceptable precision |
| TensorFlow Lite Go binding | Weaker Go ecosystem; fewer pre-trained medical NER models; cross-platform library distribution more complex than ONNX Runtime |
| Run NER in a separate process (gRPC/IPC) | Adds process management complexity; no significant memory savings (model still loaded); latency overhead of IPC |

## Related Documents

- [docs/ARCHITECTURE.md](../ARCHITECTURE.md) — System architecture overview and data flow
- `internal/infrastructure/onnx/` — ONNX Runtime Go binding and session management
- `pkg/desensitizer/` — Rule engine (L1) and pipeline orchestrator
- `internal/application/pipeline/` — Three-tier de-identification pipeline
