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

Emitted when the compliance engine replaces the entire last assistant message (e.g., L1 block).

```typescript
type ReplacePayload = {
  conversation_id: string
  content: string   // replacement text (standard prompt)
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
