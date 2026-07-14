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

| `level` | 前端行为 |
|---------|---------|
| `L2_WARNING` | 橙色警告横幅；允许继续 |
| `L3_NOTICE` | 在消息末尾追加蓝色免责声明横幅 |

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

合规引擎替换整条最后一条 assistant 消息时发出（如 L1 阻断或脱敏还原）。

```typescript
type ReplacePayload = {
  conversation_id: string
  content: string   // 替换文本（标准提示语或还原后的内容）
}
```

---

## `chat:stream:confidence`

流式响应完成后发出，携带置信度分数与 token 用量。

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

在 **严格** 脱敏模式下，当敏感内容无法被安全替换并需要用户确认后再发送时发出。

```typescript
type DeidConfirmPayload = {
  conversation_id: string
  preview: string[]  // 降级后的安全消息预览
}
```

---

## `chat:save_error`

异步消息持久化失败时发出。

```typescript
type SaveErrorPayload = {
  type: string            // 例如 "user_message"、"raw_dialogue"、"update_timestamp"
  conversation_id: string
  error: string
  timestamp: number       // 毫秒
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

---

## Ollama 事件

### `ollama:server_starting`

后端在后台启动 `ollama serve` 时发出。

### `ollama:server_ready`

Ollama 服务就绪时发出。

### `ollama:server_error`

Ollama 服务启动失败时发出。

### `ollama:pull_progress`

模型下载过程中每行进度发出。

### `ollama:pull_done`

模型下载完成时发出。

### `ollama:pull_error`

模型下载失败时发出。

---

## Embedding 迁移事件

### `embedding:migration:start`

embedding 版本迁移开始前发出。

```typescript
type EmbeddingMigrationStart = {
  total: number
}
```

### `embedding:migration:progress`

迁移处理事实期间发出。

```typescript
type EmbeddingMigrationProgress = {
  processed: number
  total: number
}
```

### `embedding:migration:done`

迁移完成时发出，包含处理数量和失败数量。

```typescript
type EmbeddingMigrationDone = {
  processed: number
  failed: number
}
```

---

## 认证与上下文事件

### `auth:degraded`

Provider 认证健康状态降级时发出，前端应刷新认证状态。

```typescript
type AuthDegradedPayload = {
  provider_id: string
  reason: string
}
```

### `context:usage_refresh`

手动会话压缩完成后发出，前端应刷新 token 用量估算。

```typescript
type ContextUsageRefreshPayload = {
  conversation_id: string
}
```

### `context:auto_compressed`

自动上下文压缩完成后发出。

```typescript
type AutoCompressedPayload = {
  conversation_id: string
  used_after: number
  fallback: boolean
}
```

---

## 错误码

| 错误码 | 含义 | 处理建议 |
|--------|------|----------|
| `stream internal error` | 流式输出期间发生 panic 并已恢复 | 重新启动流或重启 MedMemo |
| `stream failed` | 流式输出期间 Provider 返回错误 | 通过 `chat:stream_chunk` 的 `type: 'error'` 显示错误 |
| `embedding pipeline not available` | ONNX / embedding 模型加载失败 | 回退到基于关键词的记忆检索 |
| `context estimator not initialized` | 上下文估算器依赖缺失 | 应用启动完成后重试 |
| `compression service not initialized` | 压缩服务依赖缺失 | 应用启动完成后重试 |

---

*最后更新：2026-07-14*
