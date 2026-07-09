# Embedding API

> 🌐 [中文版本](../i18n/zh-Hans-CN/api/embedding.md)

This document describes Wails bindings for local embedding model status and model-directory operations.

---

## DTOs

```go
type EmbeddingStatusResponse struct {
    Available         bool   `json:"available"`
    ModelPresent      bool   `json:"model_present"`
    EngineAvailable   bool   `json:"engine_available"`
    RuntimeLibPresent bool   `json:"runtime_lib_present"`
    RuntimeLibPath    string `json:"runtime_lib_path"`
    FailureReason     string `json:"failure_reason"`
    ModelPath         string `json:"model_path"`
    ModelName         string `json:"model_name"`
    DownloadURL       string `json:"download_url"`
}
```

---

## Methods

### `GetEmbeddingStatus() (*EmbeddingStatusResponse, error)`

Reports whether the embedding model file, tokenizer file, ONNX Runtime library, and embedding engine are available.

### `GetEmbeddingModelDirPath() (string, error)`

Returns the absolute user-writable embedding model directory and creates it when missing.

### `OpenEmbeddingModelDir() error`

Opens the embedding model directory with the platform file manager. If automatic opening fails, the backend shows a dialog with the path.

---

*Last updated: 2026-07-09*
