package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEmbeddingTestDB(t *testing.T) (*EmbeddingRepoSQLite, *FactRepoSQLite, func()) {
	tmpDir := t.TempDir()
	connector, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	err = connector.Migrate(ctx)
	require.NoError(t, err)

	embeddingRepo := NewEmbeddingRepoSQLite(connector)
	factRepo := NewFactRepoSQLite(connector)
	cleanup := func() {
		connector.Close()
	}
	return embeddingRepo, factRepo, cleanup
}

func TestEmbeddingRepo_SaveAndGet(t *testing.T) {
	repo, factRepo, cleanup := setupEmbeddingTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 先插入事实（外键约束）
	f := entity.NewExtractedFact("用户", "患有", "头痛", 0.8, []string{"msg_001"})
	f.FactID = "fact_e1"
	require.NoError(t, factRepo.Save(ctx, f))

	vector := make([]float32, entity.EmbeddingDimension)
	for i := range vector {
		vector[i] = float32(i) * 0.001
	}
	e := entity.NewSemanticEmbedding("fact_e1", vector, "all-MiniLM-L6-v2")
	err := repo.Save(ctx, e)
	require.NoError(t, err)

	got, err := repo.GetByFactID(ctx, "fact_e1")
	require.NoError(t, err)
	assert.Equal(t, e.EmbeddingID, got.EmbeddingID)
	assert.Equal(t, "fact_e1", got.FactID)
	assert.Equal(t, entity.EmbeddingDimension, len(got.Vector))
	assert.Equal(t, "all-MiniLM-L6-v2", got.ModelVersion)
	assert.Equal(t, vector, got.Vector)
}

func TestEmbeddingRepo_GetByFactID_NotFound(t *testing.T) {
	repo, _, cleanup := setupEmbeddingTestDB(t)
	defer cleanup()
	ctx := context.Background()

	_, err := repo.GetByFactID(ctx, "nonexistent")
	assert.ErrorIs(t, err, entity.ErrEmbeddingNotFound)
}

func TestEmbeddingRepo_DeleteByFactID(t *testing.T) {
	repo, factRepo, cleanup := setupEmbeddingTestDB(t)
	defer cleanup()
	ctx := context.Background()

	f := entity.NewExtractedFact("用户", "患有", "头痛", 0.8, []string{"msg_002"})
	f.FactID = "fact_e2"
	require.NoError(t, factRepo.Save(ctx, f))

	vector := make([]float32, entity.EmbeddingDimension)
	e := entity.NewSemanticEmbedding("fact_e2", vector, "all-MiniLM-L6-v2")
	require.NoError(t, repo.Save(ctx, e))

	err := repo.DeleteByFactID(ctx, "fact_e2")
	require.NoError(t, err)

	_, err = repo.GetByFactID(ctx, "fact_e2")
	assert.ErrorIs(t, err, entity.ErrEmbeddingNotFound)
}

func TestEmbeddingRepo_ForeignKeyCascade(t *testing.T) {
	repo, factRepo, cleanup := setupEmbeddingTestDB(t)
	defer cleanup()
	ctx := context.Background()

	f := entity.NewExtractedFact("用户", "患有", "头痛", 0.8, []string{"msg_003"})
	f.FactID = "fact_e3"
	require.NoError(t, factRepo.Save(ctx, f))

	vector := make([]float32, entity.EmbeddingDimension)
	e := entity.NewSemanticEmbedding("fact_e3", vector, "all-MiniLM-L6-v2")
	require.NoError(t, repo.Save(ctx, e))

	// 删除事实，外键级联删除嵌入
	err := factRepo.Delete(ctx, "fact_e3")
	require.NoError(t, err)

	_, err = repo.GetByFactID(ctx, "fact_e3")
	assert.ErrorIs(t, err, entity.ErrEmbeddingNotFound)
}

func TestEmbeddingRepo_SearchSimilar(t *testing.T) {
	repo, factRepo, cleanup := setupEmbeddingTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 准备 4 个事实，每个关联不同方向的向量
	facts := []*entity.ExtractedFact{
		entity.NewExtractedFact("用户", "患有", "A", 0.8, []string{"m1"}),
		entity.NewExtractedFact("用户", "患有", "B", 0.8, []string{"m2"}),
		entity.NewExtractedFact("用户", "患有", "C", 0.8, []string{"m3"}),
		entity.NewExtractedFact("用户", "患有", "D", 0.8, []string{"m4"}),
	}
	for i, f := range facts {
		f.FactID = fmt.Sprintf("fact_sim_%d", i)
		require.NoError(t, factRepo.Save(ctx, f))
	}

	// v1: [1, 0, 0, ...] — 与 query 完全一致
	v1 := make([]float32, entity.EmbeddingDimension)
	v1[0] = 1.0

	// v2: [0.9, 0.1, 0, ...] — 高度相似
	v2 := make([]float32, entity.EmbeddingDimension)
	v2[0] = 0.9
	v2[1] = 0.1

	// v3: [0, 1, 0, ...] — 正交
	v3 := make([]float32, entity.EmbeddingDimension)
	v3[1] = 1.0

	// v4: [-1, 0, 0, ...] — 相反
	v4 := make([]float32, entity.EmbeddingDimension)
	v4[0] = -1.0

	vectors := [][]float32{v1, v2, v3, v4}
	for i, v := range vectors {
		e := entity.NewSemanticEmbedding(facts[i].FactID, v, "all-MiniLM-L6-v2")
		require.NoError(t, repo.Save(ctx, e))
	}

	// query: [1, 0, 0, ...]
	query := make([]float32, entity.EmbeddingDimension)
	query[0] = 1.0

	// topK=2，应返回 v1、v2（按相似度降序）
	results, err := repo.SearchSimilar(ctx, query, 2)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "fact_sim_0", results[0].FactID) // v1 最相似
	assert.InDelta(t, 1.0, results[0].Similarity, 0.001)
	assert.Equal(t, "fact_sim_1", results[1].FactID) // v2 次相似
	assert.Greater(t, results[0].Similarity, results[1].Similarity)

	// topK=10，应返回全部 4 个，顺序为 v1 > v2 > v3 > v4
	results, err = repo.SearchSimilar(ctx, query, 10)
	require.NoError(t, err)
	require.Len(t, results, 4)
	assert.Equal(t, "fact_sim_0", results[0].FactID)
	assert.Equal(t, "fact_sim_1", results[1].FactID)
	assert.Equal(t, "fact_sim_2", results[2].FactID)
	assert.Equal(t, "fact_sim_3", results[3].FactID)
	// 验证相似度单调递减
	for i := 1; i < len(results); i++ {
		assert.GreaterOrEqual(t, results[i-1].Similarity, results[i].Similarity)
	}
}

func TestEmbeddingRepo_SearchSimilar_EmptyResult(t *testing.T) {
	repo, _, cleanup := setupEmbeddingTestDB(t)
	defer cleanup()
	ctx := context.Background()

	query := make([]float32, entity.EmbeddingDimension)
	query[0] = 1.0

	results, err := repo.SearchSimilar(ctx, query, 5)
	require.NoError(t, err)
	assert.Empty(t, results)
}
