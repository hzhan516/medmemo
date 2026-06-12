---
name: medmemo
description: >-
  用于在 MedMemo 仓库内进行开发、重构、审查、Debug、推进版本号，
  或处理 Wails/Wire 架构、对话-记忆-合规流水线、SQLCipher/sqlite-vec 存储、
  ChatOrchestrator、Provider、脱敏、embedding 迁移等相关任务时。
---

# MedMemo Project Skill

开源桌面健康咨询信息工具（**非医疗器械**）。本地优先、多模型对话、分层记忆、四级合规拦截。

**当前版本**：以 `wails.json` → `info.productVersion` 为准（勿硬编码版本号）。

## 开始工作前

1. 读 [reference-architecture.md](reference-architecture.md) 确认分层与数据流
2. 改代码前对照 [reference-conventions.md](reference-conventions.md) 铁律
3. 推进版本时执行 [reference-release.md](reference-release.md) 检查清单
4. 实质性变更后更新 [tracker.md](tracker.md)

## 项目速览

| 项 | 值 |
|----|-----|
| 入口 | 仓库根 `main.go` + `wails_app.go`（**非** `cmd/health-assistant/`） |
| DI | `wire.go` 手改 → `wire .` 生成 `wire_gen.go`（**禁止手改**） |
| 前端 | `web/` — React 18 + TS strict + Zustand + HashRouter |
| 主库 | SQLCipher + sqlite-vec 向量检索 |
| 本地 AI | Hugot/ONNX — NER 脱敏 + Embedding（Session 非线程安全，mutex 串行） |
| Wails 绑定 | `wails_app.go` → 前端 `useWails()` |

> **未落地存根（严禁自行实现）**
>
> 以下模块仅保留接口或占位实现，**不要假设可用，也不要主动补全实现**：
> - DuckDB（`internal/infrastructure/database/duckdb.go`）— 已冻结，项目已降级至 SQLite
> - Kùzǔ 家族图谱（`FamilyRepoKuzu`）— 未接入 Wire，无实际调用路径
>
> 若需求涉及上述模块，必须先与维护者确认技术路线。

## 对话主路径（简图）

```
用户输入 → CheckEmergency → [云端] DeidentifyPipeline → MemoryRetriever
         → IntentResolver / LocalAnswerService（高置信本地短路）
         → LLM StreamChat → ComplianceInterceptor 分句检测 → Events 推送前端
         → saveMessages + extractFactsAsync
```

本地模型（Ollama/Local）**跳过脱敏**。详见 [reference-modules.md](reference-modules.md)。

## 常见任务工作流

### 新增 Wire 依赖

```
- [ ] 在对应层 ProviderSet 添加构造函数
- [ ] 修改 wire.go 添加 wire.Bind（如需要）
- [ ] 运行 wire .（仓库根目录）
- [ ] 确认 wire_gen.go 已更新，未手改
```

### 修改 Provider / 多模型

- 路由：`internal/adapters/ai/provider.go`（Ollama→LocalAdapter，Local→OpenAIAdapter，云端→OpenAIAdapter）
- 同步 `web/src/data/provider-templates.json`
- 运行 `scripts/validate-provider-templates.js`

### 修改合规 / 脱敏 / 紧急症状

- 合规规则：`resources/rules/compliance_rules_v1.json`
- 拦截器：`internal/application/compliance_interceptor.go`
- 脱敏流水线：`internal/application/pipeline/deidentify.go`
- 紧急关键词：`internal/application/emergency_detector.go`
- UI 文案须过合规红线（见 conventions）

### 修改记忆 / Embedding

- 事实层：`fact_repo` + `HealthFact`
- 向量层：`embedding_repo` + `SemanticEmbedding`
- 检索：`usecase/MemoryRetriever`
- 版本迁移：`EmbeddingMigrator`，ONNX 预热后门控（`wails_app.go`）

## 强制规则（摘要）

| 规则 | 要求 |
|------|------|
| domain 层 | 仅标准库 + `pkg/models/` |
| 错误处理 | `fmt.Errorf("...: %w", err)`，禁止裸 `return err` |
| Wire | 只改 `wire.go`，禁止手改 `wire_gen.go` |
| TODO | `// TODO(作者): 描述 [Issue#N]`，同步 `medmemo/开发日志/issues.md` |
| 合规红线 | 禁止确诊/处方/治疗建议/"AI医生"；紧急症状必须提醒 |
| 前端 | TS strict；Wails 调用走 `useWails()`；`@/` 别名 |

完整约定见 [reference-conventions.md](reference-conventions.md)。

## 版本发布

推进版本号时**必须**同步（常规 Minor/Patch）：

```
□ wails.json → info.productVersion（唯一事实来源）
□ web/package.json → version（与上完全一致，无 v 前缀）
□ web/package-lock.json → 执行 `cd web && npm install` 刷新锁定文件
□ internal/domain/entity/changelog/zh-Hans.json → 数组末尾新增条目
□ medmemo/开发日志/ → 对应版本日志
□ tracker.md → 追加滚动记录
```

大版本（如 v1.1.x → v1.2.0）额外要求见 [reference-release.md](reference-release.md)。

## 构建命令

```bash
wails dev          # 开发
wails build        # 构建（前端 embed 到 main.go）
wire .             # 依赖注入生成
go test ./...      # 后端测试
cd web && npm test # 前端测试
```

## 滚动更新

每次实质性代码修改、架构理解深化或版本推进后，在 [tracker.md](tracker.md) **顶部**追加记录（格式见该文件）。

## 延伸阅读

| 文件 | 内容 |
|------|------|
| [reference-architecture.md](reference-architecture.md) | 四层拓扑、记忆分层、DI 注入点 |
| [reference-modules.md](reference-modules.md) | 关键文件、接口、Wails 方法表 |
| [reference-conventions.md](reference-conventions.md) | 命名、并发、前端、合规红线 |
| [reference-release.md](reference-release.md) | 版本发布强制指南（原文） |
| [tracker.md](tracker.md) | 滚动更新追踪器 |
