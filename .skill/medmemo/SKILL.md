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

> **docs 滞后提示**：顶层 `docs/` 中的部分说明存在已知滞后，阅读时请以本 Skill 及源码为准。

## 入口速查表

| 任务 | 首读文件 |
|------|---------|
| 对话流 / ChatOrchestrator | `internal/application/usecase/chat.go` → `wails_app.go` |
| 记忆检索 / MemoryRetriever | `internal/application/usecase/memory.go` |
| 脱敏 / Deidentify | `internal/application/pipeline/deidentify.go` + `pkg/desensitizer/` |
| 合规拦截 / Compliance | `internal/application/compliance_interceptor.go` + `resources/rules/compliance_rules_v1.json` |
| Provider / 多模型 | `internal/adapters/ai/provider.go` + `web/src/data/provider-templates.json` |
| 会话压缩 / Title | `internal/application/usecase/compression_service.go` + `title_generator.go` |

## 开始工作前

1. 读 [reference-architecture.md](reference-architecture.md) 确认分层与数据流
2. 改代码前对照 [reference-conventions.md](reference-conventions.md) 铁律
3. 推进版本时执行 [reference-release.md](reference-release.md) 检查清单
4. 写/改文档前**必须**触发 `codebase-documenter` Skill，完成后触发 `submission-checker` Skill
5. 写/改/审/重构代码注释时**必须**触发 `code-comment` Skill
6. 实质性变更后更新 [tracker.md](tracker.md)

## 项目速览

| 项 | 值 |
|----|-----|
| 入口 | 仓库根 `main.go` + `wails_app.go`（**非** `cmd/health-assistant/`） |
| DI | `wire.go` 手改 → `wire .` 生成 `wire_gen.go`（**禁止手改**） |
| 前端 | `web/` — React 18 + TS strict + Zustand + HashRouter |
| 主库 | SQLCipher + sqlite-vec 向量检索 |
| 本地 AI | Hugot/ONNX — NER（DistilBERT-ONNX token-classification，路径 `resources/models/distilbert-ner`）+ Embedding（Session 非线程安全，mutex 串行） |
| Tokenizer | `github.com/daulet/tokenizers v1.27.0` |
| Wails 绑定 | `wails_app.go` → 前端 `useWails()` |

> **注意**：`frontend/` 为 Wails 构建产物目录（`dist/`、`wailsjs/` 已在 `.gitignore` 中忽略），请勿手动修改；前端源码位于 `web/`。

> **v2+ 规划组件（DuckDB / Kùzǔ）**
>
> 以下组件为 v2+ 远期规划占位，**不是当前活跃后端**，也**不属于本次迭代范围**：
> - DuckDB（`internal/infrastructure/database/duckdb.go`）— v2+ 分析型存储规划
> - Kùzǔ 家族图谱（`FamilyRepoKuzu`）— v2+ 图数据库规划，未接入 Wire，无实际调用路径
>
> 当前活跃后端为 SQLCipher + sqlite-vec。若需求涉及上述规划组件，必须先与维护者确认技术路线。

## 对话主路径（简图）

```
用户输入 → CheckEmergency → [云端] DeidentifyPipeline（L1 规则 → L2 DistilBERT-ONNX token-classification NER）→ MemoryRetriever
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
| Skill 触发 | 文档变更触发 `codebase-documenter` + `submission-checker`；注释变更触发 `code-comment` |
| 双语文档 | 顶层文档英文撰写，`docs/i18n/zh-Hans-CN/` 同步中文翻译 |
| 前端 | TS strict；Wails 调用走 `useWails()`；`@/` 别名 |

完整约定见 [reference-conventions.md](reference-conventions.md)。

> **权威顺序**：当源码、本 Skill、`AGENTS.md`、`docs/` 之间出现描述冲突时，优先级为 **源码 > Skill > AGENTS.md > docs/**。

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

# Linux 实测构建命令示例（根据实际 ONNX / tokenizer 库路径替换 CGO_LDFLAGS）
GOTOOLCHAIN=auto CGO_LDFLAGS="-L$(pwd)/resources/lib/linux \
  -lonnxruntime -ltokenizers" \
  go build -tags "webkit2_41,ORT" -o /tmp/medmemo .
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
