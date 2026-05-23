# 对话 API

> 🌐 [English Version](../../../api/chat.md)

本文档描述与会话管理和消息流式输出相关的 Wails 绑定方法。

---

## 方法

### `SendMessage(req SendMessageRequest) (*SendMessageResponse, error)`

发送非流式对话消息。后端编排完整流水线：输入脱敏 → 记忆检索 → 上下文组装 → LLM 调用 → 输出还原 → 合规检测。

**请求：**

```go
type SendMessageRequest struct {
    ConversationID string           `json:"conversation_id"`
    Messages       []models.Message `json:"messages"`
    Model          string           `json:"model"`
    ProviderID     string           `json:"provider_id"`
}
```

**响应：**

```go
type SendMessageResponse struct {
    Reply      string   `json:"reply"`
    Confidence float64  `json:"confidence"`
    Warnings   []string `json:"warnings"`
}
```

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

```go
type ConversationSummary struct {
    ID        string `json:"id"`
    Title     string `json:"title"`
    UpdatedAt string `json:"updated_at"`
    DeletedAt string `json:"deleted_at"` // 空字符串表示未软删除
}
```

---

### `GetConversationMessages(convID string) ([]MessageResponse, error)`

获取指定会话的全部消息。

```go
type MessageResponse struct {
    ID        string `json:"id"`
    Role      string `json:"role"`      // "user" | "assistant" | "system"
    Content   string `json:"content"`
    Timestamp string `json:"timestamp"` // Unix 时间戳（毫秒）
}
```

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

### `GenerateTitle(convID string, userMessage string)`

基于首条用户消息异步生成会话标题。结果通过 `chat:title:generated` 事件送达。此方法不阻塞调用方。

---

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
