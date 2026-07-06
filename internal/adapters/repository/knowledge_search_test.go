package repository

import (
	"context"
	"sync"
	"testing"

	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spyReranker 包装 KnowledgeReranker，记录 Rerank 调用次数与入参。
type spyReranker struct {
	*usecase.KnowledgeReranker
	mu         sync.Mutex
	calls      int
	queries    []string
	candidates [][]*repository.KnowledgeSearchResult
}

func newSpyReranker() *spyReranker {
	s := &spyReranker{KnowledgeReranker: usecase.NewKnowledgeReranker()}
	s.SetBeforeRerankHook(func(results []*repository.KnowledgeSearchResult, query string) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.calls++
		s.queries = append(s.queries, query)
		s.candidates = append(s.candidates, results)
	})
	return s
}

func (s *spyReranker) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestKnowledgeSearchService_KeywordOnly(t *testing.T) {
	repo, cleanup := newTestKnowledgeRepo(t)
	defer cleanup()

	ctx := context.Background()
	chunker := usecase.NewKnowledgeChunker(512, 32)
	tokenizer := usecase.NewKnowledgeTokenizer()
	importer := usecase.NewKnowledgeImporter(repo, chunker, tokenizer, nil)
	searchSvc := usecase.NewKnowledgeSearchService(repo, tokenizer, nil, nil)

	content := []byte("# 健康指南\n\n感冒通常由病毒引起。发热是身体免疫反应。")
	_, err := importer.ImportFile(ctx, "guide.md", content, false)
	require.NoError(t, err)

	results, err := searchSvc.SearchKeyword(ctx, "感冒症状", 10)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.Equal(t, "keyword", results[0].SourceType)
}

func TestKnowledgeSearchService_HybridUsesVector(t *testing.T) {
	repo, cleanup := newTestKnowledgeRepo(t)
	defer cleanup()

	ctx := context.Background()
	tokenizer := usecase.NewKnowledgeTokenizer()

	// 直接创建文档和片段，绕过 chunker，确保只有一个片段
	require.NoError(t, repo.SaveDocument(ctx, &entity.KnowledgeDocument{
		DocumentID: "doc_vec",
		Title:      "运动建议",
		SourceType: entity.KnowledgeSourceMarkdown,
	}))
	require.NoError(t, repo.SaveChunks(ctx, []*entity.KnowledgeChunk{
		{ChunkID: "chunk_vec", DocumentID: "doc_vec", ChunkIndex: 0, Content: "每周进行有氧运动可以提升心肺功能。"},
	}))

	vec := make([]float32, 384)
	for i := range vec {
		vec[i] = 0.01
	}
	vec[0] = 1.0
	require.NoError(t, repo.SaveEmbedding(ctx, "chunk_vec", "test-model", len(vec), vec))

	// 查询词与内容关键词不匹配，但向量完全匹配，应触发向量召回
	searchSvc := usecase.NewKnowledgeSearchService(repo, tokenizer, &staticEmbeddingSvc{vec: vec}, nil)

	results, err := searchSvc.Search(ctx, "跑步锻炼", 10)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	foundVector := false
	for _, r := range results {
		if r.SourceType == "vector" || r.SourceType == "hybrid" {
			foundVector = true
			break
		}
	}
	assert.True(t, foundVector, "hybrid search should include vector results")
}

