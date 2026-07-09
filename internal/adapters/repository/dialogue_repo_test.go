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

func setupDialogueTestDB(t *testing.T) (*DialogueRepoSQLite, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	connector, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	err = connector.Migrate(ctx)
	require.NoError(t, err)

	repo := NewDialogueRepoSQLite(connector)
	cleanup := func() {
		connector.Close()
	}
	return repo, cleanup
}

func TestDialogueRepo_InsertAndGet(t *testing.T) {
	repo, cleanup := setupDialogueTestDB(t)
	defer cleanup()
	ctx := context.Background()

	d := entity.NewRawDialogue("session_001", entity.RoleUser, "用户头疼", "kimi-v1")
	err := repo.Insert(ctx, d)
	require.NoError(t, err)

	results, err := repo.GetBySession(ctx, "session_001", 0, 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, d.MessageID, results[0].MessageID)
	assert.Equal(t, "用户头疼", results[0].Content)
	assert.Equal(t, entity.RoleUser, results[0].Role)
}

func TestDialogueRepo_InsertBatch(t *testing.T) {
	repo, cleanup := setupDialogueTestDB(t)
	defer cleanup()
	ctx := context.Background()

	var dialogues []*entity.RawDialogue
	for i := 0; i < 100; i++ {
		d := entity.NewRawDialogue("session_002", entity.RoleUser, "消息内容", "gpt-4o")
		// 避免 MessageID 冲突，给时间戳加一点偏移
		d.MessageID = "msg_batch_" + string(rune('0'+i/10)) + string(rune('0'+i%10))
		dialogues = append(dialogues, d)
	}

	err := repo.InsertBatch(ctx, dialogues)
	require.NoError(t, err)

	results, err := repo.GetBySession(ctx, "session_002", 0, 200)
	require.NoError(t, err)
	assert.Len(t, results, 100)
}

func TestDialogueRepo_GetBySession_Pagination(t *testing.T) {
	repo, cleanup := setupDialogueTestDB(t)
	defer cleanup()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		d := entity.NewRawDialogue("session_003", entity.RoleUser, "消息", "")
		d.MessageID = "msg_p_" + string(rune('0'+i))
		// 给时间戳加偏移确保排序稳定
		d.Timestamp = time.Now().UTC().Add(time.Duration(i) * time.Second)
		err := repo.Insert(ctx, d)
		require.NoError(t, err)
	}

	// 第一页
	page1, err := repo.GetBySession(ctx, "session_003", 0, 2)
	require.NoError(t, err)
	require.Len(t, page1, 2)

	// 第二页
	page2, err := repo.GetBySession(ctx, "session_003", 2, 2)
	require.NoError(t, err)
	require.Len(t, page2, 2)

	// 第三页
	page3, err := repo.GetBySession(ctx, "session_003", 4, 2)
	require.NoError(t, err)
	require.Len(t, page3, 1)
}

func TestDialogueRepo_GetRecent(t *testing.T) {
	repo, cleanup := setupDialogueTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 插入一条 30 分钟前的消息
	old := entity.NewRawDialogue("session_004", entity.RoleUser, "旧消息", "")
	old.Timestamp = time.Now().UTC().Add(-30 * time.Minute)
	old.MessageID = "msg_old"
	err := repo.Insert(ctx, old)
	require.NoError(t, err)

	// 插入一条新消息
	newMsg := entity.NewRawDialogue("session_004", entity.RoleAssistant, "新消息", "")
	newMsg.MessageID = "msg_new"
	err = repo.Insert(ctx, newMsg)
	require.NoError(t, err)

	// 查询最近 10 分钟
	results, err := repo.GetRecent(ctx, "session_004", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "新消息", results[0].Content)
}

