# API 文档

> 🌐 [English Version](../../API.md)

> 本文档描述 MedMemo 内部接口契约与 Wails 前后端绑定规范。

---

## API 参考索引

按模块组织的 Wails 绑定详细文档见 [`docs/api/`](../../api/)：

| 模块 | 说明 | 文档 |
|------|------|------|
| Chat | 会话管理与消息流式输出 | [`api/chat.md`](../../api/chat.md) |
| Provider | AI 模型提供商配置与健康检测 | [`api/provider.md`](../../api/provider.md) |
| Auth | 四层鉴权体系 | [`api/auth.md`](../../api/auth.md) |
| System | 设置、更新、免责声明与诊断 | [`api/system.md`](../../api/system.md) |
| Ollama | 本地模型检测与管理 | [`api/ollama.md`](../../api/ollama.md) |
| Memory | 个人记忆审核、搜索和注入开关 | [`api/memory.md`](../../api/memory.md) |
| Embedding | 本地 embedding 模型状态与模型目录辅助方法 | [`api/embedding.md`](../../api/embedding.md) |
| Knowledge | 本地知识文档导入与管理 | [`api/knowledge.md`](../../api/knowledge.md) |
| Events | 后端发出的 Wails Events | [`api/events.md`](../../api/events.md) |

## 环境变量

所有运行时环境变量见 [`docs/environment.md`](../../environment.md)。

---

## 内部接口契约

### LLMClient

```go
type LLMClient interface {
    // Chat 发送非流式对话请求，返回完整回复。
    Chat(ctx context.Context, messages []models.Message) (string, error)

    // StreamChat 发送流式对话请求，通过 callback 逐块推送内容。
    // Provider 返回 usage 元数据时同步返回 token 用量。
    StreamChat(ctx context.Context, messages []models.Message, callback func(chunk string)) (*models.TokenUsage, error)

    // CheckAvailability 检查当前模型是否可用。
    CheckAvailability(ctx context.Context) (bool, string)
}
```

**实现者**：
- `ai.OpenAIAdapter` — OpenAI-compatible API（Kimi / GPT / Qwen / SiliconFlow）
- `ai.LocalAdapter` — Ollama / llama.cpp 本地端点

---

### ProviderStore

```go
type ProviderStore interface {
    Create(ctx context.Context, provider *models.ProviderConfig) error
    Update(ctx context.Context, provider *models.ProviderConfig) error
    Delete(ctx context.Context, id string) error
    Get(ctx context.Context, id string) (*models.ProviderConfig, error)
    List(ctx context.Context) ([]*models.ProviderConfig, error)
}
```

Provider 配置持久化端口。v1.x 实现为 `repository.ProviderRepoSQLite`。

---

### SensitiveDetector

```go
type SensitiveDetector interface {
    Detect(ctx context.Context, text string) ([]models.SensitiveEntity, error)
}
```

**实现者**：
- `onnx.Engine` — Hugot + DistilBERT-ONNX NER 模型（L2 级）
- `desensitizer.RuleEngine` — 正则规则引擎（L1 级）

---

### MemoryRepository

```go
type MemoryRepository interface {
    Save(ctx context.Context, mem *entity.HealthMemory) error
    GetByID(ctx context.Context, id models.MemoryID) (*entity.HealthMemory, error)
    Search(ctx context.Context, query string, limit int) ([]*entity.HealthMemory, error)
    SemanticSearch(ctx context.Context, embedding []float32, topK int) ([]*entity.HealthMemory, error)
    ListByTier(ctx context.Context, tier entity.MemoryTier, limit int) ([]*entity.HealthMemory, error)
    Delete(ctx context.Context, id models.MemoryID) error
}
```

**实现者**：`repository.MemoryRepoSQLite`

---

### FamilyRepository

```go
type FamilyRepository interface {
    SaveMember(ctx context.Context, member *entity.FamilyMember) error
    GetMemberByID(ctx context.Context, id models.MemberID) (*entity.FamilyMember, error)
    ListAllMembers(ctx context.Context) ([]*entity.FamilyMember, error)
    DeleteMember(ctx context.Context, id models.MemberID) error
    FindRelations(ctx context.Context, id models.MemberID) ([]entity.FamilyRelation, error)
    FindByDisease(ctx context.Context, diseaseName string) ([]*entity.FamilyMember, error)
}
```

**实现者**：`repository.FamilyRepoKuzu`（v2+ 规划存根，v1.x 运行时不启用）

---

## Wails 前后端绑定

Wails v2 通过 Go 结构体方法自动生成前端 TypeScript 绑定。

### 绑定示例

**Go 端（`wails_app.go`）**：

