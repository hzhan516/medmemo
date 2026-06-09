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
	assert.False(t, got.IsSensitive, "默认应为非敏感")
}

func TestFactRepo_SaveAndGet_Sensitive(t *testing.T) {
	repo, cleanup := setupFactTestDB(t)
	defer cleanup()
	ctx := context.Background()

	f := entity.NewExtractedFact("用户", "患有", "高血压", 0.85, []string{"msg_001"})
	f.IsSensitive = true
	err := repo.Save(ctx, f)
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, f.FactID)
	require.NoError(t, err)
	assert.True(t, got.IsSensitive, "敏感标记应被持久化")
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

func TestFactRepo_ListAllSubjects(t *testing.T) {
	repo, cleanup := setupFactTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// approved 事实
	f1 := entity.NewExtractedFact("用户", "患有", "高血压", 0.9, []string{"msg_1"})
	f1.FactID = "fact_sub1"
	f1.SetStatus(entity.FactStatusApproved)
	require.NoError(t, repo.Save(ctx, f1))

	// 不同 subject 的 approved 事实
	f2 := entity.NewExtractedFact("医生", "建议", "检查", 0.8, []string{"msg_2"})
	f2.FactID = "fact_sub2"
	f2.SetStatus(entity.FactStatusApproved)
	require.NoError(t, repo.Save(ctx, f2))

	// pending 的事实不应出现在 subjects 中
	f3 := entity.NewExtractedFact("护士", "测量", "血压", 0.7, []string{"msg_3"})
	f3.FactID = "fact_sub3"
	require.NoError(t, repo.Save(ctx, f3))

	subjects, err := repo.ListAllSubjects(ctx)
	require.NoError(t, err)
	require.Len(t, subjects, 2)
	assert.Contains(t, subjects, "用户")
	assert.Contains(t, subjects, "医生")
}

func TestFactRepo_FindBySubject(t *testing.T) {
	repo, cleanup := setupFactTestDB(t)
	defer cleanup()
	ctx := context.Background()

	f1 := entity.NewExtractedFact("用户", "患有", "高血压", 0.9, []string{"msg_1"})
	f1.FactID = "fact_fs1"
	f1.SetStatus(entity.FactStatusApproved)
	require.NoError(t, repo.Save(ctx, f1))

	f2 := entity.NewExtractedFact("用户", "服用", "降压药", 0.85, []string{"msg_2"})
	f2.FactID = "fact_fs2"
	f2.SetStatus(entity.FactStatusApproved)
	require.NoError(t, repo.Save(ctx, f2))

	facts, err := repo.FindBySubject(ctx, "用户")
	require.NoError(t, err)
	require.Len(t, facts, 2)
}

func TestFactRepo_FindBySession(t *testing.T) {
	repo, cleanup := setupFactTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 注意：FindBySession 依赖 raw_dialogues 表，需要先插入对话
	// 这里通过 dialogue_repo 插入（如果可用）或直接用 SQL
	// 简化：直接测试空会话场景
	facts, err := repo.FindBySession(ctx, "nonexistent_session")
	require.NoError(t, err)
	assert.Len(t, facts, 0)
}

