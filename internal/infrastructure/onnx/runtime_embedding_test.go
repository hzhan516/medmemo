package onnx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEngine_HasEmbeddingPipeline(t *testing.T) {
	// Engine 应该支持嵌入 pipeline
	e := &Engine{}
	assert.False(t, e.HasEmbeddingPipeline())

	// 模拟已初始化 embedding pipeline
	e.embeddingAvailable = true
	assert.True(t, e.HasEmbeddingPipeline())
}
