# 事件 API

> 🌐 [English Version](../../../api/events.md)

本文档描述后端发出、前端通过 `EventsOn` 消费的 Wails Events。

---

## `chat:stream_chunk`

LLM 流式响应期间发出。

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

| `type` | 含义 |
|--------|------|
| `start` | 流初始化；payload 为空 |
| `content` | 新 token chunk；追加到最后一条 assistant 消息 |
| `done` | 流完成；`metadata` 包含 token 用量统计 |
| `error` | 流失败；`payload` 包含错误信息 |

**前端处理示例：**
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

流式输出期间合规引擎检测到风险等级 ≥ L2 时发出。

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

首条用户消息后，标题生成器完成时发出。

```typescript
type TitlePayload = {
  conv_id: string
  title: string
}
```

---

## `chat:stream:replace`

合规引擎替换整条最后一条 assistant 消息时发出（如 L1 阻断）。

```typescript
type ReplacePayload = {
  conversation_id: string
  content: string   // 替换文本（标准提示语）
}
```

---

## `update:progress`

更新包下载期间发出。

```typescript
type UpdateProgress = {
  percent: number
  downloaded: number
  total: number
  speed: string
}
```

---

## 事件路由说明

- 所有对话事件均在 metadata 中携带 `conversation_id`，即使用户已切换会话，前端也能将其路由到正确的会话。
- 前端不能假设闭包中的 `currentConversationId` 与事件的 `conversation_id` 匹配；始终使用事件 metadata。
