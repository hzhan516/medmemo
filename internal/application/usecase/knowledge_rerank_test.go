package usecase

import (
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnowledgeReranker_Rerank_PhraseBoostAndSort(t *testing.T) {
	t.Parallel()

	reranker := NewKnowledgeReranker()
	results := []*repository.KnowledgeSearchResult{
		{ChunkID: "c1", Content: "无关内容", Score: 0.6},
		{ChunkID: "c2", Content: "包含高血压相关描述", Score: 0.5},
	}

	out := reranker.Rerank(results, "高血压")
	require.Len(t, out, 2)

	// c2 命中查询短语获得 +0.5 加分（0.5 -> 1.0），应排在 c1（0.6）之前
	assert.Equal(t, "c2", out[0].ChunkID)
	assert.InDelta(t, 1.0, out[0].Score, 0.0001)
	assert.Equal(t, "c1", out[1].ChunkID)
	assert.InDelta(t, 0.6, out[1].Score, 0.0001)
}

func TestKnowledgeReranker_Rerank_CaseInsensitiveMatch(t *testing.T) {
	t.Parallel()

	reranker := NewKnowledgeReranker()
	results := []*repository.KnowledgeSearchResult{
		{ChunkID: "c1", Content: "Diabetes management guide", Score: 0.1},
	}

	// 查询大小写不同也应命中（内部统一转小写比较）
	out := reranker.Rerank(results, "DIABETES")
	require.Len(t, out, 1)
	assert.InDelta(t, 0.6, out[0].Score, 0.0001)
}

func TestKnowledgeReranker_Rerank_NoMatchKeepsScore(t *testing.T) {
	t.Parallel()

	reranker := NewKnowledgeReranker()
	results := []*repository.KnowledgeSearchResult{
		{ChunkID: "c1", Content: "完全不相关", Score: 0.42},
	}

	out := reranker.Rerank(results, "查询词")
	require.Len(t, out, 1)
	assert.InDelta(t, 0.42, out[0].Score, 0.0001)
}

func TestKnowledgeReranker_Rerank_StableDescendingOrder(t *testing.T) {
	t.Parallel()

	reranker := NewKnowledgeReranker()
	results := []*repository.KnowledgeSearchResult{
		{ChunkID: "low", Content: "x", Score: 0.2},
		{ChunkID: "high", Content: "y", Score: 0.9},
		{ChunkID: "mid", Content: "z", Score: 0.5},
	}

	out := reranker.Rerank(results, "无命中查询")
	require.Len(t, out, 3)
	assert.Equal(t, "high", out[0].ChunkID)
	assert.Equal(t, "mid", out[1].ChunkID)
	assert.Equal(t, "low", out[2].ChunkID)
}

func TestKnowledgeReranker_Rerank_Empty(t *testing.T) {
	t.Parallel()

	reranker := NewKnowledgeReranker()
	out := reranker.Rerank(nil, "任意")
	assert.Empty(t, out)
}

func TestKnowledgeReranker_SetBeforeRerankHook_Invoked(t *testing.T) {
	t.Parallel()

	reranker := NewKnowledgeReranker()

	var gotQuery string
	var gotLen int
	reranker.SetBeforeRerankHook(func(results []*repository.KnowledgeSearchResult, query string) {
		gotQuery = query
		gotLen = len(results)
	})

	results := []*repository.KnowledgeSearchResult{
		{ChunkID: "c1", Content: "内容", Score: 0.3},
	}
	reranker.Rerank(results, "钩子查询")

	// 测试钩子应在重排前被调用，并观察到原始入参
	assert.Equal(t, "钩子查询", gotQuery)
	assert.Equal(t, 1, gotLen)
}
