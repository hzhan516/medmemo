package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// searchRepoStub 实现 KnowledgeSearchService 依赖的检索方法，
// 其余 KnowledgeRepository 方法由内嵌接口占位。
type searchRepoStub struct {
	repository.KnowledgeRepository

	keywordResults []*repository.KnowledgeSearchResult
	keywordErr     error
	keywordCalls   int
	lastTerms      []string

	vectorResults []*repository.KnowledgeSearchResult
	vectorErr     error
	vectorCalls   int
}

func (s *searchRepoStub) SearchKeyword(_ context.Context, terms []string, _ int) ([]*repository.KnowledgeSearchResult, error) {
	s.keywordCalls++
	s.lastTerms = terms
	if s.keywordErr != nil {
		return nil, s.keywordErr
	}
	return s.keywordResults, nil
}

func (s *searchRepoStub) SearchVector(_ context.Context, _ []float32, _ int) ([]*repository.KnowledgeSearchResult, error) {
	s.vectorCalls++
	if s.vectorErr != nil {
		return nil, s.vectorErr
	}
	return s.vectorResults, nil
}

// searchEmbeddingStub 实现 port.EmbeddingService，用于控制向量检索路径。
type searchEmbeddingStub struct {
	available    bool
	embedErr     error
	embedSingle  []float32
	modelVersion string
}

func (e *searchEmbeddingStub) Embed(_ context.Context, texts []string) ([][]float32, error) {
	if e.embedErr != nil {
		return nil, e.embedErr
	}
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = e.embedSingle
	}
	return out, nil
}

func (e *searchEmbeddingStub) EmbedSingle(_ context.Context, _ string) ([]float32, error) {
	if e.embedErr != nil {
		return nil, e.embedErr
	}
	return e.embedSingle, nil
}

func (e *searchEmbeddingStub) ModelVersion() string { return e.modelVersion }
func (e *searchEmbeddingStub) IsAvailable() bool    { return e.available }

func newSearchService(repo *searchRepoStub, emb *searchEmbeddingStub) *KnowledgeSearchService {
	return NewKnowledgeSearchService(repo, NewKnowledgeTokenizer(), emb, NewKnowledgeReranker())
}

func TestKnowledgeSearchService_Search_EmptyQueryReturnsNil(t *testing.T) {
	t.Parallel()

	repo := &searchRepoStub{}
	svc := newSearchService(repo, &searchEmbeddingStub{})

	// 空白查询分词后无词项，直接返回，无需触达仓储
	out, err := svc.Search(context.Background(), "   ", 5)
	require.NoError(t, err)
	assert.Nil(t, out)
	assert.Equal(t, 0, repo.keywordCalls)
}

func TestKnowledgeSearchService_Search_KeywordOnlyWhenEmbeddingUnavailable(t *testing.T) {
	t.Parallel()

	repo := &searchRepoStub{
		keywordResults: []*repository.KnowledgeSearchResult{
			{ChunkID: "c1", DocumentID: "d1", Content: "diabetes overview", Score: 0.4},
		},
	}
	// 向量服务不可用 -> 纯关键词路径
	emb := &searchEmbeddingStub{available: false}
	svc := newSearchService(repo, emb)

	out, err := svc.Search(context.Background(), "diabetes", 5)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "c1", out[0].ChunkID)
	assert.Equal(t, "keyword", out[0].SourceType)
	assert.Equal(t, 1, repo.keywordCalls)
	assert.Equal(t, 0, repo.vectorCalls) // 不应触发向量检索
}

func TestKnowledgeSearchService_Search_HybridMergesBothPaths(t *testing.T) {
	t.Parallel()

	repo := &searchRepoStub{
		keywordResults: []*repository.KnowledgeSearchResult{
			{ChunkID: "c1", DocumentID: "d1", Content: "hypertension guide", Score: 0.9},
		},
		vectorResults: []*repository.KnowledgeSearchResult{
			{ChunkID: "c2", DocumentID: "d2", Content: "blood pressure notes", Score: 0.8},
		},
	}
	emb := &searchEmbeddingStub{available: true, embedSingle: []float32{0.1, 0.2, 0.3}}
	svc := newSearchService(repo, emb)

	out, err := svc.Search(context.Background(), "hypertension", 5)
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Equal(t, 1, repo.keywordCalls)
	assert.Equal(t, 1, repo.vectorCalls)

	ids := map[string]bool{}
	for _, r := range out {
		ids[r.ChunkID] = true
	}
	assert.True(t, ids["c1"], "keyword result must be present")
	assert.True(t, ids["c2"], "vector result must be present")
}

