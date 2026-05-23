package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestConnector(t *testing.T) *database.SQLiteConnector {
	t.Helper()
	tmpDir := t.TempDir()
	conn, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, conn.Migrate(ctx))

	t.Cleanup(func() {
		_ = conn.Close()
	})
	return conn
}

func TestConversationRepo_SaveAndGet(t *testing.T) {
	conn := setupTestConnector(t)
	repo := NewConversationRepoSQLite(conn)
	ctx := context.Background()

	conv := &entity.Conversation{
		ID:        "conv_001",
		Title:     "测试会话",
		Model:     models.ProviderKimi,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
		Messages:  make([]entity.Message, 0),
	}

	require.NoError(t, repo.Save(ctx, conv))

	got, err := repo.GetByID(ctx, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, conv.ID, got.ID)
	assert.Equal(t, conv.Title, got.Title)
	assert.Equal(t, conv.Model, got.Model)
	assert.True(t, conv.CreatedAt.Equal(got.CreatedAt), "created_at 不匹配")
	assert.True(t, conv.UpdatedAt.Equal(got.UpdatedAt), "updated_at 不匹配")
}

func TestConversationRepo_GetByID_NotFound(t *testing.T) {
	conn := setupTestConnector(t)
	repo := NewConversationRepoSQLite(conn)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "non_existent")
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound), "期望返回 ErrNotFound")
}

func TestConversationRepo_ListRecent(t *testing.T) {
	conn := setupTestConnector(t)
	repo := NewConversationRepoSQLite(conn)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	// 插入 3 条会话，按时间倒序
	for i := 0; i < 3; i++ {
		conv := &entity.Conversation{
			ID:        models.ConversationID(fmt.Sprintf("conv_%d", i)),
			Title:     fmt.Sprintf("会话 %d", i),
			Model:     models.ProviderOpenAI,
			CreatedAt: now.Add(-time.Duration(i) * time.Hour),
			UpdatedAt: now.Add(-time.Duration(i) * time.Minute),
			Messages:  make([]entity.Message, 0),
		}
		require.NoError(t, repo.Save(ctx, conv))
	}

	// 限制 limit=2，期望返回 updated_at 最新的 2 条
	list, err := repo.ListRecent(ctx, 2)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, models.ConversationID("conv_0"), list[0].ID) // 最新
	assert.Equal(t, models.ConversationID("conv_1"), list[1].ID)
}

func TestConversationRepo_SoftDelete(t *testing.T) {
	conn := setupTestConnector(t)
	repo := NewConversationRepoSQLite(conn)
	ctx := context.Background()

	conv := &entity.Conversation{
		ID:        "conv_del",
		Title:     "待删除",
		Model:     models.ProviderKimi,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
		Messages:  make([]entity.Message, 0),
	}
	require.NoError(t, repo.Save(ctx, conv))

	// 软删除
	require.NoError(t, repo.Delete(ctx, conv.ID))

	// 再次查询应返回 ErrNotFound
	_, err := repo.GetByID(ctx, conv.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))

	// ListRecent 也不应包含已删除
	list, err := repo.ListRecent(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, list, 0)
}

func TestConversationRepo_Save_UpdatesExisting(t *testing.T) {
	conn := setupTestConnector(t)
	repo := NewConversationRepoSQLite(conn)
	ctx := context.Background()

	conv := &entity.Conversation{
		ID:        "conv_update",
		Title:     "原始标题",
		Model:     models.ProviderKimi,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
		Messages:  make([]entity.Message, 0),
	}
	require.NoError(t, repo.Save(ctx, conv))

	// 修改标题后再次保存
	conv.Title = "更新后的标题"
	conv.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)
	require.NoError(t, repo.Save(ctx, conv))

	got, err := repo.GetByID(ctx, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, "更新后的标题", got.Title)
}

