package ai

import (
	"context"
	"testing"

	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddingServiceAdapter_EmbedSingle(t *testing.T) {
	// 使用 mock 引擎测试接口契约
	mock := &mockEmbeddingEngine{
		embeddings: [][]float32{{3.0, 4.0}}, // 会被 L2 归一化为 {0.6, 0.8}
	}
	svc := NewEmbeddingServiceAdapter(mock, models.CurrentEmbeddingVersion)

	vec, err := svc.EmbedSingle(context.Background(), "测试文本")
	require.NoError(t, err)
	require.Len(t, vec, 2)
	assert.InDelta(t, float32(0.6), vec[0], 0.001)
	assert.InDelta(t, float32(0.8), vec[1], 0.001)
}

func TestEmbeddingServiceAdapter_Embed(t *testing.T) {
	mock := &mockEmbeddingEngine{
		embeddings: [][]float32{
			{0.1, 0.2, 0.3},
			{0.4, 0.5, 0.6},
		},
	}
	svc := NewEmbeddingServiceAdapter(mock, models.CurrentEmbeddingVersion)

	vecs, err := svc.Embed(context.Background(), []string{"文本1", "文本2"})
	require.NoError(t, err)
	require.Len(t, vecs, 2)
}

func TestEmbeddingServiceAdapter_EmptyText(t *testing.T) {
	mock := &mockEmbeddingEngine{
		embeddings: [][]float32{{0.0, 0.0, 0.0}},
	}
	svc := NewEmbeddingServiceAdapter(mock, models.CurrentEmbeddingVersion)

	vec, err := svc.EmbedSingle(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, vec, 3)
}

func TestEmbeddingServiceAdapter_CacheHit(t *testing.T) {
	mock := &mockEmbeddingEngine{
		embeddings: [][]float32{{0.1, 0.2, 0.3}},
	}
	svc := NewEmbeddingServiceAdapter(mock, models.CurrentEmbeddingVersion)

	// 第一次调用
	_, err := svc.EmbedSingle(context.Background(), "缓存测试")
	require.NoError(t, err)
	callCount := mock.callCount

	// 第二次调用相同文本，应该命中缓存
	_, err = svc.EmbedSingle(context.Background(), "缓存测试")
	require.NoError(t, err)
	assert.Equal(t, callCount, mock.callCount, "cache should prevent duplicate engine calls")
}

func TestEmbeddingServiceAdapter_BatchSplit(t *testing.T) {
	mock := &mockEmbeddingEngine{}
	svc := NewEmbeddingServiceAdapter(mock, models.CurrentEmbeddingVersion)
	svc.batchSize = 2 // 小 batch size 便于测试

	// 3 条文本，batch size = 2，应该分 2 批
	mock.embeddings = make([][]float32, 3)
	for i := range mock.embeddings {
		mock.embeddings[i] = []float32{float32(i), float32(i), float32(i)}
	}

	vecs, err := svc.Embed(context.Background(), []string{"a", "b", "c"})
	require.NoError(t, err)
	require.Len(t, vecs, 3)
	assert.GreaterOrEqual(t, mock.callCount, 2, "should split into at least 2 batches")
}

func TestEmbeddingServiceAdapter_ContextCancel(t *testing.T) {
	mock := &mockEmbeddingEngine{checkContext: true}
	svc := NewEmbeddingServiceAdapter(mock, models.CurrentEmbeddingVersion)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.EmbedSingle(ctx, "测试")
	assert.Error(t, err)
}

func TestEmbeddingServiceAdapter_ModelVersion(t *testing.T) {
	mock := &mockEmbeddingEngine{}
	svc := NewEmbeddingServiceAdapter(mock, "test-model-v1")
	assert.Equal(t, "test-model-v1", svc.ModelVersion())
}

func TestEmbeddingServiceAdapter_CacheNamespacedByModelVersion(t *testing.T) {
	mock := &mockEmbeddingEngine{
		embeddings: [][]float32{{0.1, 0.2, 0.3}},
	}
	svcV1 := NewEmbeddingServiceAdapter(mock, "model-v1")

	_, err := svcV1.EmbedSingle(context.Background(), "共享文本")
	require.NoError(t, err)
	callCountAfterV1 := mock.callCount

	// 相同文本、不同版本，应视为缓存 miss，重新调用引擎
	svcV2 := NewEmbeddingServiceAdapter(mock, "model-v2")
	_, err = svcV2.EmbedSingle(context.Background(), "共享文本")
	require.NoError(t, err)
	assert.Greater(t, mock.callCount, callCountAfterV1, "不同 model version 不应命中同一缓存")

	// 相同版本再次调用应命中缓存
	_, err = svcV1.EmbedSingle(context.Background(), "共享文本")
	require.NoError(t, err)
	assert.Equal(t, callCountAfterV1+1, mock.callCount, "相同 model version 应命中缓存")
}

func TestEmbeddingCache_evictLRU(t *testing.T) {
	cache := newEmbeddingCache(2)

	cache.Set("a", []float32{1.0})
	cache.Set("b", []float32{2.0})
	// 缓存已满，再添加应触发驱逐
	cache.Set("c", []float32{3.0})

	// "a" 应该被驱逐
	_, ok := cache.Get("a")
	assert.False(t, ok, "oldest entry should be evicted")

	// "b" 和 "c" 应该还在
	_, ok = cache.Get("b")
	assert.True(t, ok)
	_, ok = cache.Get("c")
	assert.True(t, ok)
}

// mockEmbeddingEngine 用于测试的 mock 引擎
type mockEmbeddingEngine struct {
	embeddings   [][]float32
	callCount    int
	checkContext bool
}

func (m *mockEmbeddingEngine) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.checkContext && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	m.callCount++
	result := make([][]float32, len(texts))
	for i := range texts {
		if i < len(m.embeddings) {
			result[i] = m.embeddings[i]
		} else {
			result[i] = []float32{0.0, 0.0, 0.0}
		}
	}
	return result, nil
}

func (m *mockEmbeddingEngine) HasEmbeddingPipeline() bool {
	return true
}
