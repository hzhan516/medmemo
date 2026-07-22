# 对话 API

> 🌐 [English Version](../../../api/chat.md)

本文档描述与会话管理和消息流式输出相关的 Wails 绑定方法。

---

## 方法

### `SendMessage(req SendMessageRequest) (*SendMessageResponse, error)`

发送非流式对话消息。后端编排完整流水线：输入脱敏 → 记忆检索 → 上下文组装 → LLM 调用 → 输出还原 → 合规检测。

#### `SendMessageRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|:----:|:----|
| `conversation_id` | `string` | ✅ | 目标会话 ID |
| `messages` | `[]models.Message` | ✅ | 当前消息上下文，最后一条为新的用户消息 |
| `model` | `string` | ✅ | 要使用的模型 ID |
| `provider_id` | `string` | ✅ | 要使用的 Provider ID |
| `ai_message_id` | `string` | — | 为助手回复预生成的可选 ID（用于反馈关联） |
| `force_send` | `bool` | — | 在严格脱敏降级后用户确认发送时为 `true` |

#### `SendMessageResponse`

| 字段 | 类型 | 说明 |
|------|------|:----|
| `reply` | `string` | 助手回复内容 |
| `confidence_result` | `map[string]interface{}` | 置信度评分结果（总分、等级、解释等） |
| `warnings` | `[]string` | 合规/风险警告 |

**错误：**
- `ErrInvalidConfig` — Provider 或模型未配置
- `ErrComplianceBlocked` — 内容被合规引擎阻断

---

### `SendMessageStream(req SendMessageRequest) error`

发送流式对话消息。响应通过 Wails Events (`chat:stream_chunk`) 推送到前端。

**请求：** 与 `SendMessage` 相同。

**发出的事件：**
- `chat:stream_chunk` — 见 [事件 API](events.md)
- `chat:stream:compliance` — 合规警告/提示
- `chat:stream:confidence` — 流结束后返回置信度分数与 token 用量
- `chat:stream:replace` — 合规还原占位符时替换已流式文本
- `chat:title:generated` — 自动生成的会话标题（首条消息后）

**错误：**
- 若流无法启动则返回错误；单个 chunk 错误通过事件传递。

---

### `StopGeneration()`

中止当前活跃的流式生成。若无流正在进行则为空操作。

---

### `CreateConversation() (string, error)`

创建新会话并返回唯一 ID。会话通过 `ConversationRepository` 持久化。

**返回：** `conversation_id` 字符串。

---

### `GetConversations() ([]ConversationSummary, error)`

返回所有未删除的会话列表，按 `updated_at` 降序排列。

#### `ConversationSummary`

| 字段 | 类型 | 说明 |
|------|------|:----|
| `id` | `string` | 会话 ID |
| `title` | `string` | 会话标题 |
| `updated_at` | `string` | 最后更新时间戳（毫秒） |
| `deleted_at` | `string` | 软删除时间戳（毫秒）；空字符串表示未删除 |

---

### `GetConversationMessages(convID string) ([]MessageResponse, error)`

获取指定会话的全部消息。

#### `MessageResponse`

| 字段 | 类型 | 说明 |
|------|------|:----|
| `id` | `string` | 消息 ID |
| `role` | `string` | `user` \| `assistant` \| `system` |
| `content` | `string` | 消息内容 |
| `timestamp` | `string` | Unix 时间戳（毫秒） |
| `prompt_tokens` | `int` | 消耗的 prompt token 数 |
| `completion_tokens` | `int` | 消耗的 completion token 数 |
| `total_tokens` | `int` | 消耗的总 token 数 |
| `confidence` | `map[string]interface{}` | 置信度结果；不存在时省略 |

---

### `DeleteConversation(convID string) error`

软删除会话。记录保留 30 天可恢复。

---

### `RestoreConversation(convID string) error`

恢复软删除的会话。

---

### `HardDeleteConversation(convID string) error`

永久删除会话及其所有消息。

---

### `GetDeletedConversations() ([]ConversationSummary, error)`

返回仍处于保留期内的软删除会话。

---

### `GenerateTitle(convID string, userMessage string)`

基于首条用户消息使用本地规则生成会话标题。结果通过 `chat:title:generated` 事件送达。此方法不阻塞调用方。

---

## 上下文与压缩

### `ResolveMaxContextLength(providerID, modelID string) (int, error)`

解析指定 provider/model 的最大上下文长度。

### `EstimateContextUsage(req EstimateContextUsageRequest) (*ContextUsageResponse, error)`

估算当前消息和组装后 prompt 的 token 用量。

#### `EstimateContextUsageRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|:----:|:----|
| `conversationId` | `string` | ✅ | 会话 ID |
| `messages` | `[]models.Message` | ✅ | 待估算的消息 |
| `providerId` | `string` | ✅ | Provider ID |
| `modelId` | `string` | ✅ | 模型 ID |
| `assembledPrompt` | `[]models.Message` | — | 可选的预组装 prompt；若省略，后端将自动组装 |

#### `ContextUsageResponse`

| 字段 | 类型 | 说明 |
|------|------|:----|
| `usedTokens` | `int` | 估算已用 token 数 |
| `maxTokens` | `int` | 模型的最大上下文 token 数 |
| `ratio` | `float64` | `usedTokens / maxTokens` |
| `approximate` | `bool` | 估算是否为近似值 |

---

### `CompressSession(req CompressSessionRequest) error`

压缩会话，并在完成后发出 `context:usage_refresh` 事件。

#### `CompressSessionRequest`

| 字段 | 类型 | 必填 | 说明 |
|------|------|:----:|:----|
| `conversationId` | `string` | ✅ | 会话 ID |
| `providerId` | `string` | ✅ | 模型驱动摘要的 Provider ID |
| `modelId` | `string` | ✅ | 模型驱动摘要的模型 ID |
| `strategy` | `string` | — | `drop_earliest_n`、`summarize_and_replace` 或 `llm_self_summarize` |
| `anchorCount` | `int` | — | 保留的锚点消息数 |
| `recentCount` | `int` | — | 保留的最近消息数 |

### `GetCompressionSettings() models.CompressionSettings`

返回已保存的压缩设置。

### `SetCompressionSettings(s models.CompressionSettings) error`

保存压缩设置。

### `TestCompressionModel(providerID, modelID string) (bool, error)`

检查所选压缩模型是否可用。

---

## 紧急症状与隐私设置

### `CheckEmergency(text string) (*EmergencyResult, error)`

执行本地紧急症状检测（AGENTS.md §7.3）。独立于 AI 回复流程，延迟 <5ms。

```go
type EmergencyResult struct {
    Level   string `json:"level"`   // "A", "B", "none"
    Message string `json:"message"`
    Action  string `json:"action"`
}
```

| 等级 | 触发条件 | 前端行为 |
|------|---------|---------|
| `A` | 危急症状（胸痛伴呼吸困难、意识丧失等） | 全屏红色遮罩，提供 120/急诊选项 |
| `B` | 紧急症状（持续高热>3天、剧烈腹痛等） | 输入框上方红色警告横幅，需点击确认 |
| `none` | 未匹配紧急关键词 | 不中断 UI |

---

### `ShowEmergencyDialog(title, message string)`

为紧急症状流程显示原生警告弹窗。

### `SetDataRetentionDays(days int) error`

保存软删除数据保留天数。

### `SetDesensitizationLevel(level string) error`

保存出站脱敏级别（`standard`、`strict` 或 `off`）。

### `GetDesensitizationLevel() string`

返回当前出站脱敏级别。

---

*最后更新：2026-07-14*
