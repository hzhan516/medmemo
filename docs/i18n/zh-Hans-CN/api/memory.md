# 记忆 API

> 🌐 [English Version](../../../api/memory.md)

本文档描述用于审核抽取事实、搜索已审批记忆和控制记忆注入的 Wails 绑定方法。

---

## DTO

```go
type MemoryItem struct {
    FactID      string  `json:"fact_id"`
    Subject     string  `json:"subject"`
    Predicate   string  `json:"predicate"`
    Object      string  `json:"object"`
    Confidence  float64 `json:"confidence"`
    Status      string  `json:"status"`
    IsSensitive bool    `json:"is_sensitive"`
    CreatedAt   int64   `json:"created_at"`
}

type MemoryStats struct {
    Total    int64 `json:"total"`
    Approved int64 `json:"approved"`
    Rejected int64 `json:"rejected"`
    Pending  int64 `json:"pending"`
}
```

---

## 方法

### `GetMemories(limit int, offset int) ([]MemoryItem, error)`

返回记忆管理界面的已审批事实列表。`limit` 最大为 100，默认 20。

### `GetMemoryByID(factID string) (MemoryItem, error)`

按 ID 返回单条抽取事实。

### `DeleteMemory(factID string) error`

删除事实，并级联删除对应语义 embedding。

### `SearchMemories(query string) ([]MemoryItem, error)`

通过仓库层过滤搜索已审批记忆。

### `GetPendingReviews(limit int, offset int) ([]MemoryItem, error)`

返回等待用户审批或拒绝的抽取事实。

### `ApproveFact(factID string) error`

将待审核事实标记为已审批，并在 embedding 引擎可用时创建语义 embedding。

### `RejectFact(factID string) error`

将待审核事实标记为已拒绝，并删除该事实可能遗留的 stale embedding。

### `GetMemoryStats() (MemoryStats, error)`

返回事实总数、已审批、已拒绝和待审核数量。

### `GetMemoriesBySession(sessionID string) ([]MemoryItem, error)`

返回与会话关联的已审批记忆。

### `SetMemoryInjectionEnabled(enabled bool) error`

启用或关闭聊天 prompt 组装中的全局记忆注入。

### `SetSessionMemoryInjection(sessionID string, enabled bool) error`

为指定会话覆盖记忆注入开关。

---

*最后更新：2026-07-09*
