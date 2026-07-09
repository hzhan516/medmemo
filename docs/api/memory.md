# Memory API

> 🌐 [中文版本](../i18n/zh-Hans-CN/api/memory.md)

This document describes Wails bindings for reviewing extracted personal facts, searching approved memories, and controlling memory injection.

---

## DTOs

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

## Methods

### `GetMemories(limit int, offset int) ([]MemoryItem, error)`

Returns approved facts for the memory management UI. `limit` is capped at 100 and defaults to 20.

### `GetMemoryByID(factID string) (MemoryItem, error)`

Returns one extracted fact by ID.

### `DeleteMemory(factID string) error`

Deletes a fact and cascades deletion to its semantic embedding.

### `SearchMemories(query string) ([]MemoryItem, error)`

Searches approved memories with repository-level filtering.

### `GetPendingReviews(limit int, offset int) ([]MemoryItem, error)`

Returns pending extracted facts that require user approval or rejection.

### `ApproveFact(factID string) error`

Marks a pending fact as approved and creates its semantic embedding when the embedding engine is available.

### `RejectFact(factID string) error`

Marks a pending fact as rejected and removes any stale embedding for that fact.

### `GetMemoryStats() (MemoryStats, error)`

Returns total, approved, rejected, and pending fact counts.

### `GetMemoriesBySession(sessionID string) ([]MemoryItem, error)`

Returns approved memories linked to a conversation session.

### `SetMemoryInjectionEnabled(enabled bool) error`

Enables or disables global memory injection for chat prompt assembly.

### `SetSessionMemoryInjection(sessionID string, enabled bool) error`

Overrides memory injection for a specific conversation session.

---

*Last updated: 2026-07-09*
