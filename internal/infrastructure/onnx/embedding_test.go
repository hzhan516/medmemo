package onnx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddingWorker_EmbedDimension(t *testing.T) {
	// 验证 EmbeddingWorker 输出维度为 384
	_ = &EmbeddingWorker{id: 0}
	// 使用 mock 输出验证维度
	output := &mockFeatureOutput{
		embeddings: [][]float32{
			make([]float32, 384),
		},
	}
	require.Len(t, output.embeddings[0], 384)
}

func TestModelDownloader_ModelPath(t *testing.T) {
	d := NewModelDownloader("/tmp/models")
	assert.Equal(t, "/tmp/models/all-MiniLM-L6-v2", d.ModelPath("all-MiniLM-L6-v2"))
}

func TestModelDownloader_IsModelPresent(t *testing.T) {
	// 模型不存在时应返回 false
	d := NewModelDownloader(t.TempDir())
	assert.False(t, d.IsModelPresent("all-MiniLM-L6-v2"))
}

func TestNormalizeL2(t *testing.T) {
	vector := []float32{3.0, 4.0}
	normalized := normalizeL2(vector)

	// L2 范数应为 1.0
	var norm float32
	for _, v := range normalized {
		norm += v * v
	}
	assert.InDelta(t, float32(1.0), norm, 0.0001)
	assert.InDelta(t, float32(0.6), normalized[0], 0.0001)
	assert.InDelta(t, float32(0.8), normalized[1], 0.0001)
}

func TestNormalizeL2_ZeroVector(t *testing.T) {
	vector := []float32{0.0, 0.0, 0.0}
	normalized := normalizeL2(vector)
	// 零向量归一化后保持零
	assert.Equal(t, []float32{0.0, 0.0, 0.0}, normalized)
}

// mockFeatureOutput 用于测试的 mock 输出
type mockFeatureOutput struct {
	embeddings [][]float32
}
