# Domain Layer

> 🌐 [中文版本](../../docs/i18n/zh-Hans-CN/internal/domain/README.md)

## Why This Layer Exists

The Domain Layer is the innermost ring of Clean Architecture. It holds MedMemo's core business entities and domain rules — the logic that would remain unchanged even if we switched from Wails to Tauri, from DuckDB to PostgreSQL, or from ONNX to TensorFlow.

By isolating business concepts from technical details, we ensure that:
- Business rules are testable without databases, HTTP servers, or UI frameworks
- Changes in infrastructure do not ripple into business logic
- New developers can understand what the system *does* before learning how it *works*

---

## Directory Structure

```
internal/domain/
├── entity/       # Core business entities: Conversation, Memory, FamilyMember, HealthMemory...
├── repository/   # Repository interfaces (Ports): MemoryRepository, FamilyRepository...
└── policy/       # Policy interfaces: de-identification strategies, compliance policies
```

| Package | Purpose | Example Types |
|---------|---------|--------------|
| `entity/` | Pure business objects with behavior | `Conversation`, `HealthMemory`, `FamilyMember` |
| `repository/` | Contracts for data persistence | `MemoryRepository`, `FamilyRepository` |
| `policy/` | Abstractions for compliance & sensitivity | `DeidentifyPolicy`, `CompliancePolicy` |

---

## Import Constraints (Iron Rule)

| Allowed Imports | Forbidden Imports |
|-----------------|-------------------|
| Go standard library | `github.com/hzhan516/medmemo/internal/**/*` |
| `github.com/hzhan516/medmemo/pkg/models/` | `github.com/hzhan516/medmemo/pkg/desensitizer/` |

> ⚠️ Violating these rules will be blocked by the CI `depguard` check.

This layer knows **nothing** about:
- HTTP requests or REST APIs
- Database connection strings
- UI frameworks (React, Wails)
- AI model inference libraries
- Configuration file formats

---

## When to Add Code Here

- **New business entities** (e.g., `Conversation`, `HealthMemory`)
- **Domain errors** (e.g., `ErrRecordNotFound`, `ErrValidationFailed`)
- **Repository interfaces** (implemented by the adapter layer)
- **Domain events** (e.g., `MemoryCreated`)
- **Cross-entity business rules** that cannot belong to a single entity

---

## Design Principles

- **Entities encapsulate behavior**, not just data. A `Conversation` knows how to rename itself; a `FamilyMember` knows how to validate relationship graphs.
- **Value objects are immutable**. Dates, sensitivity levels, and medical record IDs are value objects.
- **Errors are part of the domain**. `ErrRecordNotFound` is defined here so that adapters can map their native errors (SQL `no rows`, HTTP 404) to a common language.

---

## Example

```go
// entity/conversation.go
package entity

import "time"

type Conversation struct {
	ID        string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c *Conversation) Rename(title string) error {
	if title == "" {
		return ErrValidationFailed
	}
	c.Title = title
	c.UpdatedAt = time.Now()
	return nil
}
```

---

## Related Layers

- [Application Layer](../application/README.md) — Orchestrates domain objects into use cases
- [Adapters Layer](../adapters/README.md) — Implements the repository interfaces defined here
- [Infrastructure Layer](../infrastructure/README.md) — Provides the technical capabilities adapters need

---

*Last updated: 2026-05-19*
