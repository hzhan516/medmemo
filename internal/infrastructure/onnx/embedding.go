package onnx

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/knights-analytics/hugot/pipelines"
)

// EmbeddingWorker ONNX 嵌入推理 Worker，通过 sync.Mutex 保护 pipeline.Run 的串行调用。
type EmbeddingWorker struct {
	id       int
	pipeline *pipelines.FeatureExtractionPipeline
	mu       sync.Mutex
}

// NewEmbeddingWorker 创建嵌入推理 Worker。
func NewEmbeddingWorker(id int, pipeline *pipelines.FeatureExtractionPipeline) *EmbeddingWorker {
	return &EmbeddingWorker{id: id, pipeline: pipeline}
}

// Embed 执行嵌入推理，必须在单 Worker 内串行调用。
func (w *EmbeddingWorker) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	result, err := w.pipeline.RunPipeline(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("embedding pipeline run failed: %w", err)
	}

	return result.Embeddings, nil
}

// normalizeL2 对向量进行 L2 归一化。
func normalizeL2(v []float32) []float32 {
	var norm float64
	for _, x := range v {
		norm += float64(x) * float64(x)
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return v
	}
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(float64(x) / norm)
	}
	return out
}

// ModelDownloader 负责从远程下载 ONNX 模型文件。
type ModelDownloader struct {
	baseDir string
}

// NewModelDownloader 创建模型下载器。
func NewModelDownloader(baseDir string) *ModelDownloader {
	return &ModelDownloader{baseDir: baseDir}
}

// ModelPath 返回指定模型的本地存储路径。
func (d *ModelDownloader) ModelPath(modelName string) string {
	return filepath.Join(d.baseDir, modelName)
}

// IsModelPresent 检查模型是否已下载（存在 model.onnx 和 tokenizer.json）。
func (d *ModelDownloader) IsModelPresent(modelName string) bool {
	modelDir := d.ModelPath(modelName)
	onnxPath := filepath.Join(modelDir, "model.onnx")
	tokenizerPath := filepath.Join(modelDir, "tokenizer.json")

	if _, err := os.Stat(onnxPath); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(tokenizerPath); os.IsNotExist(err) {
		return false
	}
	return true
}
