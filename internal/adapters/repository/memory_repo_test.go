package repository

import (
	"context"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMemoryRepo(t *testing.T) (*MemoryRepoSQLite, func()) {
	t.Helper()
	dir := t.TempDir()
	connector, err := database.NewSQLiteConnector(dir)
	require.NoError(t, err)

	// 确保 memories 表存在
	ctx := context.Background()
	err = connector.Migrate(ctx)
	require.NoError(t, err)

	repo := NewMemoryRepoSQLite(connector)
	return repo, func() { _ = connector.Close() }
}

func TestMemoryRepoSQLite_SaveAndGet(t *testing.T) {
	repo, cleanup := setupMemoryRepo(t)
	defer cleanup()

	ctx := context.Background()
	mem := entity.NewHealthMemory(entity.TierShortTerm, "头痛三天，伴有轻微发热", "conv_123")
	mem.Tags = []string{"症状", "头痛"}

	err := repo.Save(ctx, mem)
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, mem.ID)
	require.NoError(t, err)
	assert.Equal(t, mem.ID, got.ID)
	assert.Equal(t, mem.Content, got.Content)
	assert.Equal(t, mem.Tier, got.Tier)
	assert.Equal(t, mem.Tags, got.Tags)
}

func TestMemoryRepoSQLite_GetByID_NotFound(t *testing.T) {
	repo, cleanup := setupMemoryRepo(t)
	defer cleanup()

	_, err := repo.GetByID(context.Background(), "non-existent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryRepoSQLite_Search(t *testing.T) {
	repo, cleanup := setupMemoryRepo(t)
	defer cleanup()

	ctx := context.Background()
	mem1 := entity.NewHealthMemory(entity.TierShortTerm, "头痛三天，伴有轻微发热", "conv_1")
	mem2 := entity.NewHealthMemory(entity.TierLongTerm, "血压偏高，建议定期监测", "conv_2")

	require.NoError(t, repo.Save(ctx, mem1))
	require.NoError(t, repo.Save(ctx, mem2))

	results, err := repo.Search(ctx, "头痛", 10)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, mem1.ID, results[0].ID)
}

func TestMemoryRepoSQLite_ListByTier(t *testing.T) {
	repo, cleanup := setupMemoryRepo(t)
	defer cleanup()

	ctx := context.Background()
	mem1 := entity.NewHealthMemory(entity.TierShortTerm, "内容1", "conv_1")
	mem2 := entity.NewHealthMemory(entity.TierShortTerm, "内容2", "conv_2")
	mem3 := entity.NewHealthMemory(entity.TierLongTerm, "内容3", "conv_3")

	require.NoError(t, repo.Save(ctx, mem1))
	require.NoError(t, repo.Save(ctx, mem2))
	require.NoError(t, repo.Save(ctx, mem3))

	results, err := repo.ListByTier(ctx, entity.TierShortTerm, 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestMemoryRepoSQLite_Delete(t *testing.T) {
	repo, cleanup := setupMemoryRepo(t)
	defer cleanup()

	ctx := context.Background()
	mem := entity.NewHealthMemory(entity.TierShortTerm, "待删除", "conv_1")
	require.NoError(t, repo.Save(ctx, mem))

	err := repo.Delete(ctx, mem.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, mem.ID)
	require.Error(t, err)
}

func TestMemoryRepoSQLite_SemanticSearch_NotSupported(t *testing.T) {
	repo, cleanup := setupMemoryRepo(t)
	defer cleanup()

	_, err := repo.SemanticSearch(context.Background(), []float32{0.1, 0.2}, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sqlite-vec")
}

func TestMemoryRepoSQLite_Update(t *testing.T) {
	repo, cleanup := setupMemoryRepo(t)
	defer cleanup()

	ctx := context.Background()
	mem := entity.NewHealthMemory(entity.TierShortTerm, "原始内容", "conv_1")
	require.NoError(t, repo.Save(ctx, mem))

	// 修改内容后再次保存（INSERT OR REPLACE）
	mem.Content = "更新后的内容"
	mem.AccessedAt = time.Now()
	require.NoError(t, repo.Save(ctx, mem))

	got, err := repo.GetByID(ctx, mem.ID)
	require.NoError(t, err)
	assert.Equal(t, "更新后的内容", got.Content)
}

func TestEnsureMemorySchema(t *testing.T) {
	dir := t.TempDir()
	connector, err := database.NewSQLiteConnector(dir)
	require.NoError(t, err)
	defer connector.Close()

	// 直接调用包级函数
	err = EnsureMemorySchema(connector.DB())
	require.NoError(t, err)
}
