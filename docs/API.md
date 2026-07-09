# API Documentation

> 🌐 [中文版本](./i18n/zh-Hans-CN/API.md)

> This document describes MedMemo's internal interface contracts and Wails frontend-backend binding specifications.

---

## API Reference Index

Detailed Wails binding documentation is organized by module in [`docs/api/`](./api/):

| Module | Description | Document |
|--------|-------------|----------|
| Chat | Conversation management & message streaming | [`api/chat.md`](./api/chat.md) |
| Provider | AI model provider configuration & health | [`api/provider.md`](./api/provider.md) |
| Auth | Four-tier authentication system | [`api/auth.md`](./api/auth.md) |
| System | Settings, updates, disclaimers & diagnostics | [`api/system.md`](./api/system.md) |
| Ollama | Local model detection & management | [`api/ollama.md`](./api/ollama.md) |
| Memory | Personal memory review, search, and injection controls | [`api/memory.md`](./api/memory.md) |
| Embedding | Local embedding model status and model directory helpers | [`api/embedding.md`](./api/embedding.md) |
| Knowledge | Local knowledge document import and management | [`api/knowledge.md`](./api/knowledge.md) |
| Events | Wails Events emitted by the backend | [`api/events.md`](./api/events.md) |

---

## Internal Interface Contracts

### LLMClient

```go
type LLMClient interface {
    // Chat sends a non-streaming conversation request and returns the full reply.
    Chat(ctx context.Context, messages []models.Message) (string, error)

    // StreamChat sends a streaming conversation request, pushing content chunk-by-chunk via callback.
    // It returns token usage when the provider reports usage metadata.
    StreamChat(ctx context.Context, messages []models.Message, callback func(chunk string)) (*models.TokenUsage, error)

    // CheckAvailability checks whether the current model is available.
    CheckAvailability(ctx context.Context) (bool, string)
}
```

**Implementations**:
- `ai.OpenAIAdapter` — OpenAI-compatible API (Kimi / GPT / Qwen / SiliconFlow)
- `ai.LocalAdapter` — Ollama / llama.cpp local endpoints

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

Provider configuration persistence port. The v1.x implementation is `repository.ProviderRepoSQLite`.

---

### SensitiveDetector

```go
type SensitiveDetector interface {
    Detect(ctx context.Context, text string) ([]models.SensitiveEntity, error)
}
```

**Implementations**:
- `onnx.Engine` — Hugot + DistilBERT-ONNX NER model (L2 level)
- `desensitizer.RuleEngine` — Regular expression rule engine (L1 level)

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

**Implementation**: `repository.MemoryRepoSQLite`

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

**Implementation**: `repository.FamilyRepoKuzu` (v2+ planning stub; not active in v1.x runtime)

---

## Wails Frontend-Backend Bindings

Wails v2 automatically generates frontend TypeScript bindings from Go struct methods.

### Binding Example

**Go side (`wails_app.go`)**:

```go
type WailsApp struct {
    chatUC *usecase.ChatOrchestrator
}

// StartConversation creates a new conversation; frontend calls via window.go.main.WailsApp.StartConversation.
func (a *WailsApp) StartConversation(model string) (dto.ConversationDTO, error) {
    conv := entity.NewConversation(models.ProviderType(model))
    return dto.ToConversationDTO(conv), nil
}

// SendMessage sends a message and triggers a streaming response.
func (a *WailsApp) SendMessage(convID string, content string) error {
    // Push streaming chunks to frontend via Wails Events
    return nil
}
```

**Frontend event listeners**:

```typescript
import { EventsOn } from '@wails/runtime'

EventsOn('chat:stream', (chunk: string) => {
  // Append to current conversation message list
})

EventsOn('compliance:warning', (level: string, reason: string) => {
  // Display compliance warning banner
})
```

---

## Error Code Definitions

| Error Code | Meaning | HTTP Equivalent | Handling Suggestion |
|:-----------|:--------|:----------------|:--------------------|
| `ErrNotFound` | Record does not exist | 404 | Prompt user to create a new record |
| `ErrInvalidConfig` | Invalid configuration | 400 | Guide user to check settings |
| `ErrDuplicateEntry` | Duplicate record | 409 | Prompt user to merge or replace |
| `ErrComplianceBlocked` | Content blocked by compliance | 403 | Display standard prompt, terminate output |
| `ErrSensitiveDataLeak` | Sensitive data leak risk | 403 | Trigger secondary de-identification |

All errors are wrapped and propagated via `fmt.Errorf("...: %w", err)`. The frontend uses `errors.Is` to determine the root cause.

---

*Last updated: 2026-07-09*