```go
type WailsApp struct {
    ctx              context.Context
    chatOrchestrator *usecase.ChatOrchestrator
}

// CreateConversation 创建新会话并返回其 ID，前端通过 window.go.main.WailsApp.CreateConversation 调用。
func (a *WailsApp) CreateConversation() (string, error) {
    // 通过 ConversationRepository 持久化后返回新会话 ID。
    return "", nil
}

// SendMessage 发送消息并触发流式响应。
func (a *WailsApp) SendMessage(convID string, content string) error {
    // 通过 Wails Events 推送流式 chunk 到前端
    return nil
}
```

**前端事件监听**：

```typescript
import { EventsOn } from '@wails/runtime'

EventsOn('chat:stream', (chunk: string) => {
  // 追加到当前对话消息列表
})

EventsOn('compliance:warning', (level: string, reason: string) => {
  // 显示合规警告横幅
})
```

---

## 核心类型

这些共享类型在多个 Wails 绑定中出现。各模块专属的请求/响应类型见对应 API 文档。

### `models.ProviderConfig`

| 字段 | 类型 | 必填 | 说明 |
|------|------|:----:|:----|
| `id` | `string` | ✅ | 唯一 Provider ID |
| `name` | `string` | ✅ | 显示名称 |
| `type` | `string` | ✅ | Provider 类型枚举：`kimi`、`openai`、`qwen`、`ollama`、`local`、`microsoft`、`github`、`claude`、`gemini`、`deepseek` 等 |
| `apiHost` | `string` | ✅ | Provider API 基础地址 |
| `apiKey` | `string` | — | API Key 或访问令牌（静态加密存储） |
| `modelId` | `string` | ✅* | 默认模型 ID；若 `models` 列表包含启用模型则可省略 |
| `models` | `[]ProviderModel` | — | 该 Provider 可用模型列表 |
| `temperature` | `float64` | — | 采样温度，范围 `[0, 2]` |
| `timeoutMs` | `int` | — | 请求超时（毫秒） |
| `maxRetries` | `int` | — | 失败重试次数 |
| `maxTokens` | `int` | — | 每次回复最大 token 数 |
| `group` | `string` | — | UI 分组标签 |
| `enabled` | `bool` | — | 是否启用 |
| `sortOrder` | `int` | — | UI 排序权重 |
| `createdAt` | `int64` | — | 创建时间戳（毫秒） |
| `updatedAt` | `int64` | — | 最后更新时间戳（毫秒） |
| `auth_method` | `string` | — | 认证方式：`api_key`、`cli_token`、`oauth_device`、`service_account` |
| `auth_params` | `AuthParams` | — | 认证方式相关参数 |

### `models.Message`

| 字段 | 类型 | 必填 | 说明 |
|------|------|:----:|:----|
| `role` | `string` | ✅ | `user`、`assistant` 或 `system` |
| `content` | `string` | ✅ | 消息内容 |

### `models.CompressionSettings`

| 字段 | 类型 | 必填 | 说明 |
|------|------|:----:|:----|
| `useModel` | `bool` | — | 是否启用模型驱动的摘要 |
| `providerId` | `string` | — | 摘要模型 Provider ID |
| `modelId` | `string` | — | 摘要模型 ID |
| `anchorCount` | `int` | — | 保留不压缩的锚点消息数 |
| `recentCount` | `int` | — | 保留不压缩的最近消息数 |

### `KnowledgeDocumentDTO`

| 字段 | 类型 | 必填 | 说明 |
|------|------|:----:|:----|
| `id` | `string` | ✅ | 文档 ID |
| `title` | `string` | ✅ | 文档标题 |
| `source` | `string` | ✅ | 来源类型 |
| `citation` | `string` | — | 引用字符串 |
| `url` | `string` | — | 可选来源 URL |
| `language` | `string` | — | 文档语言 |
| `checksum` | `string` | — | 内容校验和 |
| `created_at` | `int64` | — | 创建时间戳（毫秒） |
| `updated_at` | `int64` | — | 最后更新时间戳（毫秒） |

---

## 错误码定义

| 错误码                    | 含义       | HTTP 等效 | 处理建议         |
|------------------------|----------|---------|--------------|
| `ErrNotFound`          | 记录不存在    | 404     | 提示用户创建新记录    |
| `ErrInvalidConfig`     | 配置非法     | 400     | 引导用户检查设置     |
| `ErrDuplicateEntry`    | 重复记录     | 409     | 提示用户合并或替换    |
| `ErrComplianceBlocked` | 内容被合规阻断  | 403     | 显示标准提示语，终止输出 |
| `ErrSensitiveDataLeak` | 敏感数据泄露风险 | 403     | 触发二次脱敏       |

所有错误通过 `fmt.Errorf("...: %w", err)` 包装传递，前端通过 `errors.Is` 链判断根因。

---

*最后更新：2026-07-09*
