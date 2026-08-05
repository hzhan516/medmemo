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

> ⚙️ **`models.ProviderConfig` 的权威自动生成字段表见 [`_generated/core-types.md`](../../../api/_generated/core-types.md)。**

#### `models.ProviderConfig` 字段（快速参考）

| 字段 | 类型 | 必填 | 说明 |
|------|------|:----:|:----|
| `id` | `string` | ✅ | 唯一 Provider ID |
| `name` | `string` | ✅ | 显示名称 |
| `type` | `string` | ✅ | Provider 类型枚举 |
| `apiHost` | `string` | ✅ | Provider API 基础地址 |
| `apiKey` | `string` | — | API Key / 访问令牌（静态加密存储） |
| `modelId` | `string` | ✅* | 默认模型 ID；若 `models` 存在启用项则可省略 |
| `models` | `[]ProviderModel` | — | 该 Provider 可用模型列表 |
| `temperature` | `float64` | — | 采样温度 `[0, 2]` |
| `timeoutMs` | `int` | — | 请求超时（毫秒） |
| `maxRetries` | `int` | — | 最大重试次数 |
| `maxTokens` | `int` | — | 每次回复最大 token 数 |
| `group` | `string` | — | UI 分组 |
| `enabled` | `bool` | — | 是否启用 |
| `sortOrder` | `int` | — | UI 排序权重 |
| `createdAt` | `int64` | — | 创建时间戳（毫秒） |
| `updatedAt` | `int64` | — | 最后更新时间戳（毫秒） |
| `auth_method` | `string` | — | `api_key`、`cli_token`、`oauth_device`、`service_account` |
| `auth_params` | `AuthParams` | — | 认证方式相关参数 |

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