func TestKnowledgeSearchService_RerankBoostPhrase(t *testing.T) {
	repo, cleanup := newTestKnowledgeRepo(t)
	defer cleanup()

	ctx := context.Background()
	tokenizer := usecase.NewKnowledgeTokenizer()
	reranker := newSpyReranker()
	searchSvc := usecase.NewKnowledgeSearchService(repo, tokenizer, nil, reranker.KnowledgeReranker)

	// 构造两个片段，关键词检索时后者命中，但精确短语匹配应让前者排到第一
	require.NoError(t, repo.SaveDocument(ctx, &entity.KnowledgeDocument{
		DocumentID: "doc_rr",
		Title:      "健康指南",
		SourceType: entity.KnowledgeSourceMarkdown,
	}))
	require.NoError(t, repo.SaveChunks(ctx, []*entity.KnowledgeChunk{
		{ChunkID: "chunk_rr_1", DocumentID: "doc_rr", ChunkIndex: 0, Content: "预防感冒要多洗手。"},
		{ChunkID: "chunk_rr_2", DocumentID: "doc_rr", ChunkIndex: 1, Content: "感冒通常由病毒引起。"},
	}))
	require.NoError(t, repo.SaveTerms(ctx, "chunk_rr_1", "doc_rr", map[string]int{"预防": 1, "感冒": 1}))
	require.NoError(t, repo.SaveTerms(ctx, "chunk_rr_2", "doc_rr", map[string]int{"感冒": 1, "病毒": 1, "引起": 1}))

	results, err := searchSvc.Search(ctx, "预防感冒", 10)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.Equal(t, "预防感冒要多洗手。", results[0].Content)
	assert.GreaterOrEqual(t, reranker.Calls(), 1, "reranker should be called for keyword-only results")
}

// staticEmbeddingSvc 返回固定向量的测试用 embedding 服务。
type staticEmbeddingSvc struct {
	vec []float32
}

func (s *staticEmbeddingSvc) IsAvailable() bool    { return true }
func (s *staticEmbeddingSvc) ModelVersion() string { return "test-model" }
func (s *staticEmbeddingSvc) EmbedSingle(_ context.Context, _ string) ([]float32, error) {
	return s.vec, nil
}
func (s *staticEmbeddingSvc) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = s.vec
	}
	return out, nil
}

func TestKnowledgeSearchService_KeywordOnlyReranks(t *testing.T) {
	repo, cleanup := newTestKnowledgeRepo(t)
	defer cleanup()

	ctx := context.Background()
	tokenizer := usecase.NewKnowledgeTokenizer()
	reranker := newSpyReranker()
	// embedding 服务不可用，强制走 keyword-only 路径
	searchSvc := usecase.NewKnowledgeSearchService(repo, tokenizer, nil, reranker.KnowledgeReranker)

	require.NoError(t, repo.SaveDocument(ctx, &entity.KnowledgeDocument{
		DocumentID: "doc_ko",
		Title:      "健康指南",
		SourceType: entity.KnowledgeSourceMarkdown,
	}))
	require.NoError(t, repo.SaveChunks(ctx, []*entity.KnowledgeChunk{
		{ChunkID: "chunk_ko_1", DocumentID: "doc_ko", ChunkIndex: 0, Content: "预防感冒要多洗手。"},
		{ChunkID: "chunk_ko_2", DocumentID: "doc_ko", ChunkIndex: 1, Content: "感冒通常由病毒引起。"},
	}))
	require.NoError(t, repo.SaveTerms(ctx, "chunk_ko_1", "doc_ko", map[string]int{"预防": 1, "感冒": 1}))
	require.NoError(t, repo.SaveTerms(ctx, "chunk_ko_2", "doc_ko", map[string]int{"感冒": 1, "病毒": 1, "引起": 1}))

	results, err := searchSvc.Search(ctx, "预防感冒", 10)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.GreaterOrEqual(t, reranker.Calls(), 1, "reranker must be called even when vector search is unavailable")
}

func TestKnowledgeSearchService_HybridFallbackToKeyword(t *testing.T) {
	repo, cleanup := newTestKnowledgeRepo(t)
	defer cleanup()

	ctx := context.Background()
	chunker := usecase.NewKnowledgeChunker(512, 32)
	tokenizer := usecase.NewKnowledgeTokenizer()
	importer := usecase.NewKnowledgeImporter(repo, chunker, tokenizer, nil)
	searchSvc := usecase.NewKnowledgeSearchService(repo, tokenizer, nil, nil)

	content := []byte("# 健康指南\n\n感冒通常由病毒引起。")
	_, err := importer.ImportFile(ctx, "guide.md", content, false)
	require.NoError(t, err)

	results, err := searchSvc.Search(ctx, "感冒", 10)
	require.NoError(t, err)
	require.NotEmpty(t, results)
}
