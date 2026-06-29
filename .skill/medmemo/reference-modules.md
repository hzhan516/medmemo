# 核心模块参考

## 目录职责

| 路径 | 职责 |
|------|------|
| `main.go` | 入口、embed 前端、`InitializeApp()`、优雅关闭 |
| `wails_app.go` | 全部 Wails 绑定、流式/ONNX/迁移/OAuth |
| `wire.go` / `wire_gen.go` | Wire DI 蓝图 / 生成代码 |
| `internal/application/usecase/chat.go` | `ChatOrchestrator` 对话编排 |
| `internal/application/usecase/memory.go` | `MemoryRetriever` 记忆检索 |
| `internal/application/usecase/intent_resolver.go` | 意图解析 + 本地短路门控 |
| `internal/application/usecase/local_answer.go` | 高置信事实模板回答 |
| `internal/application/usecase/local_answer_config.go` | 本地回答业务配置（模板 / 人称 / subject） |
| `internal/application/usecase/embedding_migration.go` | Embedding 版本迁移 |
| `internal/application/pipeline/deidentify.go` | 二级脱敏流水线（L1 规则 → L2 NER） |
| `internal/application/compliance_interceptor.go` | L1~L4 合规拦截 |
| `internal/application/emergency_detector.go` | A/B 级紧急症状 |
| `internal/adapters/ai/` | OpenAI/Local Adapter、ProviderFactory、Embedding |
| `internal/adapters/auth/` | OAuth Device Flow、CLI Token、Token Refresh |
| `internal/adapters/repository/` | 全部 SQLite 仓库 |
| `internal/infrastructure/onnx/` | Hugot Engine、NER Worker |
| `internal/infrastructure/database/sqlcipher.go` | 加密库 + schema 迁移 |
| `pkg/models/` | 跨层 DTO |
| `pkg/desensitizer/` | 独立 L1 规则引擎 |
| `web/src/stores/chatStore.ts` | 会话隔离、流式状态、合规字段 |
| `web/src/hooks/useWails.ts` | 后端绑定统一封装 |
| `resources/rules/compliance_rules_v1.json` | 合规规则库 |

## 关键接口

```go
// internal/application/port/llm.go
type LLMClient interface {
    Chat(ctx context.Context, messages []models.Message) (string, error)
    StreamChat(ctx context.Context, messages []models.Message, callback func(chunk string)) (*models.TokenUsage, error)
    CheckAvailability(ctx context.Context) (bool, string)
}

type LLMClientFactory interface {
    CreateClient(providerConfig *models.ProviderConfig) (LLMClient, error)
}
```

```go
// internal/application/usecase/chat.go
type Deidentifier interface {
    Execute(ctx context.Context, raw string) (models.DeidentifyResult, error)
}

type MemoryQuerier interface {
    RetrieveForContext(ctx context.Context, query, sessionID string, limit int) ([]*entity.HealthMemory, error)
}
```

## Provider 路由

| ProviderType | Adapter |
|--------------|---------|
| `ProviderOllama` | `LocalAdapter` |
| `ProviderLocal` | `OpenAIAdapter`（llama.cpp 兼容端点） |
| 云端各厂商 | `OpenAIAdapter` |

预置模板：`web/src/data/provider-templates.json`（21+ 厂商）

## Wails 绑定方法（按域）

**对话**：`SendMessage`, `SendMessageStream`, `StopGeneration`, `GetConversationMessages`, `CreateConversation`, `GenerateTitle`

**会话**：`GetConversations`, `DeleteConversation`, `RestoreConversation`, `HardDeleteConversation`, `SetDataRetentionDays`

**合规**：`CheckEmergency`, `GetDisclaimerStatus`, `AcceptDisclaimer`, `ReportComplianceFeedback`

**Provider**：`CreateProvider`, `UpdateProvider`, `DeleteProvider`, `ListProviders`, `CheckProviderHealth`

**认证**：`SaveAPIKey`, `HasAPIKey`, `TestAPIKey`, `StartOAuthDeviceFlow`, `DetectCLIToken`, `RefreshToken`

**记忆**：`ListFacts`, `ApproveFact`, `RejectFact`, `DeleteFact`, `GetMemorySettings`, `SetMemoryEnabled`

**更新**：`CheckUpdate`, `DownloadUpdate`, `ApplyUpdate`, `GetVersionNotes`, `GetVersion`

**系统**：`CollectSystemInfo`, `OpenGitHubIssue`

## 前端状态要点

`chatStore.ts`：
- `messagesMap` — 按会话 ID 隔离消息
- `streamingIds` — 多会话并发流式
- 合规字段：`warnings`, `replacedTerms`, `confidence`, `complianceFeedback`

## 领域哨兵错误

`internal/domain/entity/errors.go`：`ErrNotFound`, `ErrComplianceBlocked`, `ErrFactNotFound`, `ErrEmbeddingNotFound`, `ErrInvalidFact`

适配器层用 `errors.Is` 映射，禁止向上传递裸错误。
