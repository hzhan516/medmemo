# Ollama API

> 🌐 [中文版本](../i18n/zh-Hans-CN/api/ollama.md)

This document describes Wails bindings for local model detection and management via Ollama.

---

## Methods

### `DetectOllama() (*OllamaDetectResult, error)`

Checks the local Ollama environment state without triggering background operations.

```go
type OllamaDetectResult struct {
    Installed      bool   `json:"installed"`
    Running        bool   `json:"running"`
    HasSmolLM2     bool   `json:"has_smollm2"`
    InstallGuide   string `json:"install_guide,omitempty"`
    ServerStarting bool   `json:"server_starting,omitempty"`
    PullProgress   string `json:"pull_progress,omitempty"`
}
```

---

### `StartOllamaServer() error`

Attempts to start the Ollama background service. Returns immediately if already running.

---

### `PullOllamaModel(modelName string) error`

Triggers a model download. Progress is pushed via Wails Events.

---

### `EnsureOllamaAndSmolLM2() (*OllamaDetectResult, error)`

Idempotent setup: ensures Ollama is installed, running, and the `smollm2:135m` model is available. Automatically starts the server and pulls the model if needed.

---

### `CreateOllamaProvider() (*models.ProviderConfig, error)`

Generates a ready-to-use ProviderConfig for the local Ollama endpoint (`http://localhost:11434`).
