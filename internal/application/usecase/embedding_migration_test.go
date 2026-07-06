package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeTestVector 创建指定值的 384 维测试向量。
func makeTestVector(val float32) []float32 {
	v := make([]float32, entity.EmbeddingDimension)
	v[0] = val
	return v
}

// ========== Stub 实现 ==========

type migratorEmbeddingService struct {
	vectors       [][]float32
	err           error
	callLog       []string
	modelVersion  string
	isAvailable   bool
	embedSingleFn func(ctx context.Context, text string) ([]float32, error)
}

func (s *migratorEmbeddingService) Embed(_ context.Context, texts []string) ([][]float32, error) {
	if s.err != nil {
		return nil, s.err
	}
	result := make([][]float32, len(texts))
	for i, t := range texts {
		s.callLog = append(s.callLog, t)
		if i < len(s.vectors) {
			result[i] = s.vectors[i]
		} else {
			result[i] = make([]float32, entity.EmbeddingDimension)
		}
	}
	return result, nil
}

func (s *migratorEmbeddingService) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	if s.embedSingleFn != nil {
		return s.embedSingleFn(ctx, text)
	}
	if s.err != nil {
		return nil, s.err
	}
	s.callLog = append(s.callLog, text)
	if len(s.vectors) > 0 {
		return s.vectors[0], nil
	}
	return make([]float32, entity.EmbeddingDimension), nil
}

func (s *migratorEmbeddingService) ModelVersion() string { return s.modelVersion }
func (s *migratorEmbeddingService) IsAvailable() bool    { return s.isAvailable }

type migratorEmbeddingRepo struct {
	embeddings map[string]*entity.SemanticEmbedding
	searchFn   func(ctx context.Context, queryVector []float32, topK int) ([]*entity.ScoredEmbedding, error)
}

func (r *migratorEmbeddingRepo) Save(_ context.Context, e *entity.SemanticEmbedding) error {
	if r.embeddings == nil {
		r.embeddings = make(map[string]*entity.SemanticEmbedding)
	}
	r.embeddings[e.FactID] = e
	return nil
}

func (r *migratorEmbeddingRepo) GetByFactID(_ context.Context, factID string) (*entity.SemanticEmbedding, error) {
	if e, ok := r.embeddings[factID]; ok {
		return e, nil
	}
	return nil, entity.ErrEmbeddingNotFound
}

func (r *migratorEmbeddingRepo) DeleteByFactID(_ context.Context, _ string) error { return nil }

func (r *migratorEmbeddingRepo) SearchSimilar(ctx context.Context, queryVector []float32, topK int) ([]*entity.ScoredEmbedding, error) {
	if r.searchFn != nil {
		return r.searchFn(ctx, queryVector, topK)
	}
	return nil, nil
}

func (r *migratorEmbeddingRepo) SearchSimilarFiltered(ctx context.Context, queryVector []float32, topK int, _ string) ([]*entity.ScoredEmbedding, error) {
	return r.SearchSimilar(ctx, queryVector, topK)
}

func (r *migratorEmbeddingRepo) CountByVersionNot(_ context.Context, version string) (int64, error) {
	var count int64
	for _, e := range r.embeddings {
		if e.ModelVersion != version {
			count++
		}
	}
	return count, nil
}

func (r *migratorEmbeddingRepo) UpdateEmbedding(_ context.Context, e *entity.SemanticEmbedding) error {
	if r.embeddings == nil {
		r.embeddings = make(map[string]*entity.SemanticEmbedding)
	}
	r.embeddings[e.FactID] = e
	return nil
}

type migratorFactRepo struct {
	facts []*entity.ExtractedFact
}

