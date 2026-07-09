# ADR-002: 将 DuckDB/Kùzǔ 延后到 v2+ 规划

> 🌐 [English Version](../../../adr/002-duckdb-selection.md)

- **状态**: v1.x 已替代
- **日期**: 2026-05
- **最后更新**: 2026-07-09
- **决策人**: 后端技术负责人、AI 工程师

## 背景

最初的存储设计曾考虑将 DuckDB/Kùzǔ 用于 v2+ 分析与图谱工作负载：DuckDB 用于未来家族健康分析和向量实验，Kùzǔ 用于未来家族关系图谱遍历。

已发布的 v1.x 产品采用更小的本地优先存储栈：

- SQLCipher/SQLite 存储会话、消息、Provider 设置、事实、审计日志和知识文档。
- sqlite-vec 对已审批事实和本地知识文档执行语义向量搜索。
- DuckDB/Kùzǔ 仅作为 v2+ 规划中的冻结存根，v1.x 运行时不启用。

## 决策

在 v1.x 中替代原先的 DuckDB/Kùzǔ 活跃存储决策。DuckDB/Kùzǔ 仅保留为 v2+ 规划候选；SQLCipher/SQLite + sqlite-vec 是 v1.x 的事实来源。

## 结果

- v1.x 打包体积更小，避免额外 DuckDB/Kùzǔ CGO/runtime 库。
- 存储和迁移面更简单：加密 SQLite 加 sqlite-vec。
- 未来 v2+ 家族图谱或分析能力若要落地，必须带实现证据、迁移方案和包体积影响重新打开本 ADR。

## 相关文档

- [docs/ARCHITECTURE.md](../ARCHITECTURE.md) — 当前存储架构与 ADR 索引。
- `internal/infrastructure/database/` — SQLCipher/SQLite 连接器。
- `internal/adapters/repository/` — SQLite 仓库实现。