func TestFactRepo_FindBySession_WithData(t *testing.T) {
	tmpDir := t.TempDir()
	connector, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	defer connector.Close()

	ctx := context.Background()
	err = connector.Migrate(ctx)
	require.NoError(t, err)

	repo := NewFactRepoSQLite(connector)

	// 插入 raw_dialogues
	db := connector.DB()
	_, err = db.ExecContext(ctx, `
		INSERT INTO raw_dialogues (message_id, session_id, role, content, timestamp, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "msg_s1", "sess_a", "user", "我有高血压", 1, 1)
	require.NoError(t, err)

	// 插入关联的 extracted_fact
	f := entity.NewExtractedFact("用户", "患有", "高血压", 0.9, []string{"msg_s1"})
	f.FactID = "fact_fs_a"
	f.SetStatus(entity.FactStatusApproved)
	require.NoError(t, repo.Save(ctx, f))

	facts, err := repo.FindBySession(ctx, "sess_a")
	require.NoError(t, err)
	require.Len(t, facts, 1)
	assert.Equal(t, "fact_fs_a", facts[0].FactID)
}

func TestFactRepo_Save_DuplicateID(t *testing.T) {
	repo, cleanup := setupFactTestDB(t)
	defer cleanup()
	ctx := context.Background()

	f1 := entity.NewExtractedFact("用户", "患有", "偏头痛", 0.85, []string{"msg_001"})
	f1.FactID = "fact_dup_001"
	err := repo.Save(ctx, f1)
	require.NoError(t, err)

	// 使用相同 FactID 再次插入，应按 fact_id 幂等忽略
	f2 := entity.NewExtractedFact("用户", "服用", "阿司匹林", 0.9, []string{"msg_002"})
	f2.FactID = "fact_dup_001"
	err = repo.Save(ctx, f2)
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, "fact_dup_001")
	require.NoError(t, err)
	assert.Equal(t, "患有", got.Predicate)
	assert.Equal(t, "偏头痛", got.Object)
}

func TestFactRepo_Save_CheckConstraintError(t *testing.T) {
	repo, cleanup := setupFactTestDB(t)
	defer cleanup()
	ctx := context.Background()

	f := entity.NewExtractedFact("用户", "患有", "偏头痛", 1.2, []string{"msg_001"})
	err := repo.Save(ctx, f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save fact")
}

func TestFactRepo_UpdateStatus_NotFound(t *testing.T) {
	repo, cleanup := setupFactTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 对不存在的 fact_id 更新状态，SQLite 不会报错（RowsAffected 为 0）
	err := repo.UpdateStatus(ctx, "nonexistent_fact", entity.FactStatusApproved)
	assert.NoError(t, err, "更新不存在的记录不应 panic 或返回错误")
}

func TestFactRepo_ListByStatus_Empty(t *testing.T) {
	repo, cleanup := setupFactTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 数据库为空时查询指定状态，应返回空切片且无错误
	results, err := repo.ListByStatus(ctx, entity.FactStatusApproved, 0, 10)
	require.NoError(t, err)
	assert.Empty(t, results, "空表查询应返回空切片")
}

func TestFactRepo_Delete_DeletesEmbedding(t *testing.T) {
	tmpDir := t.TempDir()
	connector, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	defer connector.Close()

	ctx := context.Background()
	err = connector.Migrate(ctx)
	require.NoError(t, err)

	factRepo := NewFactRepoSQLite(connector)
	embRepo := NewEmbeddingRepoSQLite(connector)

	// 插入事实与 embedding
	f := entity.NewExtractedFact("用户", "患有", "头痛", 0.8, []string{"msg_001"})
	f.FactID = "fact_del_emb"
	require.NoError(t, factRepo.Save(ctx, f))

	vector := make([]float32, entity.EmbeddingDimension)
	for i := range vector {
		vector[i] = float32(i) * 0.001
	}
	emb := entity.NewSemanticEmbedding("fact_del_emb", vector, "all-MiniLM-L6-v2")
	require.NoError(t, embRepo.Save(ctx, emb))

	// 确认 embedding 存在
	_, err = embRepo.GetByFactID(ctx, "fact_del_emb")
	require.NoError(t, err)

	// 删除事实
	err = factRepo.Delete(ctx, "fact_del_emb")
	require.NoError(t, err)

	// 事实应已删除
	_, err = factRepo.GetByID(ctx, "fact_del_emb")
	assert.ErrorIs(t, err, entity.ErrFactNotFound)

	// embedding 也应同步删除
	_, err = embRepo.GetByFactID(ctx, "fact_del_emb")
	assert.ErrorIs(t, err, entity.ErrEmbeddingNotFound)
}