// TestConversationRepo_DataDir_Persistence 验证磁盘持久化（非内存）。
func TestConversationRepo_DataDir_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	conn, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	ctx := context.Background()

	require.NoError(t, conn.Migrate(ctx))

	repo := NewConversationRepoSQLite(conn)
	conv := &entity.Conversation{
		ID:        "conv_persist",
		Title:     "持久化测试",
		Model:     models.ProviderKimi,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
		Messages:  make([]entity.Message, 0),
	}
	require.NoError(t, repo.Save(ctx, conv))
	require.NoError(t, conn.Close())

	// 重新打开同一文件，验证数据仍在
	conn2, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	require.NoError(t, conn2.Migrate(ctx))
	repo2 := NewConversationRepoSQLite(conn2)

	got, err := repo2.GetByID(ctx, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, conv.Title, got.Title)
	_ = conn2.Close()

	// 清理数据库文件
	_ = os.RemoveAll(tmpDir)
}

// TestConversationRepo_ArchiveOlderThan 验证归档功能：仅归档 cutoff 之前的会话。
func TestConversationRepo_ArchiveOlderThan(t *testing.T) {
	conn := setupTestConnector(t)
	repo := NewConversationRepoSQLite(conn)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	oldConv := &entity.Conversation{
		ID:        "conv_archive_old",
		Title:     "旧会话",
		Model:     models.ProviderKimi,
		CreatedAt: now.Add(-2 * time.Hour),
		UpdatedAt: now.Add(-2 * time.Hour),
		Messages:  make([]entity.Message, 0),
	}
	newConv := &entity.Conversation{
		ID:        "conv_archive_new",
		Title:     "新会话",
		Model:     models.ProviderKimi,
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  make([]entity.Message, 0),
	}
	require.NoError(t, repo.Save(ctx, oldConv))
	require.NoError(t, repo.Save(ctx, newConv))

	cutoff := now.Add(-1 * time.Hour)
	require.NoError(t, repo.ArchiveOlderThan(ctx, cutoff))

	// 旧会话应被归档（archived_at 非空）
	var archivedAt sql.NullInt64
	err := conn.DB().QueryRowContext(ctx,
		"SELECT archived_at FROM conversations WHERE id = ?", oldConv.ID).Scan(&archivedAt)
	require.NoError(t, err)
	assert.True(t, archivedAt.Valid && archivedAt.Int64 > 0, "旧会话应被归档")

	// 新会话不应被归档
	var newArchivedAt sql.NullInt64
	err = conn.DB().QueryRowContext(ctx,
		"SELECT archived_at FROM conversations WHERE id = ?", newConv.ID).Scan(&newArchivedAt)
	require.NoError(t, err)
	assert.False(t, newArchivedAt.Valid, "新会话不应被归档")
}

// TestConversationRepo_Restore 验证 Restore 可恢复已软删除的会话。
func TestConversationRepo_Restore(t *testing.T) {
	conn := setupTestConnector(t)
	repo := NewConversationRepoSQLite(conn)
	ctx := context.Background()

	conv := &entity.Conversation{
		ID:        "conv_restore",
		Title:     "待恢复",
		Model:     models.ProviderOpenAI,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
		Messages:  make([]entity.Message, 0),
	}
	require.NoError(t, repo.Save(ctx, conv))
	require.NoError(t, repo.Delete(ctx, conv.ID))

	// 恢复后应可正常查询
	require.NoError(t, repo.Restore(ctx, conv.ID))
	got, err := repo.GetByID(ctx, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, conv.ID, got.ID)
	assert.Equal(t, conv.Title, got.Title)
}

