package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== Stub 实现 ==========

type stubEmbeddingService struct {
	vectors [][]float32
	err     error
}

func (s *stubEmbeddingService) Embed(_ context.Context, _ []string) ([][]float32, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.vectors, nil
}

func (s *stubEmbeddingService) EmbedSingle(_ context.Context, _ string) ([]float32, error) {
	if s.err != nil {
		return nil, s.err
	}
	if len(s.vectors) > 0 {
		return s.vectors[0], nil
	}
	return make([]float32, entity.EmbeddingDimension), nil
}

func (s *stubEmbeddingService) ModelVersion() string { return "test-version" }
func (s *stubEmbeddingService) IsAvailable() bool    { return true }

type stubEmbeddingRepository struct {
	results []*entity.ScoredEmbedding
	err     error
}

func (s *stubEmbeddingRepository) Save(_ context.Context, _ *entity.SemanticEmbedding) error {
	return nil
}
func (s *stubEmbeddingRepository) GetByFactID(_ context.Context, _ string) (*entity.SemanticEmbedding, error) {
	return nil, nil
}
func (s *stubEmbeddingRepository) DeleteByFactID(_ context.Context, _ string) error {
	return nil
}
func (s *stubEmbeddingRepository) SearchSimilar(_ context.Context, _ []float32, _ int) ([]*entity.ScoredEmbedding, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.results, nil
}

func (s *stubEmbeddingRepository) SearchSimilarFiltered(ctx context.Context, queryVector []float32, topK int, _ string) ([]*entity.ScoredEmbedding, error) {
	return s.SearchSimilar(ctx, queryVector, topK)
}

