package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFactMigrationTestDB(t *testing.T) (*FactRepoSQLite, *EmbeddingRepoSQLite, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	connector, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	err = connector.Migrate(ctx)
	require.NoError(t, err)

	factRepo := NewFactRepoSQLite(connector)
	embeddingRepo := NewEmbeddingRepoSQLite(connector)
	cleanup := func() {
		_ = connector.Close()
	}
	return factRepo, embeddingRepo, cleanup
}

func TestFactRepo_CountApprovedFactsNeedingEmbedding(t *testing.T) {
	factRepo, embedRepo, cleanup := setupFactMigrationTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// pending 不计入
	fPending := entity.NewExtractedFact("用户", "患有", "X", 0.8, []string{"m0"})
	fPending.FactID = "fact_m0"
	require.NoError(t, factRepo.Save(ctx, fPending))

	// approved + 无 embedding = 需要
	f1 := entity.NewExtractedFact("用户", "患有", "A", 0.8, []string{"m1"})
	f1.FactID = "fact_m1"
	f1.Status = entity.FactStatusApproved
	require.NoError(t, factRepo.Save(ctx, f1))

	// approved + 当前版本 = 不需要
	f2 := entity.NewExtractedFact("用户", "患有", "B", 0.8, []string{"m2"})
	f2.FactID = "fact_m2"
	f2.Status = entity.FactStatusApproved
	require.NoError(t, factRepo.Save(ctx, f2))
	v := make([]float32, entity.EmbeddingDimension)
	require.NoError(t, embedRepo.Save(ctx, entity.NewSemanticEmbedding(f2.FactID, v, models.CurrentEmbeddingVersion)))

	// approved + 旧版本 = 需要
	f3 := entity.NewExtractedFact("用户", "患有", "C", 0.8, []string{"m3"})
	f3.FactID = "fact_m3"
	f3.Status = entity.FactStatusApproved
	require.NoError(t, factRepo.Save(ctx, f3))
	require.NoError(t, embedRepo.Save(ctx, entity.NewSemanticEmbedding(f3.FactID, v, "old-version")))

	count, err := factRepo.CountApprovedFactsNeedingEmbedding(ctx, models.CurrentEmbeddingVersion)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestFactRepo_ListApprovedFactsNeedingEmbedding_Cursor(t *testing.T) {
	factRepo, embedRepo, cleanup := setupFactMigrationTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 创建 5 条 approved fact，按 created_at 自然递增
	for i := 0; i < 5; i++ {
		f := entity.NewExtractedFact("用户", "患有", fmt.Sprintf("F%d", i), 0.8, []string{fmt.Sprintf("m%d", i)})
		f.FactID = fmt.Sprintf("fact_c%d", i)
		f.Status = entity.FactStatusApproved
		require.NoError(t, factRepo.Save(ctx, f))
	}

	// 给 fact_c0 和 fact_c2 写入当前版本 embedding（不需要迁移）
	v := make([]float32, entity.EmbeddingDimension)
	require.NoError(t, embedRepo.Save(ctx, entity.NewSemanticEmbedding("fact_c0", v, models.CurrentEmbeddingVersion)))
	require.NoError(t, embedRepo.Save(ctx, entity.NewSemanticEmbedding("fact_c2", v, models.CurrentEmbeddingVersion)))

	// 给 fact_c1 写入旧版本 embedding，fact_c3/fact_c4 无 embedding
	require.NoError(t, embedRepo.Save(ctx, entity.NewSemanticEmbedding("fact_c1", v, "old-version")))

	// 第一页：limit=2，应返回 fact_c1（旧版本）和 fact_c3（无 embedding）
	page1, err := factRepo.ListApprovedFactsNeedingEmbedding(ctx, models.CurrentEmbeddingVersion, time.Time{}, "", 2)
	require.NoError(t, err)
	require.Len(t, page1, 2)
	assert.Equal(t, "fact_c1", page1[0].FactID)
	assert.Equal(t, "fact_c3", page1[1].FactID)

	// 第二页：用 page1 最后一条作为 cursor
	last := page1[len(page1)-1]
	page2, err := factRepo.ListApprovedFactsNeedingEmbedding(ctx, models.CurrentEmbeddingVersion, last.CreatedAt, last.FactID, 2)
	require.NoError(t, err)
	require.Len(t, page2, 1)
	assert.Equal(t, "fact_c4", page2[0].FactID)
}

func TestFactRepo_ListApprovedFactsNeedingEmbedding_UpdateDuringScan(t *testing.T) {
	factRepo, embedRepo, cleanup := setupFactMigrationTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 创建 3 条需要迁移的 approved fact
	for i := 0; i < 3; i++ {
		f := entity.NewExtractedFact("用户", "患有", fmt.Sprintf("U%d", i), 0.8, []string{fmt.Sprintf("mu%d", i)})
		f.FactID = fmt.Sprintf("fact_u%d", i)
		f.Status = entity.FactStatusApproved
		require.NoError(t, factRepo.Save(ctx, f))
	}

	// 第一页取 1 条
	page1, err := factRepo.ListApprovedFactsNeedingEmbedding(ctx, models.CurrentEmbeddingVersion, time.Time{}, "", 1)
	require.NoError(t, err)
	require.Len(t, page1, 1)

	// 模拟处理完 fact_u0 后更新其 embedding（候选集缩小，但 cursor 分页不应跳过后续项）
	v := make([]float32, entity.EmbeddingDimension)
	require.NoError(t, embedRepo.Save(ctx, entity.NewSemanticEmbedding(page1[0].FactID, v, models.CurrentEmbeddingVersion)))

	// 第二页用 cursor 继续，应返回 fact_u1
	last := page1[len(page1)-1]
	page2, err := factRepo.ListApprovedFactsNeedingEmbedding(ctx, models.CurrentEmbeddingVersion, last.CreatedAt, last.FactID, 10)
	require.NoError(t, err)
	require.Len(t, page2, 2)
	assert.Equal(t, "fact_u1", page2[0].FactID)
	assert.Equal(t, "fact_u2", page2[1].FactID)
}
