# 文档治理

> 🌐 [English Version](../../docs-governance.md)

本文档定义 MedMemo 如何保持文档的准确性、一致性与可发现性。它记录每类信息的**单一真源**、防止漂移的审计检查，以及关闭文档缺口的工作流程。

---

## 1. 单一真源

为避免「一次修改、多处陈旧副本」的问题，每类主要信息只应有一处权威位置；其他位置必须通过链接引用，而非复制内容。

| 信息类别 | 单一真源 | 镜像 / 指针 |
|----------|----------|-------------|
| 产品版本 | `wails.json` → `info.productVersion` | `web/package.json`、`web/package-lock.json`、`internal/domain/entity/changelog/zh-Hans.json` |
| 各层 README（domain/application/adapters/infrastructure） | `internal/*/README.md` | `docs/internal/*/README.md`（仅指针） |
| API 类型 / 接口 / 绑定参考 | `pkg/models/*.go`、`internal/application/port/*.go`、`wails_app_*.go` | `docs/api/_generated/*.md`（自动生成） |
| 架构总览 | `docs/ARCHITECTURE.md` | `.skill/medmemo/reference-architecture.md`（链接指向它） |
| API 模块指南 | `docs/API.md` 与 `docs/api/*.md` | `.skill/medmemo/reference-modules.md`（链接指向它） |
| 第三方许可证 | `THIRD_PARTY_LICENSES.md`（自动生成） | 由 `go.mod` / `web/package.json` 生成 |
| 合规 / 隐私红线 | `AGENTS.md` 合规章节 | `docs/COMPLIANCE.md`（扩展参考） |

规则：
- 编辑权威文件；禁止手工修改生成物。
- 若新增重复类别，要么合并，要么加入上表。
- 指针文档不得包含可能与权威源漂移的实质性内容。

---

## 2. 文档检查

所有检查在每次 push/PR 时运行（`.github/workflows/ci.yml`），并由每月定时审计（`.github/workflows/docs-audit.yml`）复用。

| 检查项 | 工具 / 脚本 | 防止的问题 |
|--------|-------------|------------|
| Markdown 断链 | `lychee` 经 `scripts/check-doc-links.sh` | 内部或外部死链 |
| 缺少中文镜像 | `scripts/check-doc-mirrors.js` | 英文文档缺少 `docs/i18n/zh-Hans-CN/` 对应版本 |
| 术语偏差 | `scripts/check-terminology.js` | 翻译或项目术语不一致 |
| 版本漂移 | `scripts/check-version-consistency.js` | `wails.json` / `web/package.json` / 文档不同步 |
| API 文档漂移 | `make docs` + `git diff --exit-code docs/api/_generated/` | 源码已改但未重生成 API 文档 |
| 许可证漂移 | `make licenses` + `git diff --exit-code THIRD_PARTY_LICENSES.md` | 依赖变更后未更新许可证清单 |

本地等价命令：

```bash
make docs-check   # 链接、镜像、术语、版本
make docs         # 重生成 API 文档
make licenses     # 重生成第三方许可证
```

---

## 3. 健康度指标

定时审计会产出 `docs-health-report.md` artifact，包含以下可趋势化跟踪的指标：

| 指标 | 来源 | 目标 |
|------|------|------|
| 断链数 | `scripts/check-doc-links.sh` | 0 |
| 缺少中文镜像数 | `scripts/check-doc-mirrors.js` | 0 |
| 术语偏差数 | `scripts/check-terminology.js` | 0 |
| 版本漂移数 | `scripts/check-version-consistency.js` | 0 |
| API 文档漂移 | `git diff --exit-code docs/api/_generated/` | 无差异 |
| 许可证清单漂移 | `git diff --exit-code THIRD_PARTY_LICENSES.md` | 无差异 |

任何非零指标或漂移均视为文档健康度失败，应跟踪至闭环。

---

## 4. 问题严重级别（P0–P3）

文档缺口按影响与紧急程度分级。在创建或更新 `medmemo/开发日志/issues.md` 时使用这些标签。

| 级别 | 定义 | 示例 | 响应时间 |
|------|------|------|----------|
| **P0** | 阻塞发布或面向用户的错误指引 | 构建命令错误、安装步骤不可用、许可证遗漏 | 立即修复 |
| **P1** | 可能误导贡献者的事实不一致 | 依赖版本过期、列出已不存在的目录 | 当前迭代修复 |
| **P2** | 不完整或难以维护的文档 | 缺少 API 字段表、内容重复 | 下个版本修复 |
| **P3** | 流程 / 治理 / 自动化债务 | 缺少定时审计、无真源表 | 经批准后按计划实施 |

---

## 5. 闭环流程

1. **开启**：在 `medmemo/开发日志/issues.md` 中新增一行，使用全局递增 issue 编号，标明严重级别与描述。
2. **修复**：编辑权威源；重生成任何派生产物；运行 `make docs-check`、`make docs` 和 `make licenses`。
3. **验证**：确认 CI 通过；对于定时审计发现的问题，确认最新审计 artifact 中该指标已达标。
4. **关闭**：更新 `medmemo/开发日志/issues.md` 中对应行：将 **完成** 设为 `✅`，**状态** 设为 `closed`。

---

## 6. 相关文档

- [`AGENTS.md`](../../../AGENTS.md) — 项目约定与红线
- [`docs/ARCHITECTURE.md`](../../ARCHITECTURE.md) — 系统架构
- [`docs/API.md`](../../API.md) — API 参考入口
- [`.github/workflows/ci.yml`](../../../.github/workflows/ci.yml) — PR 级文档检查
- [`.github/workflows/docs-audit.yml`](../../../.github/workflows/docs-audit.yml) — 每月定时审计

---

*最后更新：2026-07-14*
