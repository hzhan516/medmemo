# 核心模块参考

> 📄 **权威文档**：本文件是面向 AI 速查的精简参考，完整的接口契约与 Wails 绑定说明请优先阅读 [`docs/API.md`](../../docs/API.md) 及 [`docs/api/`](../../docs/api/)。

## 目录职责

| 路径 | 职责 |
|------|------|
| `main.go` | 入口、embed 前端、`InitializeApp()`、优雅关闭 |
| `wails_app_*.go` | 全部 Wails 绑定方法（按域拆分到多个文件） |
| `wire.go` / `wire_gen.go` | Wire DI 蓝图 / 生成代码 |
| `internal/application/usecase/chat.go` | `ChatOrchestrator` 对话编排 |
| `internal/application/usecase/memory.go` | `MemoryRetriever` 记忆检索 |
| `internal/application/usecase/intent_resolver.go` | 意图解析 + 本地短路门控 |
| `internal/application/usecase/local_answer.go` | 高置信事实模板回答 |
| `internal/application/usecase/local_answer_config.go` | 本地回答业务配置（模板 / 人称 / subject） |
| `internal/application/usecase/embedding_migration.go` | Embedding 版本迁移 |
| `internal/application/usecase/compression_service.go` | 会话上下文压缩 |
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

```go
// internal/application/port/embedding.go
type EmbeddingService interface {
    // Embed 批量生成文本嵌入，输出已 L2 归一化的 384 维向量。
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    // EmbedSingle 单条文本嵌入生成，是 Embed 的便捷包装。
    EmbedSingle(ctx context.Context, text string) ([]float32, error)
    // ModelVersion 返回当前使用的复合版本标识。
    ModelVersion() string
    // IsAvailable 返回 ONNX 嵌入推理是否可用。
    IsAvailable() bool
}
```

```go
// internal/application/usecase/chat.go
type ComplianceChecker interface {
    Check(ctx context.Context, text string) (*ComplianceResult, error)
}

// ComplianceResult 合规检查结果。
type ComplianceResult struct {
    Blocked       bool
    Level         string
    Reason        string
    SafeText      string
    Warning       string   // L2 警告文案
    Notice        string   // L3 提示文案
    MatchedRule   string   // 命中的规则 ID
    ReplacedTerms []string // inline 替换中被替换的用词规则 ID 列表
}

// *usecase.RuleComplianceChecker 实现 ComplianceChecker 接口。
```

```go
// internal/application/usecase/compression_service.go
// CompressionService 为结构体，其公开方法签名如下：
type CompressionService interface {
    Compress(ctx context.Context, conversationID models.ConversationID, providerID, modelID string, cfg CompressionConfig) (CompressionResult, error)
    CompressMessages(ctx context.Context, history []models.Message, providerID, modelID string, cfg CompressionConfig) (CompressionResult, error)
    TestModelAvailability(ctx context.Context, providerID, modelID string) (bool, error)
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

**对话**：`SendMessage`, `SendMessageStream`, `StopGeneration`, `GetConversationMessages`, `GenerateTitle`

**会话**：`CreateConversation`, `GetConversations`, `GetDeletedConversations`, `DeleteConversation`, `RestoreConversation`, `HardDeleteConversation`, `SetDataRetentionDays`, `SetDesensitizationLevel`, `GetDesensitizationLevel`

**合规 / 紧急**：`CheckEmergency`, `GetDisclaimerStatus`, `AcceptDisclaimer`, `DeclineDisclaimer`, `ShowEmergencyDialog`, `ReportComplianceFeedback`

**Context / 压缩**：`ResolveMaxContextLength`, `EstimateContextUsage`, `CompressSession`, `GetCompressionSettings`, `SetCompressionSettings`, `TestCompressionModel`

**认证**：`SaveAPIKey`, `HasAPIKey`, `TestAPIKey`, `RefreshToken`, `EnableAutoRefresh`, `DisableAutoRefresh`, `DetectCLIToken`, `BuildCLIProvider`, `StartOAuthDeviceFlow`, `CancelOAuthDeviceFlow`, `GetOAuthDeviceFlowStatus`, `GetOAuthDeviceFlowProviders`, `ParseServiceAccountJSON`, `DetectAuthMethods`

**Provider**：`GetModels`, `CreateProvider`, `UpdateProvider`, `DeleteProvider`, `ListProviders`, `CheckProviderHealth`, `GetProviderHealthStatus`

**Embedding**：`GetEmbeddingStatus`, `GetEmbeddingModelDirPath`, `OpenEmbeddingModelDir`

**Ollama**：`DetectOllama`, `StartOllamaServer`, `PullOllamaModel`, `EnsureOllamaAndSmolLM2`, `CreateOllamaProvider`

**记忆**：`GetMemories`, `GetMemoryByID`, `DeleteMemory`, `SearchMemories`, `GetPendingReviews`, `ApproveFact`, `RejectFact`, `GetMemoryStats`, `GetMemoriesBySession`, `SetMemoryInjectionEnabled`, `SetSessionMemoryInjection`

**知识库**：`SelectKnowledgeFile`, `ImportKnowledgeFile`, `ListKnowledgeDocuments`, `DeleteKnowledgeDocument`, `GetKnowledgeImportJob`

**更新**：`CheckUpdate`, `DownloadUpdate`, `ApplyUpdate`, `GetUpdateSettings`, `SetUpdateSettings`, `SkipUpdateVersion`, `OpenDownloadURL`

**系统 / 反馈**：`GetVersion`, `GetVersionInfo`, `CollectSystemInfo`, `OpenGitHubIssue`, `RecordAnswerFeedback`, `GetVersionNotes`

**生命周期**：`Startup`

## 前端状态要点

`chatStore.ts`：
- `messagesMap` — 按会话 ID 隔离消息
- `streamingIds` — 多会话并发流式
- 合规字段：`warnings`, `replacedTerms`, `confidence`, `complianceFeedback`

## 领域哨兵错误

`internal/domain/entity/errors.go`：`ErrNotFound`, `ErrComplianceBlocked`, `ErrFactNotFound`, `ErrEmbeddingNotFound`, `ErrInvalidFact`

适配器层用 `errors.Is` 映射，禁止向上传递裸错误。
