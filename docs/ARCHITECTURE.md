# Architecture

> 🌐 [中文版本](./i18n/zh-Hans-CN/ARCHITECTURE.md)

> This document describes MedMemo's system architecture design, module breakdown, and key decisions. Intended for development teams, technical reviewers, and open-source contributors.

---

## System Architecture Overview

```mermaid
flowchart TB
    subgraph FW["Frameworks & Drivers (Presentation Layer)"]
        WAILS["Wails v2 Runtime"]
        TRAY["System Tray"]
        WEBVIEW["WebView2 (UI)"]
    end

    subgraph ADPT["Interface Adapters (Adapter Layer)"]
        HTTP["HTTP Client<br/>Cloud API Adapter"]
        DUCK["DuckDB Repository"]
        KUZU["Kùzǔ Graph DB"]
        ONNX["ONNX Runtime<br/>Hugot Adapter"]
        CFG["Config/Secret Loader"]
    end

    subgraph APP["Application Core (Use Case Layer)"]
        UC_CHAT["ChatOrchestrator<br/>Conversation Orchestration"]
        UC_MEM["MemoryRetriever<br/>Memory Retrieval"]
        PIPE["DeidentifyPipeline<br/>Three-Tier De-Identification"]
        PORT_LLM["Port: LLMClient"]
        PORT_DB["Port: RecordStore"]
        PORT_DET["Port: SensitiveDetector"]
    end

    subgraph DOM["Domain (Entity Layer)"]
        ENT["Entity: Conversation<br/>HealthMemory, FamilyMember"]
        POLICY["Policy: Compliance<br/>SensitiveData"]
        SVC["Service: MemoryConsolidator<br/>FamilyGraphAnalyzer"]
    end

    FW --> ADPT
    ADPT --> APP
    APP --> DOM

    HTTP -. implements .-> PORT_LLM
    DUCK -. implements .-> PORT_DB
    ONNX -. implements .-> PORT_DET

    style DOM fill:#e8f4e8,stroke:#2d5a2d,stroke-width:2px
    style APP fill:#e8ecf4,stroke:#2d3a5a,stroke-width:2px
    style ADPT fill:#f4ece8,stroke:#5a3a2d,stroke-width:2px
    style FW fill:#f0e8f4,stroke:#4a2d5a,stroke-width:2px
```

Solid arrows = compile-time dependency direction (outer → inner). Dashed arrows = interface implementation relationship (adapters implement application-layer ports).

---

## Four-Layer Architecture Mapping

| Layer | Responsibility | Typical Package Path | Compile Dependency Principle |
|:------|:---------------|:---------------------|:-----------------------------|
| **Frameworks & Drivers** | Wails v2 frontend bridge, system tray, window management | `cmd/health-assistant/` | May only depend on inner interfaces |
| **Interface Adapters** | HTTP client, DuckDB/Kùzǔ repositories, ONNX inference adapters | `internal/adapters/*` | Depends on interfaces defined by Application layer |
| **Application Core** | Use case orchestration, de-identification pipeline, port definitions | `internal/application/*` | Does not depend on any outer layer or framework |
| **Domain** | Entity definitions, domain services, policy abstractions | `internal/domain/*` | **Zero external dependencies**, pure Go standard library |

---

## Core Data Flows

### User Conversation Request Flow

```
User Input
  → Wails Frontend (React)
    → Backend HTTP Handler (Wails Bindings)
      → ChatOrchestrator (Use Case Orchestration)
        → DeidentifyPipeline (L1 Rules → L2 NER → L3 Keywords)
          → Memory Retrieval (DuckDB/SQLite keyword + vector search)
            → Context Assembly (L1 Working Memory + L2/L3 Retrieval Results)
              → LLMClient (OpenAI/Kimi/Ollama)
                → Streaming Response Sentence Buffering
                  → Compliance Detection (L1~L4)
                    → Output Restoration (P2 Placeholder Backfill)
                      → Wails Events Push to Frontend
                        → Frontend Typewriter Effect Rendering
```

### Memory Write Flow

```
Conversation End / Archive Trigger
  → MemoryRetriever.ArchiveConversation
    → Conversation Summary Generation (LLM / Rule Template)
      → Entity Extraction (ONNX NER)
        → Conflict Detection (MemoryConsolidator)
          → [No Conflict] Direct write to DuckDB/SQLite
          → [Conflict] Highlight conflict → Request user confirmation → Write
            → Knowledge Graph Update (Kùzǔ)
              → Vector Index Update (DuckDB vss)
```

---

## Module Dependency Matrix

```
                    domain   application   adapters   infrastructure   pkg
                  ┌────────┬─────────────┬──────────┬────────────────┬─────┐
domain            │   -    │      -      │    -     │       -        │  √  │
application       │   √    │      -      │    -     │       -        │  √  |
adapters          │   √    │      √      │    -     │       √        │  √  |
infrastructure    │   -    │      -      │    -     │       -        │  -  │
pkg               │   -    │      -      │    -     │       -        │  -  │
                  └────────┴─────────────┴──────────┴────────────────┴─────┘
```

`√` = import allowed, `-` = import prohibited.

---

## Key Design Decisions (ADR Index)

| ADR | Topic | Decision | Status |
|:-----|:------|:---------|:-------|
| [ADR-001](adr/001-clean-architecture.md) | Clean Architecture Four-Layer Model | Robert C. Martin's Clean Architecture | Accepted |
| [ADR-002](adr/002-duckdb-selection.md) | Embedded Analytics Database | DuckDB v1.7+ for analytics + vector search | Accepted |
| [ADR-003](adr/003-multi-model-architecture.md) | Multi-Model LLM Architecture | Provider-Factory with 6+ backends | Accepted |
| [ADR-004](adr/004-onnx-integration.md) | Local AI Inference | Hugot + ONNX Runtime (int8 DistilBERT NER) | Accepted |
| ADR-005 | Dependency Injection | Google Wire (compile-time) | Accepted |
| ADR-006 | Vector Retrieval | DuckDB `vss` extension + HNSW index | Accepted |
| ADR-007 | De-Identification Architecture | L1 Rules + L2 NER ONNX + L3 Keyword Dictionary | Accepted |
| ADR-008 | Streaming Compliance Detection | Punctuation sentence buffering + push after detection | Accepted |

Full ADR documents are located in the `docs/adr/` directory.

---

## Build Artifact Form

| Component | Size Budget | Description |
|:----------|:------------|:------------|
| Main Binary | ~50MB | Wails Runtime + DuckDB engine + Go runtime + business logic |
| Resource Directory | ~100MB | ONNX model ~50MB + Kùzǔ library ~15MB + dictionary/rules ~10MB + frontend assets |
| **Total** | **< 150MB** | Single binary + resource directory |

Supported platforms: Windows 10+ / macOS 12+ / Linux (Ubuntu 20.04+), architectures x64 and ARM64.
