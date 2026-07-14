# 架构参考

> 📄 **权威文档**：本文件是面向 AI 速查的精简参考，完整的分层说明与数据流请优先阅读 [`docs/ARCHITECTURE.md`](../../docs/ARCHITECTURE.md)。

## 四层拓扑

```
web/ (React + Zustand)
  useWails() → WailsApp 绑定
  EventsOn() ← chat:stream:*, embedding:migration:*, auth:degraded
       │
wails_app.go — Wails 绑定层（表示层适配，非标准 Clean 层）
       │
internal/application/ — 用例编排
  ChatOrchestrator · MemoryRetriever · ComplianceInterceptor
  DeidentifyPipeline · IntentResolver · LocalAnswerService
  port/ — 接口契约（消费者端声明）
       │
  ┌────┴────┐
domain/   adapters/
entity    ai · auth · detector · repository · updater
policy    │
repo 接口  infrastructure/
          sqlcipher · onnx · secret · config
```

## 包导入白名单（depguard 核心）

| 源目录 | 允许 | 禁止 |
|--------|------|------|
| `internal/domain/*` | 标准库 + `pkg/models/` | 任何 `internal/` 子包 |
| `internal/application/*` | domain + `pkg/models/` | adapters + infrastructure |
| `internal/adapters/*` | domain + 对应 infrastructure | application |
| `internal/infrastructure/*` | 标准库 + 第三方 | domain / application / adapters |

## 记忆分层（v1.1 实现态）

| 层级 | 实体 | 仓库 | 说明 |
|------|------|------|------|
| L1 | `RawDialogue` | `DialogueRepoSQLite` | 原始对话归档 |
| L2 | `HealthFact` | `FactRepoSQLite` | 结构化事实（subject/predicate/object） |
| L3 | `SemanticEmbedding` | `EmbeddingRepoSQLite` + sqlite-vec | 语义向量检索 |
| 遗留 | `HealthMemory` | `MemoryRepoSQLite` | 存在，主检索链路以 Fact+Embedding 为主 |

## 对话数据流

1. `WailsApp.SendMessageStream` 接收请求
2. `EmergencyDetector` 检测用户输入（独立于 AI）
3. `ChatOrchestrator.StreamExecute`：
   - 云端：`DeidentifyPipeline`（L1 规则 → L2 NER）
   - `MemoryRetriever.RetrieveForContext` 注入 system 前缀
   - `IntentResolver` → 高置信时 `LocalAnswerService` 本地短路
   - `LLMClientFactory.CreateClient` → `StreamChat`
   - 流式分句 → `ComplianceChecker` → 还原 P2 占位符
4. `runtime.EventsEmit` 推送前端
5. 异步：`saveMessages`、`extractFactsAsync`（15s 全局限流）

## Wire 注入关键点

| 接口 | 实现 |
|------|------|
| `port.LLMClientFactory` | `ai.llmClientFactory` |
| `usecase.Deidentifier` | `*pipeline.DeidentifyPipeline` |
| `port.NERDetector` | `*detector.ONNXNERDetector` |
| `port.SensitiveDetector` | `*detector.RuleDetector`（仅 L1；完整两级在 Pipeline） |
| `port.EmbeddingService` | `*ai.EmbeddingServiceAdapter` |
| `database.DBConnector` | `*database.SQLCipherConnector` |
| `secret.Store` | `*secret.KeyringStore` |
| `usecase.ComplianceChecker` | `*usecase.RuleComplianceChecker` |

**未接入**：`port.FamilyRepository` / `FamilyRepoKuzu`

## ONNX 启动序列

1. `WailsApp.Startup` → `warmupONNX()` 后台预热
2. `onnxReady` channel 门控
3. `runEmbeddingMigration()` 扫描过期 embedding 并重建
4. Events：`embedding:migration:start|progress|done`

## 数据目录

- 默认：`~/.medmemo/data`
- 覆盖：`MEDMEMO_DATA_DIR` 环境变量
- Embedding 模型：首次启动从 bundle 复制到 `{DataDir}/models/{EmbeddingModelName}/`

## 关键依赖版本

以 `go.mod`、`wails.json`、`web/package.json` 为准（当前以 `wails.json` 的 `info.productVersion` 为准）：

> 仅供参考，实际以 `go.mod` / `wails.json` / `web/package.json` 为准。

| 组件 | 版本 |
|------|------|
| Go | 1.26.4 |
| Wails v2 | 2.12.0 |
| Google Wire | 0.7.0 |
| ONNX Runtime | 1.26.0 |
| Hugot | 0.7.4 |
| daulet/tokenizers | 1.27.0 |
| modernc.org/sqlite | 1.53.0 |
| sqlite-vec | 0.3.0 |
| React | 18.2.0 |
| TypeScript | 5.9.3 |

## v2+ 规划组件（DuckDB 分析层 / Kùzǔ 家族图谱）

以下组件为 v2+ 规划组件，不是当前活跃后端：

- **DuckDB 分析层**：`DuckDBConnector.Migrate` → `not implemented`
- **Kùzǔ 家族图谱**：`FamilyRepoKuzu` 全 stub，Issue#016
