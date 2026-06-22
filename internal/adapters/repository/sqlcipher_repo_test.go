package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type repositorySecretStore struct {
	data map[string][]byte
}

func newRepositorySecretStore() *repositorySecretStore {
	return &repositorySecretStore{data: make(map[string][]byte)}
}

func (s *repositorySecretStore) Set(key string, value []byte) error {
	s.data[key] = value
	return nil
}

func (s *repositorySecretStore) Get(key string) ([]byte, error) {
	value, ok := s.data[key]
	if !ok {
		return nil, fmt.Errorf("secret not found: %s", key)
	}
	return value, nil
}

func (s *repositorySecretStore) Delete(key string) error {
	delete(s.data, key)
	return nil
}

func setupSQLCipherRepositoryTestDB(t *testing.T) (*FactRepoSQLite, *EmbeddingRepoSQLite, func()) {
	t.Helper()

	connector, err := database.NewSQLCipherConnector(t.TempDir(), newRepositorySecretStore())
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, connector.Migrate(ctx))

	cleanup := func() {
		require.NoError(t, connector.Close())
	}
	return NewFactRepoSQLite(connector), NewEmbeddingRepoSQLite(connector), cleanup
}

func TestRepositorySave_SQLCipherIdempotentCompatibility(t *testing.T) {
	factRepo, embeddingRepo, cleanup := setupSQLCipherRepositoryTestDB(t)
	defer cleanup()
	ctx := context.Background()

	f1 := entity.NewExtractedFact("用户", "患有", "偏头痛", 0.85, []string{"msg_001"})
	f1.FactID = "fact_sqlcipher_dup"
	require.NoError(t, factRepo.Save(ctx, f1))

	f2 := entity.NewExtractedFact("用户", "服用", "阿司匹林", 0.9, []string{"msg_002"})
	f2.FactID = "fact_sqlcipher_dup"
	require.NoError(t, factRepo.Save(ctx, f2))

	gotFact, err := factRepo.GetByID(ctx, "fact_sqlcipher_dup")
	require.NoError(t, err)
	assert.Equal(t, "患有", gotFact.Predicate)
	assert.Equal(t, "偏头痛", gotFact.Object)

	vector1 := make([]float32, entity.EmbeddingDimension)
	vector1[0] = 1
	e1 := entity.NewSemanticEmbedding("fact_sqlcipher_dup", vector1, "all-MiniLM-L6-v2")
	require.NoError(t, embeddingRepo.Save(ctx, e1))

	vector2 := make([]float32, entity.EmbeddingDimension)
	vector2[1] = 1
	e2 := entity.NewSemanticEmbedding("fact_sqlcipher_dup", vector2, "all-MiniLM-L6-v2")
	require.NoError(t, embeddingRepo.Save(ctx, e2))

	gotEmbedding, err := embeddingRepo.GetByFactID(ctx, "fact_sqlcipher_dup")
	require.NoError(t, err)
	assert.Equal(t, e1.EmbeddingID, gotEmbedding.EmbeddingID)
	assert.Equal(t, vector1, gotEmbedding.Vector)
}

func TestEmbeddingRepo_SearchSimilar_SQLCipherFallback(t *testing.T) {
	factRepo, embeddingRepo, cleanup := setupSQLCipherRepositoryTestDB(t)
	defer cleanup()
	ctx := context.Background()

	assert.False(t, embeddingRepo.vectorSQLAvailable)

	weightFact := entity.NewExtractedFact("用户", "体重是", "110公斤", 0.95, []string{"msg_weight"})
	weightFact.FactID = "fact_weight"
	weightFact.Status = entity.FactStatusApproved
	require.NoError(t, factRepo.Save(ctx, weightFact))

	bloodPressureFact := entity.NewExtractedFact("用户", "患有", "高血压", 0.9, []string{"msg_bp"})
	bloodPressureFact.FactID = "fact_bp"
	bloodPressureFact.Status = entity.FactStatusApproved
	require.NoError(t, factRepo.Save(ctx, bloodPressureFact))

	weightVector := make([]float32, entity.EmbeddingDimension)
	weightVector[0] = 1
	require.NoError(t, embeddingRepo.Save(ctx, entity.NewSemanticEmbedding(weightFact.FactID, weightVector, "all-MiniLM-L6-v2")))

	bloodPressureVector := make([]float32, entity.EmbeddingDimension)
	bloodPressureVector[1] = 1
	require.NoError(t, embeddingRepo.Save(ctx, entity.NewSemanticEmbedding(bloodPressureFact.FactID, bloodPressureVector, "all-MiniLM-L6-v2")))

	queryVector := make([]float32, entity.EmbeddingDimension)
	queryVector[0] = 1

	results, err := embeddingRepo.SearchSimilar(ctx, queryVector, 1)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, weightFact.FactID, results[0].FactID)
	assert.InDelta(t, 1.0, results[0].Similarity, 0.001)
}

func TestEmbeddingRepo_SearchSimilarFiltered_SQLCipherFallback(t *testing.T) {
	factRepo, embeddingRepo, cleanup := setupSQLCipherRepositoryTestDB(t)
	defer cleanup()
	ctx := context.Background()

	assert.False(t, embeddingRepo.vectorSQLAvailable)

	factA := entity.NewExtractedFact("用户", "患有", "A", 0.9, []string{"msg_a"})
	factA.FactID = "fact_filt_a"
	factA.Status = entity.FactStatusApproved
	require.NoError(t, factRepo.Save(ctx, factA))

	factB := entity.NewExtractedFact("用户", "患有", "B", 0.9, []string{"msg_b"})
	factB.FactID = "fact_filt_b"
	factB.Status = entity.FactStatusApproved
	require.NoError(t, factRepo.Save(ctx, factB))

	vecA := make([]float32, entity.EmbeddingDimension)
	vecA[0] = 1
	require.NoError(t, embeddingRepo.Save(ctx, entity.NewSemanticEmbedding(factA.FactID, vecA, models.CurrentEmbeddingVersion)))

	vecB := make([]float32, entity.EmbeddingDimension)
	vecB[1] = 1
	require.NoError(t, embeddingRepo.Save(ctx, entity.NewSemanticEmbedding(factB.FactID, vecB, "old-version")))

	query := make([]float32, entity.EmbeddingDimension)
	query[0] = 1

	results, err := embeddingRepo.SearchSimilarFiltered(ctx, query, 10, models.CurrentEmbeddingVersion)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, factA.FactID, results[0].FactID)
	assert.InDelta(t, 1.0, results[0].Similarity, 0.001)

	results, err = embeddingRepo.SearchSimilarFiltered(ctx, query, 10, "")
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, factA.FactID, results[0].FactID)
	assert.Equal(t, factB.FactID, results[1].FactID)
	assert.Greater(t, results[0].Similarity, results[1].Similarity)

	results, err = embeddingRepo.SearchSimilarFiltered(ctx, query, 1, "")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, factA.FactID, results[0].FactID)

	results, err = embeddingRepo.SearchSimilarFiltered(ctx, query, 10, "no-such-version")
	require.NoError(t, err)
	assert.Empty(t, results)
}
