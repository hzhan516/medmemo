package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== Stub 实现 ==========

type stubEmbeddingService struct {
	vectors [][]float32
	err     error
}

func (s *stubEmbeddingService) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.vectors, nil
}

func (s *stubEmbeddingService) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	if s.err != nil {
		return nil, s.err
	}
	if len(s.vectors) > 0 {
		return s.vectors[0], nil
	}
	return make([]float32, entity.EmbeddingDimension), nil
}

type stubEmbeddingRepository struct {
	results []*entity.ScoredEmbedding
	err     error
}

func (s *stubEmbeddingRepository) Save(ctx context.Context, e *entity.SemanticEmbedding) error {
	return nil
}
func (s *stubEmbeddingRepository) GetByFactID(ctx context.Context, factID string) (*entity.SemanticEmbedding, error) {
	return nil, nil
}
func (s *stubEmbeddingRepository) DeleteByFactID(ctx context.Context, factID string) error {
	return nil
}
func (s *stubEmbeddingRepository) SearchSimilar(ctx context.Context, queryVector []float32, topK int) ([]*entity.ScoredEmbedding, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.results, nil
}

type stubFactRepository struct {
	facts map[string]*entity.ExtractedFact
}

func (s *stubFactRepository) Save(ctx context.Context, f *entity.ExtractedFact) error { return nil }
func (s *stubFactRepository) GetByID(ctx context.Context, factID string) (*entity.ExtractedFact, error) {
	f, ok := s.facts[factID]
	if !ok {
		return nil, entity.ErrFactNotFound
	}
	return f, nil
}
func (s *stubFactRepository) ListByStatus(ctx context.Context, status entity.FactStatus, offset, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (s *stubFactRepository) ListPending(ctx context.Context, offset, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (s *stubFactRepository) UpdateStatus(ctx context.Context, factID string, status entity.FactStatus) error {
	return nil
}
func (s *stubFactRepository) Delete(ctx context.Context, factID string) error { return nil }
func (s *stubFactRepository) GetStats(ctx context.Context) (total, approved, rejected, pending int64, err error) {
	return 0, 0, 0, 0, nil
}
func (s *stubFactRepository) ListAllSubjects(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (s *stubFactRepository) FindBySubject(ctx context.Context, subject string) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (s *stubFactRepository) FindBySession(ctx context.Context, sessionID string) ([]*entity.ExtractedFact, error) {
	return nil, nil
}

// stubFactRepositoryWithSubjects 支持实体提及检测的 stub
type stubFactRepositoryWithSubjects struct {
	stubFactRepository
	subjects  []string
	bySubject map[string][]*entity.ExtractedFact
	facts     []*entity.ExtractedFact
}

func (s *stubFactRepositoryWithSubjects) ListAllSubjects(ctx context.Context) ([]string, error) {
	return s.subjects, nil
}
func (s *stubFactRepositoryWithSubjects) FindBySubject(ctx context.Context, subject string) ([]*entity.ExtractedFact, error) {
	return s.bySubject[subject], nil
}
func (s *stubFactRepositoryWithSubjects) ListByStatus(ctx context.Context, status entity.FactStatus, offset, limit int) ([]*entity.ExtractedFact, error) {
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
		&stubFactRepository{facts: facts},
		NewDecayScorer(),
	)

	memories, err := retriever.RetrieveForContext(context.Background(), "我的血压怎么样", "session_001", 2)
	require.NoError(t, err)
	require.Len(t, memories, 2)

	assert.Equal(t, "用户 患有 高血压", memories[0].Content)
	assert.Equal(t, "用户 服用 降压药", memories[1].Content)
}

func TestMemoryRetriever_DecayRanking(t *testing.T) {
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
		&stubFactRepository{facts: facts},
		NewDecayScorer(),
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
		&stubFactRepository{facts: facts},
		NewDecayScorer(),
	)

	memories, err := retriever.RetrieveForContext(context.Background(), "query", "session_001", 10)
	require.NoError(t, err)
	require.Len(t, memories, 1)
	assert.Equal(t, "用户 患有 高血压", memories[0].Content)
}

func TestMemoryRetriever_MinConfidenceFilter(t *testing.T) {
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
		&stubFactRepository{facts: facts},
		NewDecayScorer(),
	)
	retriever.minConfidence = 0.6 // 设置较高阈值

	memories, err := retriever.RetrieveForContext(context.Background(), "query", "session_001", 10)
	require.NoError(t, err)
	require.Len(t, memories, 1)
	assert.Equal(t, "用户 患有 高血压", memories[0].Content)
}

