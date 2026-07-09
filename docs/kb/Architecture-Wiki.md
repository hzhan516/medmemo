# Architecture Wiki

> 🌐 [中文版本](../i18n/zh-Hans-CN/kb/Architecture-Wiki.md)

## Canonical References

- [Architecture](../ARCHITECTURE.md)
- [Development Guide](../DEVELOPMENT.md)
- [API Documentation](../API.md)
- [Compliance](../COMPLIANCE.md)
- [Security](../SECURITY.md)

## Key Rules

- `internal/domain/` imports only the Go standard library and `pkg/models/`.
- New dependencies are wired through root `wire.go`; never edit `wire_gen.go` by hand.
- ONNX inference must run through serialized worker sessions.

---

*Last updated: 2026-07-09*