func (s *stubEmbeddingRepository) CountByVersionNot(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (s *stubEmbeddingRepository) UpdateEmbedding(_ context.Context, _ *entity.SemanticEmbedding) error {
	return nil
}

type stubFactRepository struct {
	facts                    map[string]*entity.ExtractedFact
	approvedByPredicatesFunc func(ctx context.Context, subject string, predicates []string, limit int) ([]*entity.ExtractedFact, error)
	findBySessionFunc        func(ctx context.Context, sessionID string) ([]*entity.ExtractedFact, error)
}

func (s *stubFactRepository) Save(_ context.Context, _ *entity.ExtractedFact) error { return nil }
func (s *stubFactRepository) GetByID(_ context.Context, factID string) (*entity.ExtractedFact, error) {
	f, ok := s.facts[factID]
	if !ok {
		return nil, entity.ErrFactNotFound
	}
	return f, nil
}
func (s *stubFactRepository) FindByIDs(_ context.Context, factIDs []string) (map[string]*entity.ExtractedFact, error) {
	result := make(map[string]*entity.ExtractedFact, len(factIDs))
	for _, id := range factIDs {
		if f, ok := s.facts[id]; ok {
			result[id] = f
		}
	}
	return result, nil
}

// spyFactRepository 用于验证 recallByVector 只通过 FindByIDs 批量加载事实。
type spyFactRepository struct {
	stubFactRepository
	findByIDsCalls int
	getByIDCalls   int
}

func (s *spyFactRepository) GetByID(ctx context.Context, factID string) (*entity.ExtractedFact, error) {
	s.getByIDCalls++
	return s.stubFactRepository.GetByID(ctx, factID)
}

func (s *spyFactRepository) FindByIDs(ctx context.Context, factIDs []string) (map[string]*entity.ExtractedFact, error) {
	s.findByIDsCalls++
	return s.stubFactRepository.FindByIDs(ctx, factIDs)
}

func (s *stubFactRepository) ListByStatus(_ context.Context, _ entity.FactStatus, _, _ int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (s *stubFactRepository) ListPending(_ context.Context, _, _ int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (s *stubFactRepository) UpdateStatus(_ context.Context, _ string, _ entity.FactStatus) error {
	return nil
}
func (s *stubFactRepository) Delete(_ context.Context, _ string) error { return nil }
func (s *stubFactRepository) GetStats(_ context.Context) (total, approved, rejected, pending int64, err error) {
	return 0, 0, 0, 0, nil
}
func (s *stubFactRepository) ListAllSubjects(_ context.Context) ([]string, error) {
	return nil, nil
}
func (s *stubFactRepository) FindBySubject(_ context.Context, _ string) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (s *stubFactRepository) FindBySession(ctx context.Context, sessionID string) ([]*entity.ExtractedFact, error) {
	if s.findBySessionFunc != nil {
		return s.findBySessionFunc(ctx, sessionID)
	}
	return nil, nil
}
func (s *stubFactRepository) FindApprovedByPredicates(ctx context.Context, subject string, predicates []string, limit int) ([]*entity.ExtractedFact, error) {
	if s.approvedByPredicatesFunc != nil {
		return s.approvedByPredicatesFunc(ctx, subject, predicates, limit)
	}
	return nil, nil
}
func (s *stubFactRepository) FindLatestApprovedByPredicates(_ context.Context, _ string, _ []string) (*entity.ExtractedFact, error) {
	return nil, entity.ErrFactNotFound
}
func (s *stubFactRepository) SearchApproved(_ context.Context, _ string, _ int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}

func (s *stubFactRepository) CountApprovedFactsNeedingEmbedding(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (s *stubFactRepository) ListApprovedFactsNeedingEmbedding(_ context.Context, _ string, _ time.Time, _ string, _ int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}

// stubMemoryRepository 用于 ArchiveConversation 测试的 memoryRepo stub。
type stubMemoryRepository struct {
	saved []*entity.HealthMemory
	err   error
}

func (s *stubMemoryRepository) Save(_ context.Context, mem *entity.HealthMemory) error {
	if s.err != nil {
		return s.err
	}
	s.saved = append(s.saved, mem)
	return nil
}
func (s *stubMemoryRepository) GetByID(_ context.Context, _ models.MemoryID) (*entity.HealthMemory, error) {
	return nil, fmt.Errorf("not implemented")
}
func (s *stubMemoryRepository) Search(_ context.Context, _ string, _ int) ([]*entity.HealthMemory, error) {
	return nil, nil
}
func (s *stubMemoryRepository) SemanticSearch(_ context.Context, _ []float32, _ int) ([]*entity.HealthMemory, error) {
	return nil, nil
}
func (s *stubMemoryRepository) ListByTier(_ context.Context, _ entity.MemoryTier, _ int) ([]*entity.HealthMemory, error) {
	return nil, nil
}
func (s *stubMemoryRepository) Delete(_ context.Context, _ models.MemoryID) error {
	return nil
}

// stubFactRepositoryWithSubjects 支持实体提及检测的 stub
type stubFactRepositoryWithSubjects struct {
	stubFactRepository
	subjects  []string
	bySubject map[string][]*entity.ExtractedFact
	facts     []*entity.ExtractedFact
}

func (s *stubFactRepositoryWithSubjects) ListAllSubjects(_ context.Context) ([]string, error) {
	return s.subjects, nil
}
func (s *stubFactRepositoryWithSubjects) FindBySubject(_ context.Context, subject string) ([]*entity.ExtractedFact, error) {
	return s.bySubject[subject], nil
}
func (s *stubFactRepositoryWithSubjects) ListByStatus(_ context.Context, status entity.FactStatus, _, _ int) ([]*entity.ExtractedFact, error) {
	var result []*entity.ExtractedFact
	for _, f := range s.facts {
		if f.Status == status {
			result = append(result, f)
		}
	}
	return result, nil
}

// 确保 stub 实现满足接口
var _ repository.EmbeddingRepository = (*stubEmbeddingRepository)(nil)
var _ repository.FactRepository = (*stubFactRepository)(nil)

// ========== 测试 ==========

func TestMemoryRetriever_SemanticSearch(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()

	facts := map[string]*entity.ExtractedFact{
		"fact_a": {
			FactID: "fact_a", Subject: "用户", Predicate: "患有", Object: "高血压",
			Confidence: 0.85, Status: entity.FactStatusApproved, CreatedAt: now,
		},
		"fact_b": {
			FactID: "fact_b", Subject: "用户", Predicate: "服用", Object: "降压药",
			Confidence: 0.75, Status: entity.FactStatusApproved, CreatedAt: now,
		},
	}

	embeddings := []*entity.ScoredEmbedding{
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_a"}, Similarity: 0.95},
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_b"}, Similarity: 0.88},
	}

	retriever := NewMemoryRetriever(
		&stubEmbeddingService{},
		&stubEmbeddingRepository{results: embeddings},
		&stubFactRepository{facts: facts}, nil,
		NewDecayScorer(),
		nil,
		nil,
		nil,
	)

	memories, err := retriever.RetrieveForContext(context.Background(), "我的血压怎么样", "session_001", 2)
	require.NoError(t, err)
	require.Len(t, memories, 2)

	assert.Equal(t, "用户 患有 高血压", memories[0].Content)
	assert.Equal(t, "用户 服用 降压药", memories[1].Content)
}

func TestMemoryRetriever_RecallByVector_BatchFetch(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()

	facts := map[string]*entity.ExtractedFact{
		"fact_a": {
			FactID: "fact_a", Subject: "用户", Predicate: "患有", Object: "高血压",
			Confidence: 0.85, Status: entity.FactStatusApproved, CreatedAt: now,
		},
		"fact_b": {
			FactID: "fact_b", Subject: "用户", Predicate: "服用", Object: "降压药",
			Confidence: 0.75, Status: entity.FactStatusApproved, CreatedAt: now,
		},
		"fact_c": {
			FactID: "fact_c", Subject: "用户", Predicate: "患有", Object: "低血压",
			Confidence: 0.50, Status: entity.FactStatusApproved, CreatedAt: now,
		},
		"fact_d": {
			FactID: "fact_d", Subject: "用户", Predicate: "患有", Object: "糖尿病",
			Confidence: 0.85, Status: entity.FactStatusPending, CreatedAt: now,
		},
	}

	embeddings := []*entity.ScoredEmbedding{
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_a"}, Similarity: 0.95},
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_b"}, Similarity: 0.88},
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_c"}, Similarity: 0.80},
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_d"}, Similarity: 0.90},
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "missing"}, Similarity: 0.85},
	}

	spy := &spyFactRepository{stubFactRepository: stubFactRepository{facts: facts}}
	retriever := NewMemoryRetriever(
		&stubEmbeddingService{},
		&stubEmbeddingRepository{results: embeddings},
		spy, nil,
		NewDecayScorer(),
		nil,
		nil,
		nil,
	)

	diag, _, err := retriever.retrieveWithDiagnostics(context.Background(), "我的血压怎么样", "session_001", 5)
	require.NoError(t, err)

	require.Len(t, diag.VectorCandidates, 2)
	assert.Equal(t, "fact_a", diag.VectorCandidates[0].FactID)
	assert.Equal(t, "fact_b", diag.VectorCandidates[1].FactID)
	assert.Equal(t, 1, spy.findByIDsCalls, "应通过一次批量查询加载事实")
	assert.Equal(t, 0, spy.getByIDCalls, "不应再逐个调用 GetByID")
}

func TestMemoryRetriever_DecayRanking(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()

	// fact_old: 30 天前，similarity = 1.0 → 衰减后 ≈ 0.223
	// fact_new: 今天，similarity = 0.5 → 衰减后 = 0.5
	// 预期排序：fact_new > fact_old
	facts := map[string]*entity.ExtractedFact{
		"fact_old": {
			FactID: "fact_old", Subject: "用户", Predicate: "患有", Object: "感冒",
			Confidence: 1.0, Status: entity.FactStatusApproved,
			CreatedAt: now.Add(-30 * 24 * time.Hour),
		},
		"fact_new": {
			FactID: "fact_new", Subject: "用户", Predicate: "服用", Object: "维生素",
			Confidence: 1.0, Status: entity.FactStatusApproved,
			CreatedAt: now,
		},
	}

	// 故意把旧记忆排在前面（高相似度）
	embeddings := []*entity.ScoredEmbedding{
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_old"}, Similarity: 1.0},
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_new"}, Similarity: 0.5},
	}

	retriever := NewMemoryRetriever(
		&stubEmbeddingService{},
		&stubEmbeddingRepository{results: embeddings},
		&stubFactRepository{facts: facts}, nil,
		NewDecayScorer(),
		nil,
		nil,
		nil,
	)
	retriever.minConfidence = 0.1 // 降低阈值以便测试衰减排序

	memories, err := retriever.RetrieveForContext(context.Background(), "query", "session_001", 10)
	require.NoError(t, err)
	require.Len(t, memories, 2)

	// 时间衰减后，新记忆应排在前面
	assert.Equal(t, "用户 服用 维生素", memories[0].Content)
	assert.Equal(t, "用户 患有 感冒", memories[1].Content)
}

func TestMemoryRetriever_FilterUnapproved(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()

	facts := map[string]*entity.ExtractedFact{
		"fact_approved": {
			FactID: "fact_approved", Subject: "用户", Predicate: "患有", Object: "高血压",
			Confidence: 0.9, Status: entity.FactStatusApproved, CreatedAt: now,
		},
		"fact_pending": {
			FactID: "fact_pending", Subject: "用户", Predicate: "患有", Object: "糖尿病",
			Confidence: 0.9, Status: entity.FactStatusPending, CreatedAt: now,
		},
	}

	embeddings := []*entity.ScoredEmbedding{
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_approved"}, Similarity: 0.9},
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_pending"}, Similarity: 0.85},
	}

	retriever := NewMemoryRetriever(
		&stubEmbeddingService{},
		&stubEmbeddingRepository{results: embeddings},
		&stubFactRepository{facts: facts}, nil,
		NewDecayScorer(),
		nil,
		nil,
		nil,
	)

	memories, err := retriever.RetrieveForContext(context.Background(), "query", "session_001", 10)
	require.NoError(t, err)
	require.Len(t, memories, 1)
	assert.Equal(t, "用户 患有 高血压", memories[0].Content)
}

func TestMemoryRetriever_MinConfidenceFilter(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()

	facts := map[string]*entity.ExtractedFact{
		"fact_high": {
			FactID: "fact_high", Subject: "用户", Predicate: "患有", Object: "高血压",
			Confidence: 0.9, Status: entity.FactStatusApproved, CreatedAt: now,
		},
		"fact_low": {
			FactID: "fact_low", Subject: "用户", Predicate: "感觉", Object: "疲劳",
			Confidence: 0.3, Status: entity.FactStatusApproved, CreatedAt: now,
		},
	}

	embeddings := []*entity.ScoredEmbedding{
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_high"}, Similarity: 0.8},
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_low"}, Similarity: 0.7},
	}

	retriever := NewMemoryRetriever(
		&stubEmbeddingService{},
		&stubEmbeddingRepository{results: embeddings},
		&stubFactRepository{facts: facts}, nil,
		NewDecayScorer(),
		nil,
		nil,
		nil,
	)
	retriever.minConfidence = 0.6 // 设置较高阈值

	memories, err := retriever.RetrieveForContext(context.Background(), "query", "session_001", 10)
	require.NoError(t, err)
	require.Len(t, memories, 1)
	assert.Equal(t, "用户 患有 高血压", memories[0].Content)
}

func TestMemoryRetriever_EmbedFailure(t *testing.T) {
	t.Parallel()
	retriever := NewMemoryRetriever(
		&stubEmbeddingService{err: fmt.Errorf("embedding failed")},
		&stubEmbeddingRepository{},
		&stubFactRepository{}, nil,
		NewDecayScorer(),
		nil,
		nil,
		nil,
	)

	// 嵌入失败时应返回空结果而非报错（降级）
	memories, err := retriever.RetrieveForContext(context.Background(), "query", "session_001", 3)
	require.NoError(t, err)
	assert.Empty(t, memories)
}

func TestMemoryRetriever_WeightRecallThroughSemanticSearch(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	facts := map[string]*entity.ExtractedFact{
		"fact_weight": {
			FactID: "fact_weight", Subject: "用户", Predicate: "体重是", Object: "110公斤",
			Confidence: 0.95, Status: entity.FactStatusApproved, CreatedAt: now,
		},
	}
	embeddings := []*entity.ScoredEmbedding{
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_weight"}, Similarity: 0.92},
	}
	retriever := NewMemoryRetriever(
		&stubEmbeddingService{},
		&stubEmbeddingRepository{results: embeddings},
		&stubFactRepository{facts: facts}, nil,
		NewDecayScorer(),
		nil,
		nil,
		nil,
	)

	memories, err := retriever.RetrieveForContext(context.Background(), "我现在多重", "session_weight", 3)
	require.NoError(t, err)
	require.Len(t, memories, 1)
	assert.Equal(t, "用户 体重是 110公斤", memories[0].Content)

	messages := injectMemories([]models.Message{
		{Role: models.RoleUser, Content: "我现在多重"},
	}, memories)
	require.Len(t, messages, 2)
	assert.Equal(t, models.RoleSystem, messages[0].Role)
	assert.Contains(t, messages[0].Content, "用户 体重是 110公斤")
}

func TestMemoryRetriever_NoResults(t *testing.T) {
	t.Parallel()
	retriever := NewMemoryRetriever(
		&stubEmbeddingService{},
		&stubEmbeddingRepository{results: nil},
		&stubFactRepository{}, nil,
		NewDecayScorer(),
		nil,
		nil,
		nil,
	)

	memories, err := retriever.RetrieveForContext(context.Background(), "query", "session_001", 3)
	require.NoError(t, err)
	assert.Empty(t, memories)
}

func TestMemoryRetriever_TokenBudget(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()

	facts := map[string]*entity.ExtractedFact{}
	embeddings := []*entity.ScoredEmbedding(nil)

	// 创建 10 条事实，每条约 15 个字符
	for i := 0; i < 10; i++ {
		fid := fmt.Sprintf("fact_%d", i)
		facts[fid] = &entity.ExtractedFact{
			FactID: fid, Subject: "用户", Predicate: "服用", Object: fmt.Sprintf("药品%d", i),
			Confidence: 0.9, Status: entity.FactStatusApproved, CreatedAt: now,
		}
		embeddings = append(embeddings, &entity.ScoredEmbedding{
			SemanticEmbedding: &entity.SemanticEmbedding{FactID: fid},
			Similarity:        float64(10-i) / 10.0,
		})
	}

	retriever := NewMemoryRetriever(
		&stubEmbeddingService{},
		&stubEmbeddingRepository{results: embeddings},
		&stubFactRepository{facts: facts}, nil,
		NewDecayScorer(),
		nil,
		nil,
		nil,
	)
	retriever.tokenBudget = 50 // 很小的预算，约能容纳 3 条

	memories, err := retriever.RetrieveForContext(context.Background(), "query", "session_001", 10)
	require.NoError(t, err)
	// tokenBudget 限制下，返回的记忆数量应受限
	assert.LessOrEqual(t, len(memories), 5)
}

func TestMemoryRetriever_SetEnabled(t *testing.T) {
	t.Parallel()
	retriever := NewMemoryRetriever(&stubEmbeddingService{}, &stubEmbeddingRepository{}, &stubFactRepository{}, nil, NewDecayScorer(), nil, nil, nil)
	assert.True(t, retriever.IsEnabled())

	retriever.SetEnabled(false)
	assert.False(t, retriever.IsEnabled())

	retriever.SetEnabled(true)
	assert.True(t, retriever.IsEnabled())
}

func TestMemoryRetriever_SetSessionEnabled(t *testing.T) {
	t.Parallel()
	retriever := NewMemoryRetriever(&stubEmbeddingService{}, &stubEmbeddingRepository{}, &stubFactRepository{}, nil, NewDecayScorer(), nil, nil, nil)

	// 全局开启，会话默认开启
	assert.True(t, retriever.IsSessionEnabled("sess_1"))

	// 禁用特定会话
	retriever.SetSessionEnabled("sess_1", false)
	assert.False(t, retriever.IsSessionEnabled("sess_1"))
	assert.True(t, retriever.IsSessionEnabled("sess_2"))

	// 重新启用
	retriever.SetSessionEnabled("sess_1", true)
	assert.True(t, retriever.IsSessionEnabled("sess_1"))

	// 空 sessionID 等效于全局开关
	retriever.SetSessionEnabled("", false)
	assert.False(t, retriever.IsEnabled())
}

func TestMemoryRetriever_detectEntityMentions(t *testing.T) {
	t.Parallel()
	factRepo := &stubFactRepositoryWithSubjects{
		subjects: []string{"用户", "医生"},
		bySubject: map[string][]*entity.ExtractedFact{
			"用户": {
				{FactID: "f1", Subject: "用户", Predicate: "患有", Object: "高血压", Confidence: 0.9, Status: entity.FactStatusApproved},
			},
		},
		facts: []*entity.ExtractedFact{
			{FactID: "f1", Subject: "用户", Predicate: "患有", Object: "高血压", Confidence: 0.9, Status: entity.FactStatusApproved},
		},
	}
	retriever := NewMemoryRetriever(&stubEmbeddingService{}, &stubEmbeddingRepository{}, factRepo, nil, NewDecayScorer(), nil, nil, nil)

	memories, triggered := retriever.detectEntityMentions(context.Background(), "用户最近血压怎么样")
	assert.True(t, triggered)
	require.Len(t, memories, 1)
	assert.Equal(t, "用户 患有 高血压", memories[0].Content)
}

func TestMemoryRetriever_detectEntityMentions_NoMatch(t *testing.T) {
	t.Parallel()
	factRepo := &stubFactRepositoryWithSubjects{subjects: []string{"用户"}}
	retriever := NewMemoryRetriever(&stubEmbeddingService{}, &stubEmbeddingRepository{}, factRepo, nil, NewDecayScorer(), nil, nil, nil)

	memories, triggered := retriever.detectEntityMentions(context.Background(), "今天天气不错")
	assert.False(t, triggered)
	assert.Len(t, memories, 0)
}

func TestMemoryRetriever_detectEntityMentions_KeywordMatch(t *testing.T) {
	t.Parallel()
	// 测试 predicate/object 关键词匹配（新增能力）
	factRepo := &stubFactRepositoryWithSubjects{
		facts: []*entity.ExtractedFact{
			{FactID: "f1", Subject: "用户", Predicate: "体重是", Object: "110公斤", Confidence: 0.9, Status: entity.FactStatusApproved},
			{FactID: "f2", Subject: "用户", Predicate: "患有", Object: "高血压", Confidence: 0.85, Status: entity.FactStatusApproved},
		},
	}
	retriever := NewMemoryRetriever(&stubEmbeddingService{}, &stubEmbeddingRepository{}, factRepo, nil, NewDecayScorer(), nil, nil, nil)

	// "体重" 匹配 predicate "体重是"
	memories, triggered := retriever.detectEntityMentions(context.Background(), "我体重多少")
	assert.True(t, triggered)
	require.Len(t, memories, 1)
	assert.Equal(t, "用户 体重是 110公斤", memories[0].Content)

	// "血压" 匹配 object "高血压"
	memories, triggered = retriever.detectEntityMentions(context.Background(), "我血压怎么样")
	assert.True(t, triggered)
	require.Len(t, memories, 1)
	assert.Equal(t, "用户 患有 高血压", memories[0].Content)
}

func TestFormatMemoriesForInjection(t *testing.T) {
	t.Parallel()
	memories := []*entity.HealthMemory{
		{Content: "用户 患有 高血压", Confidence: 0.9},
		{Content: "用户 服用 降压药", Confidence: 0.85},
	}
	result := FormatMemoriesForInjection(memories)
	assert.Contains(t, result, "[相关记忆]")
	assert.Contains(t, result, "1. 用户 患有 高血压")
	assert.Contains(t, result, "2. 用户 服用 降压药")
}

func TestFormatMemoriesForInjection_Empty(t *testing.T) {
	t.Parallel()
	result := FormatMemoriesForInjection(nil)
	assert.Equal(t, "", result)
}

func TestMemoryRetriever_retrieveSemantic_error(t *testing.T) {
	t.Parallel()
	// 当 embeddingRepo.SearchSimilar 返回错误时，semanticSearch 应正确返回错误
	retriever := &MemoryRetriever{
		embeddingSvc:  &stubEmbeddingService{vectors: [][]float32{{1, 2, 3}}},
		embeddingRepo: &stubEmbeddingRepository{err: fmt.Errorf("search failed")},
		factRepo:      &stubFactRepository{},
		decayScorer:   NewDecayScorer(),
		expansionSvc:  NewQueryExpansionService(),
	}

	_, err := retriever.semanticSearch(context.Background(), []float32{1, 2, 3}, 3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search failed")
}

func TestMemoryRetriever_semanticSearchFiltersAndSorts(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	facts := map[string]*entity.ExtractedFact{
		"fact_a": {
			FactID: "fact_a", Subject: "用户", Predicate: "体重是", Object: "110kg",
			Confidence: 0.9, Status: entity.FactStatusApproved, CreatedAt: now,
		},
		"fact_b": {
			FactID: "fact_b", Subject: "用户", Predicate: "身高是", Object: "180cm",
			Confidence: 0.8, Status: entity.FactStatusApproved, CreatedAt: now,
		},
		"fact_low": {
			FactID: "fact_low", Subject: "用户", Predicate: "血压是", Object: "120/80",
			Confidence: 0.5, Status: entity.FactStatusApproved, CreatedAt: now,
		},
		"fact_pending": {
			FactID: "fact_pending", Subject: "用户", Predicate: "患有", Object: "高血压",
			Confidence: 0.95, Status: entity.FactStatusPending, CreatedAt: now,
		},
	}
	embeddings := []*entity.ScoredEmbedding{
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_a"}, Similarity: 0.7},
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_b"}, Similarity: 0.95},
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_low"}, Similarity: 0.9},
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_pending"}, Similarity: 0.99},
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "missing"}, Similarity: 0.99},
	}
	migrationState := NewMigrationState()
	migrationState.SetComplete(true)
	retriever := NewMemoryRetriever(
		&stubEmbeddingService{},
		&stubEmbeddingRepository{results: embeddings},
		&stubFactRepository{facts: facts}, nil,
		NewDecayScorer(),
		migrationState,
		nil,
		nil,
	)

	memories, err := retriever.semanticSearch(context.Background(), []float32{1, 2, 3}, 2)
	require.NoError(t, err)
	require.Len(t, memories, 2)
	assert.Equal(t, models.MemoryID("fact_b"), memories[0].ID)
	assert.Equal(t, models.MemoryID("fact_a"), memories[1].ID)
	assert.Contains(t, memories[0].Content, "身高是")
	assert.Greater(t, memories[0].Confidence, memories[1].Confidence)
}

func TestMemoryRetriever_mergeMemories_sessionGap(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	factRepo := &stubFactRepository{
		findBySessionFunc: func(_ context.Context, _ string) ([]*entity.ExtractedFact, error) {
			return []*entity.ExtractedFact{
				{
					FactID: "m1", Subject: "用户", Predicate: "体重是", Object: "110kg",
					Confidence: 0.9, Status: entity.FactStatusApproved, CreatedAt: now,
				},
				{
					FactID: "gap", Subject: "用户", Predicate: "对...过敏", Object: "青霉素",
					Confidence: 0.95, Status: entity.FactStatusApproved, CreatedAt: now,
				},
				{
					FactID: "pending", Subject: "用户", Predicate: "患有", Object: "高血压",
					Confidence: 0.95, Status: entity.FactStatusPending, CreatedAt: now,
				},
			}, nil
		},
	}
	retriever := NewMemoryRetriever(&stubEmbeddingService{}, &stubEmbeddingRepository{}, factRepo, nil, NewDecayScorer(), nil, nil, nil)

	mentionMemories := []*entity.HealthMemory{
		{ID: "m1", Content: "mention 1"},
		{ID: "m2", Content: "mention 2"},
	}
	semanticMemories := []*entity.HealthMemory{
		{ID: "m2", Content: "semantic 2"},
		{ID: "gap", Content: "semantic gap"},
		{ID: "s3", Content: "semantic 3"},
	}

	result := retriever.mergeMemories(mentionMemories, semanticMemories, true, "test-session")

	require.Len(t, result, 4)
	assert.Equal(t, "mention 1", result[0].Content)
	assert.Equal(t, "mention 2", result[1].Content)
	assert.Equal(t, "用户 对...过敏 青霉素", result[2].Content)
	assert.Equal(t, "semantic 3", result[3].Content)
}

func TestMemoryRetriever_checkSessionGap(t *testing.T) {
	t.Parallel()
	retriever := NewMemoryRetriever(&stubEmbeddingService{}, &stubEmbeddingRepository{}, &stubFactRepository{}, nil, NewDecayScorer(), nil, nil, nil)

	// 空 sessionID 应返回 false
	assert.False(t, retriever.checkSessionGap(""))

	// 首次访问应返回 false
	assert.False(t, retriever.checkSessionGap("sess_1"))

	// 记录访问时间
	retriever.recordSessionAccess("sess_1")

	// 10 分钟内再次访问应返回 false
	assert.False(t, retriever.checkSessionGap("sess_1"))

	// 模拟超过 10 分钟前的访问时间
	retriever.mu.Lock()
	retriever.sessionAccessTimes["sess_1"] = time.Now().UTC().Add(-11 * time.Minute)
	retriever.mu.Unlock()

	// 超过 10 分钟应返回 true
	assert.True(t, retriever.checkSessionGap("sess_1"))
}

func TestMemoryRetriever_recordSessionAccess(t *testing.T) {
	t.Parallel()
	retriever := NewMemoryRetriever(&stubEmbeddingService{}, &stubEmbeddingRepository{}, &stubFactRepository{}, nil, NewDecayScorer(), nil, nil, nil)

	sessionID := "sess_test"
	retriever.recordSessionAccess(sessionID)

	retriever.mu.RLock()
	accessTime, ok := retriever.sessionAccessTimes[sessionID]
	retriever.mu.RUnlock()

	require.True(t, ok, "应记录会话访问时间")
	assert.WithinDuration(t, time.Now().UTC(), accessTime, 2*time.Second)
}

func TestMemoryRetriever_recallBySessionGap(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	facts := map[string]*entity.ExtractedFact{
		"fact_bp": {
			FactID: "fact_bp", Subject: "用户", Predicate: "患有", Object: "高血压",
			Confidence: 0.9, Status: entity.FactStatusApproved, CreatedAt: now,
		},
		"fact_weight": {
			FactID: "fact_weight", Subject: "用户", Predicate: "体重是", Object: "70公斤",
			Confidence: 0.85, Status: entity.FactStatusApproved, CreatedAt: now,
		},
	}
	factRepo := &stubFactRepository{
		facts: facts,
		findBySessionFunc: func(_ context.Context, _ string) ([]*entity.ExtractedFact, error) {
			return []*entity.ExtractedFact{facts["fact_bp"], facts["fact_weight"]}, nil
		},
	}
	retriever := NewMemoryRetriever(&stubEmbeddingService{}, &stubEmbeddingRepository{}, factRepo, nil, NewDecayScorer(), nil, nil, nil)

	candidates, status := retriever.recallBySessionGap(context.Background(), "sess_gap_1")
	require.Equal(t, "success", status.Status)
	require.Len(t, candidates, 2)
	assert.Contains(t, candidates[0].Content, "高血压")
	assert.Equal(t, []RetrievalPath{PathSessionGap}, candidates[0].MatchedPaths)
}

func TestMemoryRetriever_ArchiveConversation(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	facts := map[string]*entity.ExtractedFact{
		"fact_bp": {
			FactID: "fact_bp", Subject: "用户", Predicate: "患有", Object: "高血压",
			Confidence: 0.9, Status: entity.FactStatusApproved, CreatedAt: now,
		},
	}
	factRepo := &stubFactRepository{
		facts: facts,
		findBySessionFunc: func(_ context.Context, _ string) ([]*entity.ExtractedFact, error) {
			return []*entity.ExtractedFact{facts["fact_bp"]}, nil
		},
	}
	memRepo := &stubMemoryRepository{}
	retriever := NewMemoryRetriever(&stubEmbeddingService{}, &stubEmbeddingRepository{}, factRepo, memRepo, NewDecayScorer(), nil, nil, nil)

	err := retriever.ArchiveConversation(context.Background(), "conv_archive_1")
	require.NoError(t, err)
	require.Len(t, memRepo.saved, 1)
	assert.Equal(t, entity.TierShortTerm, memRepo.saved[0].Tier)
	assert.Contains(t, memRepo.saved[0].Content, "高血压")
	assert.Equal(t, models.ConversationID("conv_archive_1"), memRepo.saved[0].SourceConv)
}

func TestMemoryRetriever_ArchiveConversation_NoMemoryRepo(t *testing.T) {
	t.Parallel()
	retriever := NewMemoryRetriever(&stubEmbeddingService{}, &stubEmbeddingRepository{}, &stubFactRepository{}, nil, NewDecayScorer(), nil, nil, nil)
	err := retriever.ArchiveConversation(context.Background(), "conv_x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "memory repository not available")
}

func TestMemoryRetriever_detectEntityMentions_QueryHowManyJinMatchesWeightFact(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	facts := []*entity.ExtractedFact{
		{
			FactID: "fact_weight", Subject: "用户", Predicate: "体重是", Object: "110公斤",
			Confidence: 0.9, Status: entity.FactStatusApproved, CreatedAt: now,
		},
	}

	retriever := NewMemoryRetriever(
		&stubEmbeddingService{},
		&stubEmbeddingRepository{},
		&stubFactRepositoryWithWeightFacts{facts: facts}, nil,
		NewDecayScorer(),
		nil,
		nil,
		nil,
	)

	matched, ok := retriever.detectEntityMentions(context.Background(), "我多少斤")
	require.True(t, ok, "同义问法应能召回体重事实")
	require.Len(t, matched, 1)
	// 返回给 LLM 上下文的必须是原始事实文本，不能是 retrieval text
	assert.Equal(t, "用户 体重是 110公斤", matched[0].Content)
}

// stubFactRepositoryWithWeightFacts 用于体重同义词召回测试

type stubFactRepositoryWithWeightFacts struct {
	stubFactRepository
	facts []*entity.ExtractedFact
}

func (s *stubFactRepositoryWithWeightFacts) ListByStatus(_ context.Context, status entity.FactStatus, _ int, _ int) ([]*entity.ExtractedFact, error) {
	var result []*entity.ExtractedFact
	for _, f := range s.facts {
		if f.Status == status {
			result = append(result, f)
		}
	}
	return result, nil
}

// ========== 混合检索管线测试 ==========

func TestRetrieveWithDiagnostics_IntentPath(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()

	facts := map[string]*entity.ExtractedFact{
		"fact_weight": {
			FactID: "fact_weight", Subject: "用户", Predicate: "体重是", Object: "70公斤",
			Confidence: 0.9, Status: entity.FactStatusApproved, CreatedAt: now,
		},
	}

	factRepo := &stubFactRepository{facts: facts}
	factRepo.approvedByPredicatesFunc = func(_ context.Context, _ string, predicates []string, _ int) ([]*entity.ExtractedFact, error) {
		if len(predicates) > 0 && predicates[0] == "体重是" {
			return []*entity.ExtractedFact{facts["fact_weight"]}, nil
		}
		return nil, nil
	}

	expansionSvc := NewQueryExpansionService()
	intentResolver := NewIntentResolver(expansionSvc)

	retriever := NewMemoryRetriever(
		&stubEmbeddingService{},
		&stubEmbeddingRepository{},
		factRepo, nil,
		NewDecayScorer(),
		nil,
		intentResolver,
		expansionSvc,
	)

	diag, memories, err := retriever.retrieveWithDiagnostics(context.Background(), "我多少斤", "session_001", 3)
	require.NoError(t, err)

	// 意图召回应命中
	assert.NotEmpty(t, diag.IntentCandidates)
	assert.Equal(t, "fact_weight", diag.IntentCandidates[0].FactID)
	assert.Contains(t, diag.IntentCandidates[0].MatchedPaths, PathIntent)

	// 诊断字段应非空
	assert.NotNil(t, diag.DetectedIntent)
	assert.Equal(t, ConfidenceHigh, diag.DetectedIntent.Confidence)
	assert.NotEmpty(t, diag.PathStatuses)

	_ = memories
}

func TestIntentConfidenceToLevel_AllLevels(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 3, intentConfidenceToLevel(ConfidenceHigh))
	assert.Equal(t, 2, intentConfidenceToLevel(ConfidenceMedium))
	assert.Equal(t, 1, intentConfidenceToLevel(ConfidenceLow))
	assert.Equal(t, 0, intentConfidenceToLevel(IntentConfidence(0)))
}

func TestRetrieveWithDiagnostics_PureSemanticPath(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()

	facts := map[string]*entity.ExtractedFact{
		"fact_allergy": {
			FactID: "fact_allergy", Subject: "用户", Predicate: "对...过敏", Object: "青霉素",
			Confidence: 0.95, Status: entity.FactStatusApproved, CreatedAt: now,
		},
	}

	embeddings := []*entity.ScoredEmbedding{
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_allergy"}, Similarity: 0.9},
	}

	factRepo := &stubFactRepositoryWithSubjects{
		stubFactRepository: stubFactRepository{facts: facts},
		facts:              []*entity.ExtractedFact{facts["fact_allergy"]},
	}
	retriever := NewMemoryRetriever(
		&stubEmbeddingService{},
		&stubEmbeddingRepository{results: embeddings},
		factRepo, nil,
		NewDecayScorer(),
		nil,
		nil,
		nil,
	)

	diag, memories, err := retriever.retrieveWithDiagnostics(context.Background(), "注射前需要提醒医生哪些个人情况", "session_semantic", 3)
	require.NoError(t, err)

	assert.Nil(t, diag.DetectedIntent)
	assert.Empty(t, diag.IntentCandidates)
	assert.Empty(t, diag.KeywordCandidates)
	require.NotEmpty(t, diag.VectorCandidates)
	assert.Equal(t, "fact_allergy", diag.VectorCandidates[0].FactID)
	require.NotEmpty(t, diag.SelectedMemories)
	assert.Equal(t, "fact_allergy", diag.SelectedMemories[0].FactID)
	require.NotEmpty(t, memories)
	assert.Equal(t, models.MemoryID("fact_allergy"), memories[0].ID)
}

func TestRetrieveWithDiagnostics_AllPathsFailGracefully(t *testing.T) {
	t.Parallel()
	// 无 embedding 结果、无 fact、无 intent → 各路径全部空，应优雅返回空
	retriever := NewMemoryRetriever(
		&stubEmbeddingService{},
		&stubEmbeddingRepository{results: nil},
		&stubFactRepository{}, nil,
		NewDecayScorer(),
		nil,
		nil,
		nil,
	)

	diag, memories, err := retriever.retrieveWithDiagnostics(context.Background(), "随便问", "session_001", 3)
	require.NoError(t, err)
	assert.Empty(t, memories)

	// 应有 5 条 PathStatus（intent/keyword/vector/recent/session_gap）
	assert.Len(t, diag.PathStatuses, 5)

	// 汇总应为 0
	assert.Equal(t, 0, diag.TotalApprovedFacts)
	assert.Equal(t, 0, diag.TotalRejected)
}

func TestMergeCandidates_DedupAcrossPaths(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()

	// 同一个 fact 从 intent 和 keyword 两路命中
	intentCands := []RetrievalCandidate{
		{
			FactID: "fact_shared", Content: "用户 血压偏高 收缩压140",
			Snippet: "用户 血压偏高...", CreatedAt: now, Confidence: 0.9,
			MatchedPaths: []RetrievalPath{PathIntent},
			IntentLevel:  3, RecencyScore: 0.9,
			Reasons: []string{"intent: blood_pressure"},
		},
	}

	keywordCands := []RetrievalCandidate{
		{
			FactID: "fact_shared", Content: "用户 血压偏高 收缩压140",
			Snippet: "用户 血压偏高...", CreatedAt: now, Confidence: 0.9,
			MatchedPaths: []RetrievalPath{PathKeyword},
			KeywordScore: 0.85, RecencyScore: 0.9,
			Reasons: []string{"keyword: 血压"},
		},
	}

	merged := mergeCandidates(intentCands, keywordCands)
	require.Len(t, merged, 1)

	// 应合并 matched_paths
	assert.Len(t, merged[0].MatchedPaths, 2)
	assert.Contains(t, merged[0].MatchedPaths, PathIntent)
	assert.Contains(t, merged[0].MatchedPaths, PathKeyword)

	// 应保留最高 IntentLevel
	assert.Equal(t, 3, merged[0].IntentLevel)

	// 应保留最高 KeywordScore
	assert.Equal(t, 0.85, merged[0].KeywordScore)

	// 应合并 reasons
	assert.Len(t, merged[0].Reasons, 2)
}

func TestRerank_IntentLevelPriority(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()

	candidates := []RetrievalCandidate{
		{FactID: "f_low", Content: "low", Snippet: "low", CreatedAt: now, IntentLevel: 1, VectorSimilarity: 0.95, RecencyScore: 1.0},
		{FactID: "f_high", Content: "high", Snippet: "high", CreatedAt: now, IntentLevel: 3, VectorSimilarity: 0.5, RecencyScore: 1.0},
	}

	req := &RetrievalRequest{
		Intent: &IntentResult{Confidence: ConfidenceHigh},
	}

	sorted := rerank(candidates, req)
	require.Len(t, sorted, 2)

	// intent_level 高的应排前面
	assert.Equal(t, "f_high", sorted[0].FactID)
	assert.Equal(t, "f_low", sorted[1].FactID)
}

func TestRerank_RecencyOverVectorSimilarity(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()

	candidates := []RetrievalCandidate{
		{FactID: "f_stale_highvec", Content: "stale", Snippet: "stale", CreatedAt: now,
			VectorSimilarity: 0.99, RecencyScore: 0.2},
		{FactID: "f_fresh_lowvec", Content: "fresh", Snippet: "fresh", CreatedAt: now,
			VectorSimilarity: 0.5, RecencyScore: 0.95},
	}

	sorted := rerank(candidates, nil)
	require.Len(t, sorted, 2)

	// recency 高的应排前面，即使 vector_similarity 低
	assert.Equal(t, "f_fresh_lowvec", sorted[0].FactID)
	assert.Equal(t, "f_stale_highvec", sorted[1].FactID)
}

func TestBuildExpandedQuery_Basic(t *testing.T) {
	t.Parallel()
	// 无 intent 时仅返回 normalized
	result := BuildExpandedQuery("血压偏高", nil)
	assert.Equal(t, "血压偏高", result)
}

func TestBuildExpandedQuery_WithPredicates(t *testing.T) {
	t.Parallel()
	intent := &IntentResult{
		Intent:     "blood_pressure",
		Confidence: ConfidenceHigh,
		Predicates: []string{"血压偏高", "血压异常"},
	}

	result := BuildExpandedQuery("血压偏高", intent)
	assert.Contains(t, result, "血压偏高")
	assert.Contains(t, result, "血压异常")
}

func TestBuildExpandedQuery_EmptyInput(t *testing.T) {
	t.Parallel()
	result := BuildExpandedQuery("", nil)
	assert.Equal(t, "", result)
}

func TestRetrieveWithDiagnostics_DiagnosticsFields(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()

	facts := map[string]*entity.ExtractedFact{
		"fact_a": {
			FactID: "fact_a", Subject: "用户", Predicate: "服用", Object: "维生素",
			Confidence: 0.9, Status: entity.FactStatusApproved, CreatedAt: now,
		},
	}

	embeddings := []*entity.ScoredEmbedding{
		{SemanticEmbedding: &entity.SemanticEmbedding{FactID: "fact_a"}, Similarity: 0.9},
	}

	retriever := NewMemoryRetriever(
		&stubEmbeddingService{},
		&stubEmbeddingRepository{results: embeddings},
		&stubFactRepository{facts: facts}, nil,
		NewDecayScorer(),
		nil,
		nil,
		nil,
	)

	diag, memories, err := retriever.retrieveWithDiagnostics(context.Background(), "query", "session_test", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, memories)

	// 验证诊断字段
	assert.NotNil(t, diag)
	assert.Equal(t, "query", diag.ExpandedQuery)
	assert.NotEmpty(t, diag.VectorCandidates)
	assert.NotEmpty(t, diag.MergedCandidates)
	assert.NotEmpty(t, diag.SelectedMemories)
	assert.Equal(t, 1, diag.TotalApprovedFacts)

	// 验证 rejected 在 token budget 截断内
	assert.GreaterOrEqual(t, diag.TotalRejected, 0)
}
