# Application Layer

> 🌐 [中文版本](../../i18n/zh-Hans-CN/internal/application/README.md)

## Role

The application layer orchestrates user use cases. It defines what the system can do and which external capabilities it needs through ports.

## Structure

```
internal/application/
├── usecase/    # ChatOrchestrator, MemoryRetriever, TitleGenerator, compression
├── port/       # LLMClient, repositories, detectors, token counters
├── pipeline/   # Two-level de-identification orchestration
├── stream/     # Wails event streaming broker
└── updater/    # Update use cases
```

## Import Rules

| Allowed | Prohibited |
|---------|------------|
| `internal/domain/*` | `internal/adapters/*` |
| `pkg/models/` | `internal/infrastructure/*` |
| Go standard library | UI/framework packages |

## Responsibilities

1. Coordinate workflows such as sending a message, retrieving memories, de-identifying outbound text, and applying compliance checks.
2. Define transaction and timeout boundaries for each use case.
3. Declare ports that outer layers implement.

---

*Last updated: 2026-07-09*
