package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
)

func setupFactTestDB(t *testing.T) (*FactRepoSQLite, func()) {
	tmpDir := t.TempDir()
	connector, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	err = connector.Migrate(ctx)
	require.NoError(t, err)

	repo := NewFactRepoSQLite(connector)
	cleanup := func() {
		connector.Close()
	}
	return repo, cleanup
}

func TestFactRepo_SaveAndGet(t *testing.T) {
	repo, cleanup := setupFactTestDB(t)
	defer cleanup()
	ctx := context.Background()

	f := entity.NewExtractedFact("用户", "患有", "偏头痛", 0.85, []string{"msg_001"})
	err := repo.Save(ctx, f)
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, f.FactID)
	require.NoError(t, err)
	assert.Equal(t, f.Subject, got.Subject)
	assert.Equal(t, f.Predicate, got.Predicate)
	assert.Equal(t, f.Object, got.Object)
	assert.Equal(t, f.Confidence, got.Confidence)
	assert.Equal(t, []string{"msg_001"}, got.SourceMsgIDs)
	assert.Equal(t, entity.FactStatusPending, got.Status)
}

func TestFactRepo_GetByID_NotFound(t *testing.T) {
	repo, cleanup := setupFactTestDB(t)
	defer cleanup()
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	assert.ErrorIs(t, err, entity.ErrFactNotFound)
}

func TestFactRepo_ListByStatus(t *testing.T) {
	repo, cleanup := setupFactTestDB(t)
	defer cleanup()
	ctx := context.Background()

	f1 := entity.NewExtractedFact("用户", "患有", "A", 0.9, []string{"msg_001"})
	f1.FactID = "fact_a"
	require.NoError(t, repo.Save(ctx, f1))

	f2 := entity.NewExtractedFact("用户", "服用", "B", 0.3, []string{"msg_002"})
	f2.FactID = "fact_b"
	f2.SetStatus(entity.FactStatusRejected)
	require.NoError(t, repo.Save(ctx, f2))

	pending, err := repo.ListPending(ctx, 0, 10)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, "fact_a", pending[0].FactID)

	rejected, err := repo.ListByStatus(ctx, entity.FactStatusRejected, 0, 10)
	require.NoError(t, err)
	require.Len(t, rejected, 1)
	assert.Equal(t, "fact_b", rejected[0].FactID)
}

func TestFactRepo_UpdateStatus(t *testing.T) {
	repo, cleanup := setupFactTestDB(t)
	defer cleanup()
	ctx := context.Background()

	f := entity.NewExtractedFact("用户", "患有", "C", 0.8, []string{"msg_003"})
	f.FactID = "fact_c"
	require.NoError(t, repo.Save(ctx, f))

	err := repo.UpdateStatus(ctx, "fact_c", entity.FactStatusApproved)
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, "fact_c")
	require.NoError(t, err)
	assert.Equal(t, entity.FactStatusApproved, got.Status)
	assert.NotNil(t, got.ReviewedAt)
}

func TestFactRepo_Delete(t *testing.T) {
	repo, cleanup := setupFactTestDB(t)
	defer cleanup()
	ctx := context.Background()

	f := entity.NewExtractedFact("用户", "患有", "D", 0.7, []string{"msg_004"})
	f.FactID = "fact_d"
	require.NoError(t, repo.Save(ctx, f))

	err := repo.Delete(ctx, "fact_d")
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, "fact_d")
	assert.ErrorIs(t, err, entity.ErrFactNotFound)
}

func TestFactRepo_GetStats(t *testing.T) {
	repo, cleanup := setupFactTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 2 pending
	for i := 0; i < 2; i++ {
		f := entity.NewExtractedFact("用户", "患有", string(rune('A'+i)), 0.8, []string{"msg_x"})
		f.FactID = "fact_s" + string(rune('0'+i))
		require.NoError(t, repo.Save(ctx, f))
	}

	// 1 approved
	f3 := entity.NewExtractedFact("用户", "服用", "C", 0.9, []string{"msg_y"})
	f3.FactID = "fact_s2"
	f3.SetStatus(entity.FactStatusApproved)
	require.NoError(t, repo.Save(ctx, f3))

	// 1 rejected
	f4 := entity.NewExtractedFact("用户", "检查", "D", 0.3, []string{"msg_z"})
	f4.FactID = "fact_s3"
	f4.SetStatus(entity.FactStatusRejected)
	require.NoError(t, repo.Save(ctx, f4))

	total, approved, rejected, pending, err := repo.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(4), total)
	assert.Equal(t, int64(1), approved)
	assert.Equal(t, int64(1), rejected)
	assert.Equal(t, int64(2), pending)
}
