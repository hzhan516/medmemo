# Events API

> 🌐 [中文版本](../i18n/zh-Hans-CN/api/events.md)

This document describes Wails Events emitted by the backend and consumed by the frontend via `EventsOn`.

---

## `chat:stream_chunk`

Emitted during a streaming LLM response.

```typescript
type StreamChunk = {
  type: 'start' | 'content' | 'done' | 'error'
  payload: string
  metadata?: {
    conversation_id?: string
    model?: string
    provider_id?: string
    latency_ms?: number
    token_count?: number
    prompt_tokens?: number
    completion_tokens?: number
  }
}
```

| `type` | Meaning |
|--------|---------|
| `start` | Stream initialization; payload is empty |
| `content` | New token chunk; append to the last assistant message |
| `done` | Stream complete; `metadata` contains token usage |
| `error` | Stream failed; `payload` contains the error message |

**Frontend handling:**
```typescript
import { EventsOn } from '@wails/runtime/runtime'

EventsOn('chat:stream_chunk', (chunk: StreamChunk) => {
  if (chunk.type === 'content') {
    appendToLastMessageForConversation(chunk.metadata.conversation_id, chunk.payload)
  }
})
```

---

## `chat:stream:compliance`

Emitted when the compliance engine detects a risk level ≥ L2 during streaming.

```typescript
type CompliancePayload = {
  conversation_id?: string
  level: string        // "L2_WARNING" | "L3_NOTICE"
  warning: string
  notice: string
  replacedTerms?: string[]
  matchedRule?: string
}
```

| `level` | Frontend Behavior |
|---------|-------------------|
| `L2_WARNING` | Orange warning banner; allow continuing |
| `L3_NOTICE` | Blue disclaimer banner appended to the message |

---

## `chat:title:generated`

Emitted after the first user message when the title generator completes.

```typescript
type TitlePayload = {
  conv_id: string
  title: string
}
```

---

## `chat:stream:replace`

Emitted when the compliance engine replaces the entire last assistant message (e.g., L1 block or de-identification restoration).

```typescript
type ReplacePayload = {
  conversation_id: string
  content: string   // replacement text (standard prompt or restored content)
}
```

---

## `chat:stream:confidence`

Emitted after a streaming response completes, carrying the confidence score and token usage.

```typescript
type ConfidencePayload = {
  conversation_id: string
  confidence: {
    overall_score: number
    level: string
    explanation: string
    suggestion: string
    missing_info: string[]
    citations: string[]
    breakdown?: Record<string, number>
  }
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  truncated: boolean
}
```

---

## `chat:deid:confirm`

Emitted in **Strict** de-identification mode when sensitive content cannot be safely replaced and requires user confirmation before sending.

```typescript
type DeidConfirmPayload = {
  conversation_id: string
  preview: string[]  // degraded (safe) message preview
}
```

---

## `chat:save_error`

Emitted when asynchronous message persistence fails.

```typescript
type SaveErrorPayload = {
  type: string            // e.g. "user_message", "raw_dialogue", "update_timestamp"
  conversation_id: string
  error: string
  timestamp: number       // ms
}
```

---

## `update:progress`

Emitted during update package download.

```typescript
type UpdateProgress = {
  percent: number
  downloaded: number
  total: number
  speed: string
}
```

---

## Event Routing Notes

- All chat events carry `conversation_id` in metadata so the frontend can route them to the correct session even if the user has switched conversations.
- The frontend must not assume `currentConversationId` in closure scope matches the event's `conversation_id`; always use the event metadata.

---

## Ollama Events

### `ollama:server_starting`

Emitted when the backend starts `ollama serve` in the background.

### `ollama:server_ready`

Emitted when the Ollama server is ready.

### `ollama:server_error`

Emitted when server startup fails.

### `ollama:pull_progress`

Emitted for each progress line during model download.

### `ollama:pull_done`

Emitted when model download finishes.

### `ollama:pull_error`

Emitted when model download fails.

---

## Embedding Migration Events

### `embedding:migration:start`

Emitted before embedding-version migration starts.

```typescript
type EmbeddingMigrationStart = {
  total: number
}
```

### `embedding:migration:progress`

Emitted while migration is processing facts.

```typescript
type EmbeddingMigrationProgress = {
  processed: number
  total: number
}
```

### `embedding:migration:done`

Emitted when migration completes, with processed and failed counts.

```typescript
type EmbeddingMigrationDone = {
  processed: number
  failed: number
}
```

---

## Auth and Context Events

### `auth:degraded`

Emitted when provider auth health degrades and the frontend should refresh auth state.

```typescript
type AuthDegradedPayload = {
  provider_id: string
  reason: string
}
```

### `context:usage_refresh`

Emitted after manual session compression so the frontend can refresh token-usage estimates.

```typescript
type ContextUsageRefreshPayload = {
  conversation_id: string
}
```

### `context:auto_compressed`

Emitted after automatic context compression.

```typescript
type AutoCompressedPayload = {
  conversation_id: string
  used_after: number
  fallback: boolean
}
```

---

## Error Codes

| Code | Meaning | Handling |
|------|---------|----------|
| `stream internal error` | A panic was recovered during streaming | Restart the stream or restart MedMemo |
| `stream failed` | Provider returned an error during streaming | Display error via `chat:stream_chunk` with `type: 'error'` |
| `embedding pipeline not available` | ONNX / embedding model failed to load | Fall back to keyword-based memory retrieval |
| `context estimator not initialized` | Context estimator dependency missing | Retry after app startup completes |
| `compression service not initialized` | Compression dependency missing | Retry after app startup completes |

---

*Last updated: 2026-07-14*
