# Provider API

> 🌐 [English Version](../../../api/provider.md)

本文档描述管理 AI 模型提供商（Kimi、OpenAI、通义千问、Ollama 等）的 Wails 绑定方法。

---

## 方法

### `GetModels() ([]ModelInfo, error)`

返回内置的可用模型列表。此为静态降级方案；权威模型列表来自各 Provider 配置。

```go
type ModelInfo struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Provider string `json:"provider"`
}
```

---

### `ListProviders() ([]models.ProviderConfig, error)`

从后端 SQLite 返回所有已配置的 Provider，包括 API Key（静态加密存储）、模型列表和健康状态。

---

### `CreateProvider(config models.ProviderConfig) error`

持久化新 Provider 配置。API Key 在存储前使用 AES-256-GCM 加密。

---

### `UpdateProvider(config models.ProviderConfig) error`

更新现有 Provider。若 API Key 为空，则保留现有加密密钥。

---

### `DeleteProvider(id string) error`

删除 Provider 配置及其关联的加密 API Key。

---

### `CheckProviderHealth(providerID string) (*HealthResultResponse, error)`

对指定 Provider 执行即时健康检测。

```go
type HealthResultResponse struct {
    ProviderID string `json:"provider_id"`
    Status     string `json:"status"`     // "green" | "yellow" | "red"
    LatencyMs  int64  `json:"latency_ms"`
    CheckedAt  string `json:"checked_at"`
    Error      string `json:"error,omitempty"`
}
```

---

### `GetProviderHealthStatus(providerID string) (*HealthResultResponse, error)`

返回最近一次缓存的健康结果，不触发新检测。

---

### `TestAPIKey(providerType string, apiKey string, apiHost string) (*TestAPIKeyResult, error)`

向 Provider 认证端点验证 API Key，但不持久化存储。

```go
type TestAPIKeyResult struct {
    Valid   bool     `json:"valid"`
    Message string   `json:"message"`
    Models  []string `json:"models,omitempty"`
}
```

**支持的 provider 类型：** `kimi`, `openai`, `deepseek`, `claude`, `gemini`, `qwen`, `github`, `microsoft`
