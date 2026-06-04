package port

import "context"

// EmbeddingService 定义文本嵌入生成接口。
// 实现者基于 ONNX Runtime 的 FeatureExtractionPipeline 将文本转换为语义向量。
type EmbeddingService interface {
	// Embed 批量生成文本嵌入，输出已 L2 归一化的 384 维向量。
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	// EmbedSingle 单条文本嵌入生成，是 Embed 的便捷包装。
	EmbedSingle(ctx context.Context, text string) ([]float32, error)
}
