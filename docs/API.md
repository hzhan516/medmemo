# API 文档

> 本文档描述 MedMemo 内部接口契约与 Wails 前后端绑定规范。

---

## 内部接口契约

### LLMClient

```go
type LLMClient interface {
    // Chat 发送非流式对话请求，返回完整回复。
    Chat(ctx context.Context, messages []models.Message) (string, error)

    // StreamChat 发送流式对话请求，通过 callback 逐块推送内容。
    StreamChat(ctx context.Context, messages []models.Message, callback func(chunk string)) error

    // CheckAvailability 检查当前模型是否可用。
    CheckAvailability(ctx context.Context) (bool, string)
}
```

**实现者**：
- `ai.OpenAIAdapter` — OpenAI-compatible API（Kimi / GPT / Qwen / SiliconFlow）
- `ai.LocalAdapter` — Ollama / llama.cpp 本地端点

---

### RecordStore

```go
type RecordStore interface {
    Save(ctx context.Context, key string, value []byte) error
    Get(ctx context.Context, key string) ([]byte, error)
    Delete(ctx context.Context, key string) error
}
```

通用键值存储端口，可由 SQLite / DuckDB / 本地文件系统实现。

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

**实现者**：`repository.MemoryRepoDuckDB`

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

**实现者**：`repository.FamilyRepoKuzu`

---

## Wails 前后端绑定

Wails v2 通过 Go 结构体方法自动生成前端 TypeScript 绑定。

### 绑定示例

**Go 端（`internal/infrastructure/wails/app.go` — 待实现）**：

```go
type WailsApp struct {
    chatUC *usecase.ChatOrchestrator
}

// StartConversation 创建新会话，前端通过 window.go.main.App.StartConversation 调用。
func (a *WailsApp) StartConversation(model string) (dto.ConversationDTO, error) {
    conv := entity.NewConversation(models.ProviderType(model))
    return dto.ToConversationDTO(conv), nil
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
