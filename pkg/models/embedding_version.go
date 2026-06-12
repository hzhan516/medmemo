package models

// CurrentEmbeddingVersion 当前 embedding 版本标识。
// 格式：{ONNX模型名}+{检索管线版本}
// 变更此值将触发下次启动时对所有旧版本 embedding 的后台重建。
const CurrentEmbeddingVersion = "all-MiniLM-L6-v2+retrieval-v2"

// EmbeddingModelName ONNX 模型目录名（与版本标识解耦）。
// 用于文件路径解析，与 CurrentEmbeddingVersion 中的模型部分对应。
const EmbeddingModelName = "all-MiniLM-L6-v2"