func TestKnowledgeSearchService_Search_VectorFailureFallsBackToKeyword(t *testing.T) {
	t.Parallel()

	repo := &searchRepoStub{
		keywordResults: []*repository.KnowledgeSearchResult{
			{ChunkID: "c1", DocumentID: "d1", Content: "asthma care", Score: 0.5},
		},
		vectorErr: errors.New("vector index unavailable"),
	}
	emb := &searchEmbeddingStub{available: true, embedSingle: []float32{0.1}}
	svc := newSearchService(repo, emb)

	// 向量检索失败不应阻断，回退到关键词候选
	out, err := svc.Search(context.Background(), "asthma", 5)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "c1", out[0].ChunkID)
	assert.Equal(t, 1, repo.vectorCalls)
}

func TestKnowledgeSearchService_Search_EmbedSingleErrorSkipsVector(t *testing.T) {
	t.Parallel()

	repo := &searchRepoStub{
		keywordResults: []*repository.KnowledgeSearchResult{
			{ChunkID: "c1", DocumentID: "d1", Content: "flu symptoms", Score: 0.5},
		},
	}
	// 嵌入生成失败 -> 不调用 SearchVector，仍返回关键词结果
	emb := &searchEmbeddingStub{available: true, embedErr: errors.New("embed failed")}
	svc := newSearchService(repo, emb)

	out, err := svc.Search(context.Background(), "flu", 5)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, 0, repo.vectorCalls)
}

func TestKnowledgeSearchService_Search_KeywordErrorPropagates(t *testing.T) {
	t.Parallel()

	repo := &searchRepoStub{keywordErr: errors.New("db closed")}
	svc := newSearchService(repo, &searchEmbeddingStub{available: false})

	out, err := svc.Search(context.Background(), "cancer", 5)
	require.Error(t, err)
	assert.Nil(t, out)
	assert.Contains(t, err.Error(), "keyword search failed")
}

func TestKnowledgeSearchService_SearchKeyword_Only(t *testing.T) {
	t.Parallel()

	repo := &searchRepoStub{
		keywordResults: []*repository.KnowledgeSearchResult{
			{ChunkID: "c1", DocumentID: "d1", Content: "vaccine info", Score: 0.7},
		},
	}
	svc := newSearchService(repo, &searchEmbeddingStub{available: true, embedSingle: []float32{0.1}})

	out, err := svc.SearchKeyword(context.Background(), "vaccine", 5)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "c1", out[0].ChunkID)
	assert.Equal(t, 0, repo.vectorCalls) // 关键词专用方法不触发向量检索
}

func TestKnowledgeSearchService_SearchKeyword_EmptyQuery(t *testing.T) {
	t.Parallel()

	repo := &searchRepoStub{}
	svc := newSearchService(repo, &searchEmbeddingStub{})

	out, err := svc.SearchKeyword(context.Background(), "   ", 5)
	require.NoError(t, err)
	assert.Nil(t, out)
	assert.Equal(t, 0, repo.keywordCalls)
}

func TestKnowledgeSearchService_SearchVector_Only(t *testing.T) {
	t.Parallel()

	repo := &searchRepoStub{
		vectorResults: []*repository.KnowledgeSearchResult{
			{ChunkID: "c2", DocumentID: "d2", Content: "semantic hit", Score: 0.6},
		},
	}
	emb := &searchEmbeddingStub{available: true, embedSingle: []float32{0.1, 0.2}}
	svc := newSearchService(repo, emb)

	out, err := svc.SearchVector(context.Background(), "anything", 5)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "c2", out[0].ChunkID)
	assert.Equal(t, 1, repo.vectorCalls)
}

func TestKnowledgeSearchService_SearchVector_UnavailableReturnsNil(t *testing.T) {
	t.Parallel()

	repo := &searchRepoStub{}
	// 嵌入服务不可用时向量检索直接返回 nil
	svc := newSearchService(repo, &searchEmbeddingStub{available: false})

	out, err := svc.SearchVector(context.Background(), "anything", 5)
	require.NoError(t, err)
	assert.Nil(t, out)
	assert.Equal(t, 0, repo.vectorCalls)
}

func TestKnowledgeSearchService_SearchVector_EmbedErrorPropagates(t *testing.T) {
	t.Parallel()

	repo := &searchRepoStub{}
	emb := &searchEmbeddingStub{available: true, embedErr: errors.New("boom")}
	svc := newSearchService(repo, emb)

	out, err := svc.SearchVector(context.Background(), "anything", 5)
	require.Error(t, err)
	assert.Nil(t, out)
	assert.Contains(t, err.Error(), "failed to embed query")
}