// TestConversationRepo_HardDelete 验证 HardDelete 永久删除会话。
func TestConversationRepo_HardDelete(t *testing.T) {
	conn := setupTestConnector(t)
	repo := NewConversationRepoSQLite(conn)
	ctx := context.Background()

	conv := &entity.Conversation{
		ID:        "conv_hard_del",
		Title:     "永久删除",
		Model:     models.ProviderKimi,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
		Messages:  make([]entity.Message, 0),
	}
	require.NoError(t, repo.Save(ctx, conv))
	require.NoError(t, repo.HardDelete(ctx, conv.ID))

	_, err := repo.GetByID(ctx, conv.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

// TestConversationRepo_PermanentlyDeleteOlderThan 验证仅删除早于 cutoff 的已软删除会话。
func TestConversationRepo_PermanentlyDeleteOlderThan(t *testing.T) {
	conn := setupTestConnector(t)
	repo := NewConversationRepoSQLite(conn)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	// 会话 A：已软删除且 deleted_at 为 2 小时前
	oldConv := &entity.Conversation{
		ID:        "conv_perm_old",
		Title:     "旧删除",
		Model:     models.ProviderOpenAI,
		CreatedAt: now.Add(-3 * time.Hour),
		UpdatedAt: now.Add(-3 * time.Hour),
		Messages:  make([]entity.Message, 0),
	}
	require.NoError(t, repo.Save(ctx, oldConv))
	require.NoError(t, repo.Delete(ctx, oldConv.ID))
	_, err := conn.DB().ExecContext(ctx,
		"UPDATE conversations SET deleted_at = ? WHERE id = ?",
		now.Add(-2*time.Hour).UnixMilli(), oldConv.ID)
	require.NoError(t, err)

	// 会话 B：已软删除但 deleted_at 为现在
	newConv := &entity.Conversation{
		ID:        "conv_perm_new",
		Title:     "新删除",
		Model:     models.ProviderOpenAI,
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  make([]entity.Message, 0),
	}
	require.NoError(t, repo.Save(ctx, newConv))
	require.NoError(t, repo.Delete(ctx, newConv.ID))

	// cutoff 为 1 小时前：只应删除 A
	cutoff := now.Add(-1 * time.Hour)
	require.NoError(t, repo.PermanentlyDeleteOlderThan(ctx, cutoff))

	_, err = repo.GetByID(ctx, oldConv.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound), "旧会话应被永久删除")

	_, err = repo.GetByID(ctx, newConv.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound), "新会话已被软删除，GetByID 不应查到")
}

// TestConversationRepo_UpdateTimestamp 验证仅更新会话时间戳。
func TestConversationRepo_UpdateTimestamp(t *testing.T) {
	conn := setupTestConnector(t)
	repo := NewConversationRepoSQLite(conn)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	conv := &entity.Conversation{
		ID:        "conv_update_ts",
		Title:     "时间戳测试",
		Model:     models.ProviderKimi,
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  make([]entity.Message, 0),
	}
	require.NoError(t, repo.Save(ctx, conv))

	newTime := now.Add(1 * time.Hour)
	require.NoError(t, repo.UpdateTimestamp(ctx, conv.ID, newTime))

	got, err := repo.GetByID(ctx, conv.ID)
	require.NoError(t, err)
	assert.True(t, got.UpdatedAt.Equal(newTime), "updated_at 应被更新")
	assert.Equal(t, conv.Title, got.Title, "标题不应被改变")
}

// TestConversationRepo_UpdateTitle 验证仅更新会话标题。
func TestConversationRepo_UpdateTitle(t *testing.T) {
	conn := setupTestConnector(t)
	repo := NewConversationRepoSQLite(conn)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	conv := &entity.Conversation{
		ID:        "conv_update_title",
		Title:     "原始标题",
		Model:     models.ProviderOpenAI,
		CreatedAt: now,
		UpdatedAt: now,
		Messages:  make([]entity.Message, 0),
	}
	require.NoError(t, repo.Save(ctx, conv))

	require.NoError(t, repo.UpdateTitle(ctx, conv.ID, "新标题"))

	got, err := repo.GetByID(ctx, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, "新标题", got.Title)
}

// TestConversationRepo_Save_DBClosed 验证数据库关闭后 Save 返回错误。
func TestConversationRepo_Save_DBClosed(t *testing.T) {
	conn := setupTestConnector(t)
	repo := NewConversationRepoSQLite(conn)
	ctx := context.Background()

	conv := &entity.Conversation{
		ID:        "conv_save_err",
		Title:     "错误测试",
		Model:     models.ProviderKimi,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
		Messages:  make([]entity.Message, 0),
	}
	require.NoError(t, repo.Save(ctx, conv))
	require.NoError(t, conn.Close())

	err := repo.Save(ctx, conv)
	require.Error(t, err, "数据库已关闭，Save 应返回错误")
}

// TestConversationRepo_Delete_DBClosed 验证数据库关闭后 Delete 返回错误。
func TestConversationRepo_Delete_DBClosed(t *testing.T) {
	conn := setupTestConnector(t)
	repo := NewConversationRepoSQLite(conn)
	ctx := context.Background()

	require.NoError(t, conn.Close())

	err := repo.Delete(ctx, "any_id")
	require.Error(t, err, "数据库已关闭，Delete 应返回错误")
}
