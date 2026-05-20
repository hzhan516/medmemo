# Application Layer

> 🌐 [中文版本](../../docs/i18n/zh-Hans-CN/internal/application/README.md)

## Why This Layer Exists

The Application Layer is the second ring of Clean Architecture. It orchestrates domain objects to fulfill specific user use cases — such as "send a chat message" or "retrieve relevant memories for a conversation."

This layer defines **what the system can do** (Use Cases) and **what external capabilities it needs** (Ports). It does not know who provides those capabilities — database access, HTTP calls, and AI inference are all injected by the adapter layer.

By keeping this layer free of framework code, we ensure that:
- Use cases are testable with mocked ports
- Business workflows are visible in one place
- New features can be added without touching infrastructure

---

## Directory Structure

```
internal/application/
├── usecase/    # Use case implementations: ChatOrchestrator, MemoryRetriever, TitleGenerator...
├── port/       # Port definitions: LLMClient, MemoryRepository, SensitiveDetector, ComplianceChecker...
└── pipeline/   # De-identification pipeline orchestrator: coordinates L1/L2/L3 sanitization
```

| Package | Purpose | Example Types |
|---------|---------|--------------|
| `usecase/` | Concrete business workflows | `ChatOrchestrator`, `MemoryRetriever` |
| `port/` | Interfaces for external capabilities | `LLMClient`, `MemoryRepository`, `ComplianceChecker` |
| `pipeline/` | Multi-stage data transformation | `DeidentifyPipeline` |

---

## Import Constraints

| Allowed Imports | Forbidden Imports |
|-----------------|-------------------|
| `github.com/hzhan516/medmemo/internal/domain/*` | `github.com/hzhan516/medmemo/internal/adapters/*` |
| `github.com/hzhan516/medmemo/pkg/models/` | `github.com/hzhan516/medmemo/internal/infrastructure/*` |
| Go standard library | — |

This layer never imports adapter implementations or infrastructure packages directly. All external dependencies arrive through the `port/` interfaces.

---

## Core Responsibilities

1. **Use Case Orchestration** — Receive input → call domain objects → coordinate adapters → return output
2. **Transaction Boundaries** — Define the atomic boundary of a use case (e.g., "send message" includes sanitization, API call, compliance check, and persistence)
3. **Port Definitions** — Declare external capabilities via Go interfaces; implementations are injected by the adapter layer

---

## Design Principles

- **One file per use case**. `ChatOrchestrator` handles the full chat flow; `MemoryRetriever` handles memory search. This makes workflows easy to locate and test.
- **Ports are minimal**. `LLMClient` has only `Chat()`, `StreamChat()`, and `CheckAvailability()`. No provider-specific types leak through.
- **Pipelines are composable**. The de-identification pipeline can run L1 (rule-based), L2 (NER model), or L3 (keyword dictionary) independently or in sequence.

---

## Example

```go
// port/llm.go
package port

type LLMClient interface {
	Chat(messages []Message) (string, error)
	StreamChat(messages []Message, callback func(chunk string))
	CheckAvailability() (bool, string)
}

// usecase/chat.go
package usecase

func (o *ChatOrchestrator) Execute(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	// 1. Emergency symptom detection
	// 2. Three-stage de-identification pipeline
	// 3. LLM invocation
	// 4. Compliance interception
	// 5. Message persistence
}
```

---

## Related Layers

- [Domain Layer](../domain/README.md) — Provides the entities and rules this layer orchestrates
- [Adapters Layer](../adapters/README.md) — Implements the ports defined here
- [Infrastructure Layer](../infrastructure/README.md) — Provides the raw technical capabilities adapters consume

---

*Last updated: 2026-05-19*
