# Ollama API

> 🌐 [English Version](../../../api/ollama.md)

本文档描述通过 Ollama 进行本地模型检测和管理的 Wails 绑定方法。

---

## 方法

### `DetectOllama() (*OllamaDetectResult, error)`

检查本地 Ollama 环境状态，不触发后台操作。

```go
type OllamaDetectResult struct {
    Installed      bool   `json:"installed"`                 // PATH 中是否存在 ollama 命令
    Running        bool   `json:"running"`                   // 11434 端口是否响应
    HasSmolLM2     bool   `json:"has_smollm2"`               // 是否已下载 smollm2:135m 模型
    InstallGuide   string `json:"install_guide,omitempty"`   // 未安装时返回安装引导
    ServerStarting bool   `json:"server_starting,omitempty"` // 是否正在后台启动服务
    PullProgress   string `json:"pull_progress,omitempty"`   // 模型下载进度文本
}
```

---

### `StartOllamaServer() error`

尝试启动 Ollama 后台服务。若已在运行则立即返回。

---

### `PullOllamaModel(modelName string) error`

触发模型下载。进度通过 Wails Events 推送。

---

### `EnsureOllamaAndSmolLM2() (*OllamaDetectResult, error)`

幂等设置：确保 Ollama 已安装、正在运行且 `smollm2:135m` 模型可用。需要时自动启动服务和拉取模型。

---

### `CreateOllamaProvider() (*models.ProviderConfig, error)`

为本地 Ollama 端点（`http://localhost:11434`）生成可直接使用的 ProviderConfig。