func TestMemoryRetriever_EmbedFailure(t *testing.T) {
	retriever := NewMemoryRetriever(
		&stubEmbeddingService{err: fmt.Errorf("embedding failed")},
		&stubEmbeddingRepository{},
		&stubFactRepository{},
		NewDecayScorer(),
	)

	// 嵌入失败时应返回空结果而非报错（降级）
	memories, err := retriever.RetrieveForContext(context.Background(), "query", "session_001", 3)
	require.NoError(t, err)
	assert.Empty(t, memories)
}

func TestMemoryRetriever_NoResults(t *testing.T) {
	retriever := NewMemoryRetriever(
		&stubEmbeddingService{},
		&stubEmbeddingRepository{results: nil},
		&stubFactRepository{},
		NewDecayScorer(),
	)

	memories, err := retriever.RetrieveForContext(context.Background(), "query", "session_001", 3)
	require.NoError(t, err)
	assert.Empty(t, memories)
}

func TestMemoryRetriever_TokenBudget(t *testing.T) {
	now := time.Now().UTC()

	facts := map[string]*entity.ExtractedFact{}
	embeddings := []*entity.ScoredEmbedding{}

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
		&stubFactRepository{facts: facts},
		NewDecayScorer(),
	)
	retriever.tokenBudget = 50 // 很小的预算，约能容纳 3 条

	memories, err := retriever.RetrieveForContext(context.Background(), "query", "session_001", 10)
	require.NoError(t, err)
	// tokenBudget 限制下，返回的记忆数量应受限
	assert.LessOrEqual(t, len(memories), 5)
}

func TestMemoryRetriever_SetEnabled(t *testing.T) {
	retriever := NewMemoryRetriever(&stubEmbeddingService{}, &stubEmbeddingRepository{}, &stubFactRepository{}, NewDecayScorer())
	assert.True(t, retriever.IsEnabled())

	retriever.SetEnabled(false)
	assert.False(t, retriever.IsEnabled())

	retriever.SetEnabled(true)
	assert.True(t, retriever.IsEnabled())
}

func TestMemoryRetriever_SetSessionEnabled(t *testing.T) {
	retriever := NewMemoryRetriever(&stubEmbeddingService{}, &stubEmbeddingRepository{}, &stubFactRepository{}, NewDecayScorer())

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
	retriever := NewMemoryRetriever(&stubEmbeddingService{}, &stubEmbeddingRepository{}, factRepo, NewDecayScorer())

	memories, triggered := retriever.detectEntityMentions(context.Background(), "用户最近血压怎么样")
	assert.True(t, triggered)
	require.Len(t, memories, 1)
	assert.Equal(t, "用户 患有 高血压", memories[0].Content)
}

func TestMemoryRetriever_detectEntityMentions_NoMatch(t *testing.T) {
	factRepo := &stubFactRepositoryWithSubjects{subjects: []string{"用户"}}
	retriever := NewMemoryRetriever(&stubEmbeddingService{}, &stubEmbeddingRepository{}, factRepo, NewDecayScorer())

	memories, triggered := retriever.detectEntityMentions(context.Background(), "今天天气不错")
	assert.False(t, triggered)
	assert.Len(t, memories, 0)
}

func TestMemoryRetriever_detectEntityMentions_KeywordMatch(t *testing.T) {
	// 测试 predicate/object 关键词匹配（新增能力）
	factRepo := &stubFactRepositoryWithSubjects{
		facts: []*entity.ExtractedFact{
			{FactID: "f1", Subject: "用户", Predicate: "体重是", Object: "110公斤", Confidence: 0.9, Status: entity.FactStatusApproved},
			{FactID: "f2", Subject: "用户", Predicate: "患有", Object: "高血压", Confidence: 0.85, Status: entity.FactStatusApproved},
		},
	}
	retriever := NewMemoryRetriever(&stubEmbeddingService{}, &stubEmbeddingRepository{}, factRepo, NewDecayScorer())

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
	result := FormatMemoriesForInjection(nil)
	assert.Equal(t, "", result)
}
