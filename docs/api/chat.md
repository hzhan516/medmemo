# Chat API

> 🌐 [中文版本](../i18n/zh-Hans-CN/api/chat.md)

This document describes Wails bindings related to conversation management and message streaming.

---

## Methods

### `SendMessage(req SendMessageRequest) (*SendMessageResponse, error)`

Sends a non-streaming chat message. The backend orchestrates the full pipeline: de-identification → memory retrieval → context assembly → LLM invocation → output restoration → compliance check.

#### `SendMessageRequest`

| Field | Type | Required | Description |
|-------|------|:--------:|:------------|
| `conversation_id` | `string` | ✅ | Target conversation ID |
| `messages` | `[]models.Message` | ✅ | Current message context, including the new user message as the last element |
| `model` | `string` | ✅ | Model ID to use |
| `provider_id` | `string` | ✅ | Provider ID to use |
| `ai_message_id` | `string` | — | Optional pre-generated ID for the assistant reply (used for feedback correlation) |
| `force_send` | `bool` | — | `true` if the user confirmed sending under strict de-identification degradation |

#### `SendMessageResponse`

| Field | Type | Description |
|-------|------|:------------|
| `reply` | `string` | Assistant reply content |
| `confidence_result` | `map[string]interface{}` | Confidence scoring result (overall score, level, explanation, ...) |
| `warnings` | `[]string` | Compliance / risk warnings |

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
- `chat:stream:confidence` — confidence score and token usage after stream ends
- `chat:stream:replace` — replaces the streamed text when compliance restores placeholders
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

#### `ConversationSummary`

| Field | Type | Description |
|-------|------|:------------|
| `id` | `string` | Conversation ID |
| `title` | `string` | Conversation title |
| `updated_at` | `string` | Last update timestamp (ms) |
| `deleted_at` | `string` | Soft-delete timestamp (ms); empty if not deleted |

---

### `GetConversationMessages(convID string) ([]MessageResponse, error)`

Retrieves all messages for a given conversation.

#### `MessageResponse`

| Field | Type | Description |
|-------|------|:------------|
| `id` | `string` | Message ID |
| `role` | `string` | `user` \| `assistant` \| `system` |
| `content` | `string` | Message content |
| `timestamp` | `string` | Unix timestamp (ms) |
| `prompt_tokens` | `int` | Prompt tokens consumed |
| `completion_tokens` | `int` | Completion tokens consumed |
| `total_tokens` | `int` | Total tokens consumed |
| `confidence` | `map[string]interface{}` | Confidence result, omitted if absent |

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

### `GetDeletedConversations() ([]ConversationSummary, error)`

Returns soft-deleted conversations that are still within the retention window.

---

### `GenerateTitle(convID string, userMessage string)`

Generates a conversation title based on the first user message using local rules. The result is delivered via `chat:title:generated` event. This method does not block the caller.

---

## Context and Compression

### `ResolveMaxContextLength(providerID, modelID string) (int, error)`

Resolves the maximum context length for a provider/model pair.

### `EstimateContextUsage(req EstimateContextUsageRequest) (*ContextUsageResponse, error)`

Estimates token usage for the current messages and assembled prompt.

#### `EstimateContextUsageRequest`

| Field | Type | Required | Description |
|-------|------|:--------:|:------------|
| `conversationId` | `string` | ✅ | Conversation ID |
| `messages` | `[]models.Message` | ✅ | Messages to estimate |
| `providerId` | `string` | ✅ | Provider ID |
| `modelId` | `string` | ✅ | Model ID |
| `assembledPrompt` | `[]models.Message` | — | Optional pre-assembled prompt; if omitted, the backend assembles it |

#### `ContextUsageResponse`

| Field | Type | Description |
|-------|------|:------------|
| `usedTokens` | `int` | Estimated used tokens |
| `maxTokens` | `int` | Maximum context tokens for the model |
| `ratio` | `float64` | `usedTokens / maxTokens` |
| `approximate` | `bool` | Whether the estimate is approximate |

---

### `CompressSession(req CompressSessionRequest) error`

Compresses a conversation session and emits `context:usage_refresh` when done.

#### `CompressSessionRequest`

| Field | Type | Required | Description |
|-------|------|:--------:|:------------|
| `conversationId` | `string` | ✅ | Conversation ID |
| `providerId` | `string` | ✅ | Provider ID for model-based summarization |
| `modelId` | `string` | ✅ | Model ID for summarization |
| `strategy` | `string` | — | `drop_earliest_n`, `summarize_and_replace`, or `llm_self_summarize` |
| `anchorCount` | `int` | — | Number of anchor messages to keep |
| `recentCount` | `int` | — | Number of recent messages to keep |

---

### `GetCompressionSettings() models.CompressionSettings`

Returns persisted compression settings.

### `SetCompressionSettings(s models.CompressionSettings) error`

Persists compression settings.

### `TestCompressionModel(providerID, modelID string) (bool, error)`

Checks whether the selected compression model is available.

---

## Emergency and Privacy Settings

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

---

### `ShowEmergencyDialog(title, message string)`

Shows a native warning dialog for emergency-symptom flows.

### `SetDataRetentionDays(days int) error`

Persists the soft-delete retention window in days.

### `SetDesensitizationLevel(level string) error`

Persists the outbound de-identification level (`standard`, `strict`, or `off`).

### `GetDesensitizationLevel() string`

Returns the current outbound de-identification level.

---

*Last updated: 2026-07-09*
