# Embedding API

> 🌐 [English Version](../../../api/embedding.md)

本文档描述本地 embedding 模型状态与模型目录操作相关的 Wails 绑定方法。

---

## DTO

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

## 方法

### `GetEmbeddingStatus() (*EmbeddingStatusResponse, error)`

返回 embedding 模型文件、tokenizer 文件、ONNX Runtime 动态库与 embedding 引擎是否可用。

### `GetEmbeddingModelDirPath() (string, error)`

返回用户可写的 embedding 模型目录绝对路径，并在缺失时创建目录。

### `OpenEmbeddingModelDir() error`

使用平台文件管理器打开 embedding 模型目录。若自动打开失败，后端会弹窗提示路径。

---

*最后更新：2026-07-09*
