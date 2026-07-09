# 认证 API

> 🌐 [English Version](../../../api/auth.md)

本文档描述四层鉴权体系（CLI Token、OAuth Device Flow、API Key、本地模型）的 Wails 绑定方法。

---

## 方法

### `SaveAPIKey(provider string, apiKey string) error`

加密并将 API Key 存储到系统密钥环（macOS Keychain / Windows Credential Manager / Linux Secret Service）。

---

### `HasAPIKey(provider string) (bool, error)`

检查给定 Provider 是否存在加密存储的 API Key。

---

### `TestAPIKey(providerType string, apiKey string, apiHost string) (*TestAPIKeyResult, error)`

见 [Provider API](provider.md)。

---

### `DetectAuthMethods() (*AuthDetectResult, error)`

自动检测当前机器上可用的认证方式。

```go
type AuthDetectResult struct {
    Methods []AuthMethodDetectStatus `json:"methods"`
}

type AuthMethodDetectStatus struct {
    Method       string `json:"method"`                  // "cli_token" | "oauth_device" | "api_key" | "local"
    Available    bool   `json:"available"`
    Connected    bool   `json:"connected"`
    Tier         int    `json:"tier"`                    // 1-4
    ProviderType string `json:"provider_type,omitempty"`
    Detail       string `json:"detail,omitempty"`
    Error        string `json:"error,omitempty"`
}
```

| 层级 | 方式 | 说明 |
|------|------|------|
| 1 | `cli_token` | 自动检测本地 CLI 凭据（Kimi/Gemini） |
| 2 | `oauth_device` | OAuth 2.0 Device Flow（RFC 8628），支持 Token 自动刷新 |
| 3 | `api_key` | 手动输入 API Key，加密存储 |
| 4 | `local` | Ollama / 本地模型检测 |

---

### `BuildCLIProvider(providerType, modelID string) (*models.ProviderConfig, error)`

从检测到的 CLI 凭据构建 Provider 配置。

---

### `DetectCLIToken(providerType string) (*auth.CLIDetectResult, error)`

检测受支持 Provider 的 CLI 是否存在可用本地凭据。

---

### `RefreshToken(providerID string) error`

在 Provider 支持刷新时刷新已保存的 OAuth/CLI token。

---

### `EnableAutoRefresh(providerID string) error`

为 Provider 启用自动 token 刷新。

---

### `DisableAutoRefresh(providerID string) error`

为 Provider 关闭自动 token 刷新。

---

### `StartOAuthDeviceFlow(providerType string) (*DeviceFlowStartResponse, error)`

启动 OAuth 2.0 Device Flow。同时启动本地回调服务器作为授权完成的通知通道。

```go
type DeviceFlowStartResponse struct {
    UserCode        string `json:"user_code"`
    VerificationURI string `json:"verification_uri"`
    DeviceCode      string `json:"device_code"`
    ExpiresIn       int    `json:"expires_in"`
    Interval        int    `json:"interval"`
    RedirectURI     string `json:"redirect_uri,omitempty"`
}
```

---

### `CancelOAuthDeviceFlow(deviceCode string) error`

取消进行中的 OAuth Device Flow 并关闭回调服务器。

---

### `GetOAuthDeviceFlowStatus(deviceCode string) (*DeviceFlowStatusResponse, error)`

轮询 Device Flow 授权的当前状态。

```go
type DeviceFlowStatusResponse struct {
    DeviceCode   string `json:"device_code"`
    ProviderType string `json:"provider_type"`
    Status       string `json:"status"` // "pending" | "authorized" | "expired" | "error"
    Error        string `json:"error,omitempty"`
    ProviderID   string `json:"provider_id,omitempty"`
    ProviderName string `json:"provider_name,omitempty"`
}
```

---

### `GetOAuthDeviceFlowProviders() ([]OAuthDeviceFlowProviderInfo, error)`

列出支持 OAuth Device Flow 的厂商及其配置状态。

```go
type OAuthDeviceFlowProviderInfo struct {
    ProviderType string `json:"provider_type"`
    Name         string `json:"name"`
    Available    bool   `json:"available"`
    Configured   bool   `json:"configured"`
    Detail       string `json:"detail"`
}
```

---

### `ParseServiceAccountJSON(jsonStr string) (map[string]string, error)`

解析 Google 服务账号 JSON 字符串，提取项目 ID、客户端邮箱和私钥 ID。

---

*最后更新：2026-07-09*
