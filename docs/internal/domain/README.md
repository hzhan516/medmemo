# Domain Layer

> 🌐 [中文版本](../../i18n/zh-Hans-CN/internal/domain/README.md)

## Role

The domain layer contains MedMemo's core entities, policies, domain services, and repository abstractions.

The layer is deliberately free of framework, database, UI, and network dependencies.

## Structure

```
internal/domain/
├── entity/       # Conversation, Message, ExtractedFact, HealthMemory, changelog
├── repository/   # Domain-facing repository interfaces
└── policy/       # Compliance and sensitivity policies
```

## Import Rules

| Allowed | Prohibited |
|---------|------------|
| Go standard library | any `internal/*` package |
| `pkg/models/` | `pkg/desensitizer/` |

## When to Add Code Here

- Add or change core business entities.
- Define domain errors and invariants.
- Define repository contracts consumed by domain/application logic.

---

*Last updated: 2026-07-09*
