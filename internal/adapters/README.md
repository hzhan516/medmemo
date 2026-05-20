# Adapters Layer

> 🌐 [中文版本](../../docs/i18n/zh-Hans-CN/internal/adapters/README.md)

## Why This Layer Exists

The Adapters Layer is the third ring of Clean Architecture. It translates data formats and protocols from external systems into forms the application layer understands.

This layer implements the interfaces defined in `application/port/`, bridging the gap between MedMemo's business core and the outside world — whether that world is an OpenAI API endpoint, a DuckDB database file, or a local ONNX Runtime session.

By isolating external system details here, we ensure that:
- Swapping from OpenAI to Kimi requires changing only one adapter file
- Database migrations do not affect use case logic
- Third-party API changes are contained in a single location

---

## Directory Structure

```
internal/adapters/
├── ai/           # AI model client adapters: OpenAI, Kimi, Ollama, Local...
├── repository/   # Data persistence adapters: DuckDB, SQLite, Kùzǔ implementations...
├── detector/     # Sensitive detection adapters: rule engine, NER model...
└── dto/          # Data Transfer Object layer: external format ↔ domain format
```

| Package | Purpose | Example Types |
|---------|---------|--------------|
| `ai/` | LLM provider integrations | `OpenAIAdapter`, `KimiAdapter`, `OllamaAdapter` |
| `repository/` | Database access implementations | `MemoryRepo`, `FamilyRepo` |
| `detector/` | PII and sensitive data detection | `RuleDetector`, `NERDetector` |
| `dto/` | Pure transformation functions | `ToDomain()`, `FromDomain()` |

---

## Import Constraints

| Allowed Imports | Forbidden Imports |
|-----------------|-------------------|
| `github.com/hzhan516/medmemo/internal/domain/*` | `github.com/hzhan516/medmemo/internal/application/*` |
| `github.com/hzhan516/medmemo/internal/infrastructure/*` | `github.com/hzhan516/medmemo/cmd/*` |
| `github.com/hzhan516/medmemo/pkg/models/` | — |

---

## Core Responsibilities

1. **Interface Implementation** — Fulfill contracts defined in `application/port/` (e.g., `LLMClient`, `MemoryRepository`)
2. **Data Transformation** — Convert external API responses and database records into domain entities via the DTO layer
3. **Error Mapping** — Translate external errors (HTTP timeouts, database connection failures) into domain errors

---

## Design Principles

- **One external system, one adapter**. OpenAI has its own adapter; Kimi has its own. Differences in headers, authentication, and response shapes are encapsulated locally.
- **DTO conversions are pure functions**. Functions in `dto/` are stateless, side-effect-free, and return `error` rather than panicking.
- **Graceful degradation**. Adapters implement `CheckAvailability()`. If an LLM provider is unreachable, the system can fall back to a local model or cached responses.

---

## Example

```go
// ai/openai_adapter.go
package ai

import "github.com/hzhan516/medmemo/internal/application/port"

type OpenAIAdapter struct {
	client *http.Client
	apiKey string
}

func (a *OpenAIAdapter) Chat(messages []port.Message) (string, error) {
	// Implement OpenAI API call
}

// Registered via Wire ProviderSet
var ProviderSet = wire.NewSet(
	NewOpenAIAdapter,
	wire.Bind(new(port.LLMClient), new(*OpenAIAdapter)),
)
```

---

## Related Layers

- [Application Layer](../application/README.md) — Defines the ports this layer implements
- [Domain Layer](../domain/README.md) — Provides the entities this layer transforms data into
- [Infrastructure Layer](../infrastructure/README.md) — Provides the raw clients and runtimes this layer uses

---

*Last updated: 2026-05-19*
