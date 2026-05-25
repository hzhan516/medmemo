package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
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
