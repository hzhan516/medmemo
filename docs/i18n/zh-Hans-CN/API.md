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