func TestDialogueRepo_GetUnprocessed(t *testing.T) {
	repo, cleanup := setupDialogueTestDB(t)
	defer cleanup()
	ctx := context.Background()

	d1 := entity.NewRawDialogue("session_005", entity.RoleUser, "未处理", "")
	d1.MessageID = "msg_u1"
	err := repo.Insert(ctx, d1)
	require.NoError(t, err)

	d2 := entity.NewRawDialogue("session_005", entity.RoleUser, "已处理", "")
	d2.MessageID = "msg_u2"
	d2.MarkProcessed()
	err = repo.Insert(ctx, d2)
	require.NoError(t, err)

	results, err := repo.GetUnprocessed(ctx, 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "msg_u1", results[0].MessageID)
}

func TestDialogueRepo_MarkProcessed(t *testing.T) {
	repo, cleanup := setupDialogueTestDB(t)
	defer cleanup()
	ctx := context.Background()

	d := entity.NewRawDialogue("session_006", entity.RoleUser, "测试", "")
	d.MessageID = "msg_mark"
	err := repo.Insert(ctx, d)
	require.NoError(t, err)

	err = repo.MarkProcessed(ctx, "msg_mark")
	require.NoError(t, err)

	results, err := repo.GetUnprocessed(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestDialogueRepo_MarkFailed(t *testing.T) {
	repo, cleanup := setupDialogueTestDB(t)
	defer cleanup()
	ctx := context.Background()

	d := entity.NewRawDialogue("session_007", entity.RoleUser, "测试", "")
	d.MessageID = "msg_fail"
	err := repo.Insert(ctx, d)
	require.NoError(t, err)

	err = repo.MarkFailed(ctx, "msg_fail")
	require.NoError(t, err)

	results, err := repo.GetUnprocessed(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestDialogueRepo_MarkProcessing(t *testing.T) {
	repo, cleanup := setupDialogueTestDB(t)
	defer cleanup()
	ctx := context.Background()

	d := entity.NewRawDialogue("session_008", entity.RoleUser, "测试处理中", "")
	d.MessageID = "msg_processing"
	err := repo.Insert(ctx, d)
	require.NoError(t, err)

	err = repo.MarkProcessing(ctx, "msg_processing")
	require.NoError(t, err)

	results, err := repo.GetUnprocessed(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestDialogueRepo_Insert_DuplicateID(t *testing.T) {
	repo, cleanup := setupDialogueTestDB(t)
	defer cleanup()
	ctx := context.Background()

	d1 := entity.NewRawDialogue("session_dup", entity.RoleUser, "第一条消息", "kimi-v1")
	d1.MessageID = "msg_dup_001"
	err := repo.Insert(ctx, d1)
	require.NoError(t, err)

	// 使用相同 MessageID 再次插入，应触发 SQLite UNIQUE 约束冲突
	d2 := entity.NewRawDialogue("session_dup", entity.RoleAssistant, "第二条消息", "kimi-v1")
	d2.MessageID = "msg_dup_001"
	err = repo.Insert(ctx, d2)
	assert.Error(t, err, "重复 ID 插入应返回错误")
}

func TestDialogueRepo_GetUnprocessed_Empty(t *testing.T) {
	repo, cleanup := setupDialogueTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 空数据库中查询未处理消息，应返回空切片且无错误
	results, err := repo.GetUnprocessed(ctx, 10)
	require.NoError(t, err)
	assert.Empty(t, results, "空表查询应返回空切片")
}

func TestDialogueRepo_MarkProcessing_NotFound(t *testing.T) {
	repo, cleanup := setupDialogueTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 对不存在的 message_id 标记为处理中，SQLite 不会报错
	err := repo.MarkProcessing(ctx, "nonexistent_msg")
	assert.NoError(t, err, "标记不存在的记录不应 panic 或返回错误")
}

func TestDialogueRepo_InsertBatch_Empty(t *testing.T) {
	repo, cleanup := setupDialogueTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// 传入空切片，事务应正常提交且无错误
	err := repo.InsertBatch(ctx, []*entity.RawDialogue{})
	assert.NoError(t, err, "空切片批量插入不应报错")
}
