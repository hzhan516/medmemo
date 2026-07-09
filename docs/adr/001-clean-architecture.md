# ADR-001: Adopting the Clean Architecture Four-Layer Directory Model

> 🌐 [中文版本](../i18n/zh-Hans-CN/adr/001-clean-architecture.md)

- **Status**: Accepted
- **Date**: 2026-05
- **Deciders**: Backend Tech Lead

## Context

MedMemo is a desktop health information tool with the following core requirements:
1. **Long-term maintainability**: Health data models (memories, family relations, conversations) evolve rapidly across versions and require clear business boundaries.
2. **Testability**: Medical compliance logic (de-identification, interception, terminology validation) must be unit-testable in isolation without external frameworks.
3. **Replaceability**: Local AI inference engines (ONNX Runtime), databases (SQLCipher/SQLite + sqlite-vec, with DuckDB/Kùzǔ only as v2+ planning candidates), and LLM providers all carry replacement or downgrade risks.

Before project kickoff, the team evaluated three backend architecture styles:

| Architecture Style | Pros | Cons | Suitability |
|:-------------------|:-----|:-----|:------------|
| **MVC / Three-Layer** | Simple and intuitive, quick onboarding | Business logic coupled with frameworks, hard to unit-test | ❌ Does not meet testability requirements |
| **Hexagonal Architecture (Ports & Adapters)** | Clear input/output ports, adapters replaceable | Non-uniform directory naming, steep learning curve for new members | ⚠️ Abstract concepts, limited Go ecosystem support |
| **Clean Architecture (Four-Layer)** | Strict inward dependency direction, domain layer testable with pure Go, naturally fits Go package mechanism | More directory levels, small features may span multiple layers | ✅ Best fit for MedMemo |

## Decision

Adopt **Robert C. Martin's Clean Architecture four-layer model**, mapped to Go package structure as follows:

```
internal/
├── domain/         # Entities Layer — core business entities and rules
├── application/    # Use Cases Layer — use case orchestration and port definitions
├── adapters/       # Interface Adapters Layer — external system adapter implementations
└── infrastructure/ # Frameworks & Drivers Layer — technical framework encapsulation
```

### Key Constraints

1. **Zero external dependencies in the domain layer**: Only Go standard library and `pkg/models/` are allowed. Violations will be blocked by CI depguard checks.
2. **Application layer does not directly call adapter implementations**: Decoupled via interfaces (Ports); the adapter layer implements interfaces defined by the application layer.
3. **Infrastructure layer knows nothing about business logic**: Only encapsulates third-party frameworks (Wails, SQLCipher/SQLite, sqlite-vec, ONNX Runtime); does not import any business packages.
4. **Dependency injection completed at compile time via Wire**: Runtime reflection injection is prohibited to ensure cold-start speed and compile-time safety.

## Consequences

### Positive Impacts

- **100% unit-testable domain layer**: Pure Go standard library, no need to mock external dependencies.
- **Controlled technology replacement cost**: When replacing databases or AI inference engines, only the adapter/infrastructure layers need rewriting; domain/application layers remain unaffected.
- **New-member onboarding is guided by structure**: The four-layer directories themselves serve as architecture documentation; combined with per-layer READMEs, new developers can quickly locate code.

### Negative Impacts

- **Small feature changes may touch 4+ files**: Adding an entity requires changes to domain (entity definition), application (use case), adapter (repository implementation), and infrastructure (database table), which is slightly less efficient than MVC.
- **Over-engineering risk**: For simple CRUD operations, four-layer splitting feels redundant. Mitigation: application layer is allowed to passthrough simple queries directly without forcing every operation through a domain service.

## Alternatives Considered

| Alternative | Rejection Reason |
|:------------|:-----------------|
| Traditional Three-Layer (Controller-Service-DAO) | Business logic coupled with frameworks; difficult to perform pure unit tests on the compliance engine in an offline desktop scenario |
| Feature-Based Partitioning | MedMemo modules have strong cross-cutting concerns (memory pool ↔ family graph ↔ compliance engine); feature-based partitioning would create circular dependencies |

## Related Documents

- [docs/DEVELOPMENT.md](../DEVELOPMENT.md) — Detailed development standards and dependency rules
- [docs/internal/domain/README.md](../internal/domain/README.md)
- [docs/internal/application/README.md](../internal/application/README.md)
- [docs/internal/adapters/README.md](../internal/adapters/README.md)
- [docs/internal/infrastructure/README.md](../internal/infrastructure/README.md)

---

*Last updated: 2026-07-09*
