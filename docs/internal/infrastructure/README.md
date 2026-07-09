# Infrastructure Layer

> 🌐 [中文版本](../../i18n/zh-Hans-CN/internal/infrastructure/README.md)

## Role

The infrastructure layer wraps technical frameworks and platform-specific resources. It does not know MedMemo business rules.

## Structure

```
internal/infrastructure/
├── config/      # Local configuration loader and validation
├── database/    # SQLCipher/SQLite connectors and sqlite-vec setup
├── onnx/        # ONNX Runtime and Hugot engine wrappers
├── secret/      # Platform keyring wrapper
└── updater/     # Platform update installer helpers
```

## Import Rules

| Allowed | Prohibited |
|---------|------------|
| Go standard library | `internal/domain/*` |
| third-party framework libraries | `internal/application/*` |
| platform libraries | `internal/adapters/*` |

## Responsibilities

1. Initialize technical resources such as database connectors, ONNX runtime paths, configuration, and keyring access.
2. Expose concrete types for Wire assembly.
3. Encapsulate cross-platform differences.

---

*Last updated: 2026-07-09*
