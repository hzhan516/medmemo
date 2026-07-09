# Chat API

> 🌐 [中文版本](../i18n/zh-Hans-CN/api/chat.md)

This document describes Wails bindings related to conversation management and message streaming.

---

## Methods

### `SendMessage(req SendMessageRequest) (*SendMessageResponse, error)`

Sends a non-streaming chat message. The backend orchestrates the full pipeline: de-identification → memory retrieval → context assembly → LLM invocation → output restoration → compliance check.

**Request:**

```go
type SendMessageRequest struct {
    ConversationID string           `json:"conversation_id"`
    Messages       []models.Message `json:"messages"`
    Model          string           `json:"model"`
    ProviderID     string           `json:"provider_id"`
}
```

**Response:**

```go
type SendMessageResponse struct {
    Reply      string   `json:"reply"`
    Confidence float64  `json:"confidence"`
    Warnings   []string `json:"warnings"`
}
```

**Errors:**
- `ErrInvalidConfig` — provider or model not configured
- `ErrComplianceBlocked` — content blocked by compliance engine

---

### `SendMessageStream(req SendMessageRequest) error`

Sends a streaming chat message. The response is pushed to the frontend via Wails Events (`chat:stream_chunk`).

**Request:** same as `SendMessage`.

**Events emitted:**
- `chat:stream_chunk` — see [Events API](events.md)
- `chat:stream:compliance` — compliance warning/notice
- `chat:title:generated` — auto-generated conversation title (after first message)

**Errors:**
- Returns error if the stream cannot be started; individual chunk errors are delivered via event.

---

### `StopGeneration()`

Aborts the currently active streaming generation. No-op if no stream is in progress.

---

### `CreateConversation() (string, error)`

Creates a new conversation and returns its unique ID. The conversation is persisted via `ConversationRepository`.

**Returns:** `conversation_id` string.

---

### `GetConversations() ([]ConversationSummary, error)`

Returns the list of all non-deleted conversations, sorted by `updated_at` descending.

```go
type ConversationSummary struct {
    ID        string `json:"id"`
    Title     string `json:"title"`
    UpdatedAt string `json:"updated_at"`
    DeletedAt string `json:"deleted_at"` // empty if not soft-deleted
}
```

---

### `GetConversationMessages(convID string) ([]MessageResponse, error)`

Retrieves all messages for a given conversation.

```go
type MessageResponse struct {
    ID        string `json:"id"`
    Role      string `json:"role"`      // "user" | "assistant" | "system"
    Content   string `json:"content"`
    Timestamp string `json:"timestamp"` // Unix timestamp (ms)
}
```

---

### `DeleteConversation(convID string) error`

Soft-deletes a conversation. The record remains recoverable for 30 days.

---

### `RestoreConversation(convID string) error`

Restores a soft-deleted conversation.

---

### `HardDeleteConversation(convID string) error`

Permanently removes a conversation and all its messages.

---

### `GenerateTitle(convID string, userMessage string)`

Asynchronously generates a conversation title based on the first user message. The result is delivered via `chat:title:generated` event. This method does not block the caller.

---

### `CheckEmergency(text string) (*EmergencyResult, error)`

Performs local emergency symptom detection (AGENTS.md §7.3). This is independent of the AI reply flow and executes in <5ms.

```go
type EmergencyResult struct {
    Level   string `json:"level"`   // "A", "B", "none"
    Message string `json:"message"`
    Action  string `json:"action"`
}
```

| Level | Trigger | Frontend Behavior |
|-------|---------|-------------------|
| `A` | Critical symptoms (chest pain + dyspnea, unconsciousness, etc.) | Full-screen red overlay with 120/emergency options |
| `B` | Urgent symptoms (persistent high fever >3d, severe abdominal pain, etc.) | Red warning banner above input; require acknowledgement |
| `none` | No emergency keywords matched | No UI interruption |
