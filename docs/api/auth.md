# Auth API

> 🌐 [中文版本](../i18n/zh-Hans-CN/api/auth.md)

This document describes Wails bindings for the four-tier authentication system (CLI Token, OAuth Device Flow, API Key, Local Model).

---

## Methods

### `SaveAPIKey(provider string, apiKey string) error`

Encrypts and stores an API key in the system keyring (macOS Keychain / Windows Credential Manager / Linux Secret Service).

---

### `HasAPIKey(provider string) (bool, error)`

Checks whether an encrypted API key exists for the given provider.

---

### `TestAPIKey(providerType string, apiKey string, apiHost string) (*TestAPIKeyResult, error)`

See [Provider API](provider.md).

---

### `DetectAuthMethods() (*AuthDetectResult, error)`

Auto-detects available authentication methods on the current machine.

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

| Tier | Method | Description |
|------|--------|-------------|
| 1 | `cli_token` | Auto-detect local CLI credentials (Kimi/Gemini) |
| 2 | `oauth_device` | OAuth 2.0 Device Flow (RFC 8628) with auto-refresh |
| 3 | `api_key` | Manual API key input with encrypted storage |
| 4 | `local` | Ollama / local model detection |

---

### `BuildCLIProvider(providerType, modelID string) (*models.ProviderConfig, error)`

Constructs a provider configuration from detected CLI credentials.

---

### `StartOAuthDeviceFlow(providerType string) (*DeviceFlowStartResponse, error)`

Initiates OAuth 2.0 Device Flow. A local callback server is started to receive the authorization completion notification.

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

Cancels an in-progress OAuth Device Flow and shuts down the callback server.

---

### `GetOAuthDeviceFlowStatus(deviceCode string) (*DeviceFlowStatusResponse, error)`

Polls the current status of a Device Flow authorization.

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

Lists providers that support OAuth Device Flow and their configuration status.

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

Parses a Google service account JSON string and extracts the project ID, client email, and private key ID.
