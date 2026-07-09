# 架构 Wiki

> 🌐 [English Version](../../../kb/Architecture-Wiki.md)

## 权威参考

- [Architecture](../../../ARCHITECTURE.md)
- [Development Guide](../../../DEVELOPMENT.md)
- [API Documentation](../../../API.md)
- [Compliance](../../../COMPLIANCE.md)
- [Security](../../../SECURITY.md)

## 关键规则

- `internal/domain/` 只导入 Go 标准库和 `pkg/models/`。
- 新依赖通过仓库根 `wire.go` 接入；禁止手动编辑 `wire_gen.go`。
- ONNX 推理必须通过串行化 worker session 执行。

---

*最后更新：2026-07-09*
