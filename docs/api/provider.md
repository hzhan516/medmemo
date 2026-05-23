# Provider API

> 🌐 [中文版本](../i18n/zh-Hans-CN/api/provider.md)

This document describes Wails bindings for managing AI model providers (Kimi, OpenAI, Qwen, Ollama, etc.).

---

## Methods

### `GetModels() ([]ModelInfo, error)`

Returns the built-in list of available models. This is a static fallback; the authoritative model list comes from each provider's configuration.

```go
type ModelInfo struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Provider string `json:"provider"`
}
```

---

### `ListProviders() ([]models.ProviderConfig, error)`

Returns all configured providers from the backend SQLite store, including API keys (encrypted at rest), model lists, and health status.

---

### `CreateProvider(config models.ProviderConfig) error`

Persists a new provider configuration. The API key is encrypted with AES-256-GCM before storage.

---

### `UpdateProvider(config models.ProviderConfig) error`

Updates an existing provider. If the API key is empty, the existing encrypted key is preserved.

---

### `DeleteProvider(id string) error`

Removes a provider configuration and its associated encrypted API key.

---

### `CheckProviderHealth(providerID string) (*HealthResultResponse, error)`

Performs an on-demand health check for the specified provider.

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

Returns the most recent cached health result without triggering a new check.

---

### `TestAPIKey(providerType string, apiKey string, apiHost string) (*TestAPIKeyResult, error)`

Validates an API key against the provider's authentication endpoint without persisting it.

```go
type TestAPIKeyResult struct {
    Valid   bool     `json:"valid"`
    Message string   `json:"message"`
    Models  []string `json:"models,omitempty"`
}
```

**Supported provider types:** `kimi`, `openai`, `deepseek`, `claude`, `gemini`, `qwen`, `github`, `microsoft`
