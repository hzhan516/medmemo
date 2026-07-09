# 系统 API

> 🌐 [English Version](../../../api/system.md)

本文档描述系统设置、更新、免责声明和诊断相关的 Wails 绑定方法。

---

## 免责声明

### `GetDisclaimerStatus() (*DisclaimerStatus, error)`

检查用户是否需要同意当前版本的免责声明。

```go
type DisclaimerStatus struct {
    Required bool   `json:"required"`
    Text     string `json:"text"`
    Version  string `json:"version"`
}
```

---

### `AcceptDisclaimer(version string) error`

记录用户对免责声明的同意，包含时间戳和设备哈希。

---

### `DeclineDisclaimer()`

触发应用关闭。用户拒绝免责声明时调用。

---

## 自动更新

### `CheckUpdate() (*UpdateInfoResponse, error)`

查询 GitHub Releases API 获取可用更新。

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

下载指定版本的更新包。进度通过 `update:progress` 事件推送。

```go
type DownloadUpdateRequest struct {
    Version string `json:"version"`
}
```

**返回：** 下载资源的本地文件路径。

---

### `ApplyUpdate(assetPath string) error`

应用已下载的更新。macOS 和 Linux 上替换当前二进制；Windows 上启动更新程序后退出。

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

持久化更新偏好设置。

---

### `SkipUpdateVersion(v string) error`

将指定版本加入跳过列表，之后不再提示。

---

### `OpenDownloadURL(url string)`

通过 `runtime.BrowserOpenURL` 在系统默认浏览器中打开指定 URL。

---

## 诊断

### `GetVersion() string`

返回当前应用版本号。

---

### `GetVersionInfo() (*VersionInfoResponse, error)`

返回 About 页与状态栏使用的结构化版本信息。

```go
type VersionInfoResponse struct {
    Version         string `json:"version"`
    DisplayVersion  string `json:"display_version"`
    BuildNumber     string `json:"build_number"`
    Channel         string `json:"channel"`
    PrereleaseLabel string `json:"prerelease_label"`
    Prerelease      bool   `json:"prerelease"`
}
```

---

### `CollectSystemInfo() (*feedback.SystemInfo, error)`

收集用于错误报告的去标识化系统信息（操作系统、架构、应用版本、内存使用）。

---

### `OpenGitHubIssue(userDescription string, errorLog string) error`

在系统浏览器中打开 GitHub Issue 创建页面，并预填充标题和正文。

---

### `GetVersionNotes() []entity.VersionNote`

返回解析后的版本说明，用于更新后的"新功能"弹窗展示。

---

## 反馈

### `RecordAnswerFeedback(messageID string, answerType string, helpful bool) error`

记录用户对 AI 回答是否有帮助或不准确的反馈。

### `ReportComplianceFeedback(ruleID string, originalText string) error`

记录合规误判反馈，供后续复核。

---

*最后更新：2026-07-09*