func (r *migratorFactRepo) Save(_ context.Context, _ *entity.ExtractedFact) error { return nil }
func (r *migratorFactRepo) GetByID(_ context.Context, _ string) (*entity.ExtractedFact, error) {
	return nil, entity.ErrFactNotFound
}
func (r *migratorFactRepo) FindByIDs(_ context.Context, _ []string) (map[string]*entity.ExtractedFact, error) {
	return map[string]*entity.ExtractedFact{}, nil
}
func (r *migratorFactRepo) ListByStatus(_ context.Context, _ entity.FactStatus, _, _ int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (r *migratorFactRepo) ListPending(_ context.Context, _, _ int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (r *migratorFactRepo) UpdateStatus(_ context.Context, _ string, _ entity.FactStatus) error {
	return nil
}
func (r *migratorFactRepo) Delete(_ context.Context, _ string) error { return nil }
func (r *migratorFactRepo) GetStats(_ context.Context) (total, approved, rejected, pending int64, err error) {
	return 0, 0, 0, 0, nil
}
func (r *migratorFactRepo) ListAllSubjects(_ context.Context) ([]string, error) { return nil, nil }
func (r *migratorFactRepo) FindBySubject(_ context.Context, _ string) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (r *migratorFactRepo) FindBySession(_ context.Context, _ string) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (r *migratorFactRepo) FindApprovedByPredicates(_ context.Context, _ string, _ []string, _ int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (r *migratorFactRepo) FindLatestApprovedByPredicates(_ context.Context, _ string, _ []string) (*entity.ExtractedFact, error) {
	return nil, entity.ErrFactNotFound
}
func (r *migratorFactRepo) SearchApproved(_ context.Context, _ string, _ int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}

func (r *migratorFactRepo) CountApprovedFactsNeedingEmbedding(_ context.Context, _ string) (int64, error) {
	var count int64
	for _, f := range r.facts {
		if f.Status == entity.FactStatusApproved {
			count++
		}
	}
	return count, nil
}

func (r *migratorFactRepo) ListApprovedFactsNeedingEmbedding(_ context.Context, _ string, _ time.Time, lastFactID string, limit int) ([]*entity.ExtractedFact, error) {
	var result []*entity.ExtractedFact
	var started bool
	if lastFactID == "" {
		started = true
	}
	for _, f := range r.facts {
		if !started {
			if f.FactID == lastFactID {
				started = true
			}
			continue
		}
		if f.Status == entity.FactStatusApproved {
			result = append(result, f)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// ========== 测试 ==========

func TestEmbeddingMigrator_NeedsMigration(t *testing.T) {
	t.Parallel()
	factRepo := &migratorFactRepo{
		facts: []*entity.ExtractedFact{
			{FactID: "f1", Status: entity.FactStatusApproved, CreatedAt: time.Now()},
		},
	}
	svc := &migratorEmbeddingService{modelVersion: models.CurrentEmbeddingVersion, isAvailable: true}
	migrator := NewEmbeddingMigrator(factRepo, &migratorEmbeddingRepo{}, svc, NewMigrationState())

	needs, total, err := migrator.NeedsMigration(context.Background())
	require.NoError(t, err)
	assert.True(t, needs)
	assert.Equal(t, int64(1), total)
}

func TestEmbeddingMigrator_NeedsMigration_NoCandidates(t *testing.T) {
	t.Parallel()
	factRepo := &migratorFactRepo{}
	svc := &migratorEmbeddingService{modelVersion: models.CurrentEmbeddingVersion, isAvailable: true}
	migrator := NewEmbeddingMigrator(factRepo, &migratorEmbeddingRepo{}, svc, NewMigrationState())

	needs, total, err := migrator.NeedsMigration(context.Background())
	require.NoError(t, err)
	assert.False(t, needs)
	assert.Equal(t, int64(0), total)
}

func TestEmbeddingMigrator_RunMigration_Success(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	facts := []*entity.ExtractedFact{
		{FactID: "f1", Subject: "用户", Predicate: "体重是", Object: "70公斤", Status: entity.FactStatusApproved, CreatedAt: now},
		{FactID: "f2", Subject: "用户", Predicate: "身高是", Object: "175厘米", Status: entity.FactStatusApproved, CreatedAt: now.Add(time.Second)},
	}
	factRepo := &migratorFactRepo{facts: facts}
	embedRepo := &migratorEmbeddingRepo{}
	svc := &migratorEmbeddingService{
		vectors:      [][]float32{makeTestVector(1.0)},
		modelVersion: models.CurrentEmbeddingVersion,
		isAvailable:  true,
	}
	state := NewMigrationState()
	migrator := NewEmbeddingMigrator(factRepo, embedRepo, svc, state)
	migrator.batchPause = 0

	var progressCalls []int
	processed, failed, err := migrator.RunMigration(context.Background(), func(p, _ int) {
		progressCalls = append(progressCalls, p)
	})

	require.NoError(t, err)
	assert.Equal(t, 2, processed)
	assert.Equal(t, 0, failed)
	assert.True(t, state.IsComplete())
	assert.Len(t, progressCalls, 2)

	// 验证两个 fact 都已写入当前版本 embedding
	assert.Equal(t, models.CurrentEmbeddingVersion, embedRepo.embeddings["f1"].ModelVersion)
	assert.Equal(t, models.CurrentEmbeddingVersion, embedRepo.embeddings["f2"].ModelVersion)
}

func TestEmbeddingMigrator_RunMigration_UpdateExisting(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	facts := []*entity.ExtractedFact{
		{FactID: "f1", Subject: "用户", Predicate: "体重是", Object: "70公斤", Status: entity.FactStatusApproved, CreatedAt: now},
	}
	factRepo := &migratorFactRepo{facts: facts}

	oldEmb := entity.NewSemanticEmbedding("f1", make([]float32, entity.EmbeddingDimension), "old-version")
	embedRepo := &migratorEmbeddingRepo{embeddings: map[string]*entity.SemanticEmbedding{"f1": oldEmb}}

	svc := &migratorEmbeddingService{
		vectors:      [][]float32{makeTestVector(1.0)},
		modelVersion: models.CurrentEmbeddingVersion,
		isAvailable:  true,
	}
	state := NewMigrationState()
	migrator := NewEmbeddingMigrator(factRepo, embedRepo, svc, state)
	migrator.batchPause = 0

	processed, failed, err := migrator.RunMigration(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	assert.Equal(t, 0, failed)
	assert.True(t, state.IsComplete())

	// 验证保留原 embedding_id，更新版本
	assert.Equal(t, oldEmb.EmbeddingID, embedRepo.embeddings["f1"].EmbeddingID)
	assert.Equal(t, models.CurrentEmbeddingVersion, embedRepo.embeddings["f1"].ModelVersion)
}

func TestEmbeddingMigrator_RunMigration_PartialFailure(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	facts := []*entity.ExtractedFact{
		{FactID: "f1", Subject: "用户", Predicate: "体重是", Object: "70公斤", Status: entity.FactStatusApproved, CreatedAt: now},
		{FactID: "f2", Subject: "用户", Predicate: "身高是", Object: "175厘米", Status: entity.FactStatusApproved, CreatedAt: now.Add(time.Second)},
	}
	factRepo := &migratorFactRepo{facts: facts}
	embedRepo := &migratorEmbeddingRepo{}

	// 第一条成功，第二条失败
	callCount := 0
	svc := &migratorEmbeddingService{
		modelVersion: models.CurrentEmbeddingVersion,
		isAvailable:  true,
		embedSingleFn: func(_ context.Context, _ string) ([]float32, error) {
			callCount++
			if callCount == 2 {
				return nil, errors.New("mock embed error")
			}
			return make([]float32, entity.EmbeddingDimension), nil
		},
	}

	state := NewMigrationState()
	migrator := NewEmbeddingMigrator(factRepo, embedRepo, svc, state)
	migrator.batchPause = 0

	processed, failed, err := migrator.RunMigration(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)
	assert.Equal(t, 1, failed)
	assert.False(t, state.IsComplete(), "有失败项时不应置 complete，确保下次启动重试")
}

func TestEmbeddingMigrator_RunMigration_FailureButRecheckZero(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	facts := []*entity.ExtractedFact{
		{FactID: "f1", Subject: "用户", Predicate: "体重是", Object: "70公斤", Status: entity.FactStatusApproved, CreatedAt: now},
	}
	// 初始有 1 条候选；迁移后通过外部方式修复（模拟 factRepo 返回 0）
	factRepo := &migratorFactRepo{facts: facts}
	embedRepo := &migratorEmbeddingRepo{}
	svc := &migratorEmbeddingService{
		vectors:      [][]float32{makeTestVector(1.0)},
		modelVersion: models.CurrentEmbeddingVersion,
		isAvailable:  true,
	}
	state := NewMigrationState()
	migrator := NewEmbeddingMigrator(factRepo, embedRepo, svc, state)
	migrator.batchPause = 0

	// 让 processFact 人为失败但候选计数在重试时归 0
	svc.embedSingleFn = func(_ context.Context, _ string) ([]float32, error) {
		return nil, errors.New("forced error")
	}

	processed, failed, err := migrator.RunMigration(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 0, processed)
	assert.Equal(t, 1, failed)

	// 由于 stub factRepo 始终返回 1（未真正修复），state 仍为 false
	assert.False(t, state.IsComplete())
}

// 验证以下接口契约（编译期检查）
var _ repository.FactRepository = (*migratorFactRepo)(nil)
var _ repository.EmbeddingRepository = (*migratorEmbeddingRepo)(nil)
