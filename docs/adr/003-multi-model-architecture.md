# ADR-003: Adopting a Provider-Factory Multi-Model Architecture

> 🌐 [中文版本](../i18n/zh-Hans-CN/adr/003-multi-model-architecture.md)

- **Status**: Accepted
- **Date**: 2026-05
- **Deciders**: Backend Tech Lead, Product Owner

## Context

MedMemo's core value proposition includes **"multi-model free switching"** — users must be able to choose their preferred AI backend based on personal priorities (privacy, cost, quality, latency). The project targets four distinct user personas with divergent AI needs:

| Persona | Priority | Preferred Backend |
|:--------|:---------|:------------------|
| Urban white-collar (25–35) | Low latency, high quality | Cloud LLM (Kimi / GPT-4o) |
| Family health manager (40–55) | Cost control, consistent quality | Mid-tier cloud (Qwen / SiliconFlow) |
| Privacy-focused tech user (25–45) | Data never leaves device | Local Ollama / llama.cpp |
| Elderly caregiver proxy (35–50) | Simple setup, no API key hassle | Local model or pre-configured cloud |

Hard-coding a single LLM backend would:
1. **Violate user autonomy** — force all users onto one provider's pricing and data policy.
2. **Create vendor lock-in** — if the sole provider changes pricing or terms, the entire product is impacted.
3. **Prevent offline usage** — cloud-only models make the app unusable without internet, contradicting the offline-first design principle.

The application layer (`internal/application/`) must interact with LLM backends through a unified interface so that switching models does not require changes to use-case orchestrators.

## Decision

Adopt a **Provider-Factory pattern** with a unified `LLMClient` interface, supporting **6+ backend providers** through OpenAI-compatible API protocols and local HTTP APIs.

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│  Application Layer (Use Cases)                               │
│  ┌─────────────────┐  ┌─────────────────┐                   │
│  │ ChatOrchestrator│  │ MemoryRetriever │                   │
│  └────────┬────────┘  └────────┬────────┘                   │
│           │                    │                             │
│           └────────────────────┘                             │
│                    │                                         │
│           ┌────────▼────────┐                                │
│           │   port.LLMClient │  ← Interface (consumer-side) │
│           └────────┬────────┘                                │
└────────────────────┼────────────────────────────────────────┘
                     │
┌────────────────────┼────────────────────────────────────────┐
│  Adapter Layer     │                                         │
│  ┌─────────────────▼─────────────────────────────────────┐  │
│  │          llmClientFactory (Provider Factory)           │  │
│  └────────┬────────────┬────────────┬────────────┬────────┘  │
│           │            │            │            │            │
│     ┌─────▼─────┐ ┌────▼─────┐ ┌────▼─────┐ ┌───▼────┐      │
│     │OpenAI     │ │Kimi      │ │Ollama    │ │Local   │      │
│     │Adapter    │ │Adapter   │ │Adapter   │ │(Hugot) │      │
│     └───────────┘ └──────────┘ └──────────┘ └────────┘      │
└─────────────────────────────────────────────────────────────┘
```

### Supported Providers

| Type | Provider | Protocol | Default Endpoint | Auth Method |
|:-----|:---------|:---------|:-----------------|:------------|
| Cloud | Moonshot Kimi | OpenAI-compatible | `api.moonshot.cn` | API Token |
| Cloud | OpenAI GPT | OpenAI API | `api.openai.com` | API Token |
| Cloud | Alibaba Qwen | OpenAI-compatible | `dashscope.aliyuncs.com` | API Token |
| Cloud | SiliconFlow | OpenAI-compatible | `api.siliconflow.cn` | API Token |
| Local | Ollama | HTTP REST API | `localhost:11434` | None |
| Local | llama.cpp | HTTP REST API (OpenAI-compatible) | `localhost:8080` | None |

### LLMClient Interface Contract

The interface is defined in `internal/application/port/` (consumer-side declaration, per Clean Architecture rules):

```go
type LLMClient interface {
    Chat(ctx context.Context, messages []models.Message) (string, error)
    StreamChat(ctx context.Context, messages []models.Message, callback func(chunk string)) (*models.TokenUsage, error)
    CheckAvailability(ctx context.Context) (bool, string)
}
```

All adapters implement this identical contract, ensuring the application layer remains agnostic to the underlying provider.

### Context Inheritance & Window Truncation

When switching models mid-conversation, the context (message history) is preserved. The truncation strategy is:

1. **Anchor preservation**: Always retain the earliest 3 dialogue rounds in full — they establish the conversation topic and user intent.
2. **Middle compression**: Rounds 4 to N-3 are summarized into a single condensed context block via the outgoing model before the switch.
3. **Recent preservation**: Always retain the most recent 3 rounds in full to maintain conversational coherence.

When switching from a large cloud model to a smaller local model, the compression rate is higher and a confirmation dialog warns the user: *"Switching to a local model may reduce context retention. Continue?"*

### Key Constraints

1. **Provider configuration is user-editable**: Users can add, edit, or remove provider configs via the settings UI. Each config is validated through `ProviderConfig.Validate()` (base fields + auth method params).
2. **Endpoint and model resolution**: The factory resolves the final API endpoint and model ID through a priority chain: user-configured value → provider-type default → fallback constant.
3. **Availability probing**: `CheckAvailability` performs a lightweight HEAD or minimal completion request; unavailable providers are grayed out in the UI but not deleted.

## Consequences

### Positive Impacts

- **User autonomy**: Each user chooses the backend that matches their privacy, cost, and quality preferences.
- **Vendor-risk mitigation**: No single provider failure can render the app unusable; users can switch instantly.
- **Offline capability**: Local Ollama/llama.cpp backends allow full offline operation, fulfilling the offline-first principle.
- **Community extensibility**: New providers can be added by implementing the `LLMClient` interface in `internal/adapters/ai/` without touching application logic.

### Negative Impacts

- **Testing matrix expansion**: Each provider adapter requires separate integration tests (mock server + real API smoke tests), increasing CI maintenance.
- **Context switching UX friction**: Model switches mid-conversation may cause subtle behavior changes (different tokenization, refusal patterns, formatting). The compression/summary step adds latency.
- **Configuration complexity**: Users must understand API keys, endpoints, and model IDs — a steeper onboarding curve than "it just works" single-model apps.
- **Local model resource pressure**: Running Ollama on the same machine as ONNX Runtime and the v1.x SQLCipher/SQLite + sqlite-vec stack pushes memory usage; minimum recommended RAM is 8GB. DuckDB remains a v2+ planning candidate, not a v1.x runtime dependency.

## Alternatives Considered

| Alternative | Rejection Reason |
|:------------|:-----------------|
| Single hard-coded cloud model (e.g., GPT-4o only) | Violates multi-model switching pillar; vendor lock-in; no offline capability; forces all users onto one pricing model |
| Fully abstract custom protocol for all LLMs | Over-engineering; Go ecosystem lacks mature "universal LLM SDK"; OpenAI-compatible API is the de-facto standard |
| Per-provider custom adapter without factory | Leads to scattered initialization logic; violates DRY; harder to add new providers |

## Related Documents

- [docs/ARCHITECTURE.md](../ARCHITECTURE.md) — System architecture overview
- `internal/application/port/llm_client.go` — `LLMClient` interface definition
- `internal/adapters/ai/provider.go` — `llmClientFactory` implementation
- `pkg/models/provider.go` — `ProviderConfig` validation logic

---

*Last updated: 2026-07-09*
