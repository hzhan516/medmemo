# Adapters Layer

> 🌐 [中文版本](../../i18n/zh-Hans-CN/internal/adapters/README.md)

## Role

The adapters layer implements application ports and translates external systems into shapes the use cases understand.

## Structure

```
internal/adapters/
├── ai/           # LLM adapters: OpenAI-compatible APIs and local endpoints
├── auth/         # OAuth, CLI token, and token-refresh adapters
├── detector/     # Rule-based and ONNX-backed detectors
├── repository/   # SQLCipher/SQLite repositories
└── updater/      # GitHub release adapter
```

## Import Rules

| Allowed | Prohibited |
|---------|------------|
| `internal/domain/*` | `internal/application/*` |
| `internal/infrastructure/*` | repository-root app assembly |
| `pkg/models/` | direct UI dependencies |

## Responsibilities

1. Implement application-layer ports such as `LLMClient`, repositories, and health-check adapters.
2. Convert external API responses and database rows into domain entities or shared models.
3. Map external failures into contextual wrapped errors.

---

*Last updated: 2026-07-09*
