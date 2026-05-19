package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/internal/infrastructure/database"
	"github.com/medmemo/medmemo/pkg/models"
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
