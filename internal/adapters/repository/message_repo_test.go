package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageRepo_SaveAndList(t *testing.T) {
	conn := setupTestConnector(t)
	msgRepo := NewMessageRepoSQLite(conn)
	convRepo := NewConversationRepoSQLite(conn)
	ctx := context.Background()
	convID := models.ConversationID("conv_msg_001")

	// 先创建会话（外键约束）
	conv := &entity.Conversation{
		ID:        convID,
		Title:     "消息测试会话",
		Model:     models.ProviderKimi,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
		Messages:  make([]entity.Message, 0),
	}
	require.NoError(t, convRepo.Save(ctx, conv))

	// 保存两条消息
	msg1 := &entity.Message{
		ID:        "msg_001",
		Role:      models.RoleUser,
		Content:   "你好",
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
	}
	msg2 := &entity.Message{
		ID:        "msg_002",
		Role:      models.RoleAssistant,
		Content:   "您好，请问有什么可以帮您？",
		Timestamp: time.Now().UTC().Truncate(time.Millisecond).Add(time.Second),
	}
	require.NoError(t, msgRepo.Save(ctx, convID, msg1))
	require.NoError(t, msgRepo.Save(ctx, convID, msg2))

	// 查询消息
	msgs, nextCursor, err := msgRepo.ListByConversation(ctx, convID, "", 10)
	require.NoError(t, err)
	assert.Len(t, msgs, 2)
	assert.Empty(t, nextCursor)
	// 按时间倒序，msg2 最新
	assert.Equal(t, msg2.ID, msgs[0].ID)
	assert.Equal(t, msg1.ID, msgs[1].ID)
}

func TestMessageRepo_CursorPagination(t *testing.T) {
	conn := setupTestConnector(t)
	msgRepo := NewMessageRepoSQLite(conn)
	convRepo := NewConversationRepoSQLite(conn)
	ctx := context.Background()
	convID := models.ConversationID("conv_cursor")

	conv := &entity.Conversation{
		ID:        convID,
		Title:     "分页测试",
		Model:     models.ProviderKimi,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
		Messages:  make([]entity.Message, 0),
	}
	require.NoError(t, convRepo.Save(ctx, conv))

	// 插入 5 条消息，间隔 1 秒
	now := time.Now().UTC().Truncate(time.Millisecond)
	for i := 0; i < 5; i++ {
		msg := &entity.Message{
			ID:        fmt.Sprintf("msg_p_%d", i),
			Role:      models.RoleUser,
			Content:   fmt.Sprintf("消息 %d", i),
			Timestamp: now.Add(time.Duration(i) * time.Second),
		}
		require.NoError(t, msgRepo.Save(ctx, convID, msg))
	}

	// 第一页 limit=2，取最新的 2 条（msg_p_4, msg_p_3）
	page1, cursor1, err := msgRepo.ListByConversation(ctx, convID, "", 2)
	require.NoError(t, err)
	require.Len(t, page1, 2)
	assert.Equal(t, "msg_p_4", page1[0].ID)
	assert.Equal(t, "msg_p_3", page1[1].ID)
	assert.NotEmpty(t, cursor1)

	// 第二页
	page2, cursor2, err := msgRepo.ListByConversation(ctx, convID, cursor1, 2)
	require.NoError(t, err)
	require.Len(t, page2, 2)
	assert.Equal(t, "msg_p_2", page2[0].ID)
	assert.Equal(t, "msg_p_1", page2[1].ID)
	assert.NotEmpty(t, cursor2)

	// 第三页，只剩一条
	page3, cursor3, err := msgRepo.ListByConversation(ctx, convID, cursor2, 2)
	require.NoError(t, err)
	require.Len(t, page3, 1)
	assert.Equal(t, "msg_p_0", page3[0].ID)
	assert.Empty(t, cursor3)
}

func TestMessageRepo_SoftDelete(t *testing.T) {
	conn := setupTestConnector(t)
	msgRepo := NewMessageRepoSQLite(conn)
	convRepo := NewConversationRepoSQLite(conn)
	ctx := context.Background()
	convID := models.ConversationID("conv_del_msg")

	conv := &entity.Conversation{
		ID:        convID,
		Title:     "删除测试",
		Model:     models.ProviderKimi,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
		Messages:  make([]entity.Message, 0),
	}
	require.NoError(t, convRepo.Save(ctx, conv))

	msg := &entity.Message{
		ID:        "msg_to_delete",
		Role:      models.RoleUser,
		Content:   "这条消息将被删除",
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
	}
	require.NoError(t, msgRepo.Save(ctx, convID, msg))

	// 软删除
	require.NoError(t, msgRepo.SoftDelete(ctx, msg.ID))

	// 查询不应包含已删除消息
	msgs, _, err := msgRepo.ListByConversation(ctx, convID, "", 10)
	require.NoError(t, err)
	assert.Len(t, msgs, 0)

	// 恢复
	require.NoError(t, msgRepo.Restore(ctx, msg.ID))

	msgs, _, err = msgRepo.ListByConversation(ctx, convID, "", 10)
	require.NoError(t, err)
	assert.Len(t, msgs, 1)
	assert.Equal(t, msg.Content, msgs[0].Content)
}

func TestMessageRepo_ListByConversation_Empty(t *testing.T) {
	conn := setupTestConnector(t)
	msgRepo := NewMessageRepoSQLite(conn)
	convRepo := NewConversationRepoSQLite(conn)
	ctx := context.Background()
	convID := models.ConversationID("conv_empty")

	conv := &entity.Conversation{
		ID:        convID,
		Title:     "空会话",
		Model:     models.ProviderKimi,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		UpdatedAt: time.Now().UTC().Truncate(time.Millisecond),
		Messages:  make([]entity.Message, 0),
	}
	require.NoError(t, convRepo.Save(ctx, conv))

	msgs, nextCursor, err := msgRepo.ListByConversation(ctx, convID, "", 10)
	require.NoError(t, err)
	assert.Len(t, msgs, 0)
	assert.Empty(t, nextCursor)
}
