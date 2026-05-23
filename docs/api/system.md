# System API

> 🌐 [中文版本](../i18n/zh-Hans-CN/api/system.md)

This document describes Wails bindings for system settings, updates, disclaimers, and diagnostics.

---

## Disclaimer

### `GetDisclaimerStatus() (*DisclaimerStatus, error)`

Checks whether the user must accept the current version of the disclaimer.

```go
type DisclaimerStatus struct {
    Required bool   `json:"required"`
    Text     string `json:"text"`
    Version  string `json:"version"`
}
```

---

### `AcceptDisclaimer(version string) error`

Records the user's acceptance of the disclaimer with timestamp and device hash.

---

### `DeclineDisclaimer()`

Triggers application shutdown. Called when the user refuses the disclaimer.

---

## Update

### `CheckUpdate() (*UpdateInfoResponse, error)`

Queries the GitHub releases API for available updates.

```go
type UpdateInfoResponse struct {
    Version     string `json:"version"`
    Name        string `json:"name"`
    Body        string `json:"body"`
    PublishedAt string `json:"published_at"`
    Mandatory   bool   `json:"mandatory"`
    Channel     string `json:"channel"` // "stable" | "beta"
}
```

---

### `DownloadUpdate(req DownloadUpdateRequest) (string, error)`

Downloads the update package for the specified version. Progress is pushed via `update:progress` event.

```go
type DownloadUpdateRequest struct {
    Version string `json:"version"`
}
```

**Returns:** local file path of the downloaded asset.

---

### `ApplyUpdate(assetPath string) error`

Applies the downloaded update. On macOS and Linux this replaces the current binary; on Windows it spawns the updater and exits.

---

### `GetUpdateSettings() (*UpdateSettingsResponse, error)`

```go
type UpdateSettingsResponse struct {
    CheckEnabled bool   `json:"check_enabled"`
    Channel      string `json:"channel"`
    SkipVersion  string `json:"skip_version"`
}
```

---

### `SetUpdateSettings(req UpdateSettingsResponse) error`

Persists update preferences.

---

### `SkipUpdateVersion(v string) error`

Adds a version to the skip list so it will not be prompted again.

---

### `OpenDownloadURL(url string)`

Opens the specified URL in the system default browser via `runtime.BrowserOpenURL`.

---

## Diagnostics

### `GetVersion() string`

Returns the current application version string.

---

### `CollectSystemInfo() (*feedback.SystemInfo, error)`

Gathers anonymized system information for bug reports (OS, arch, app version, memory usage).

---

### `OpenGitHubIssue(userDescription string, errorLog string) error`

Opens the GitHub issue creation page in the system browser with pre-filled title and body.

---

### `GetVersionNotes() []entity.VersionNote`

Returns parsed version notes for display in the "What's New" modal after an update.
