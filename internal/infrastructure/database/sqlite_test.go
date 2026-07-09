package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSQLiteConnector(t *testing.T) {
	tmpDir := t.TempDir()
	connector, err := NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, connector)
	defer connector.Close()

	// 验证底层 DB 可用
	db := connector.DB()
	require.NotNil(t, db)

	var version int
	err = db.QueryRow("PRAGMA user_version").Scan(&version)
	require.NoError(t, err)
	// NewSQLiteConnector 仅创建连接，不自动执行迁移，初始版本为 0
	assert.Equal(t, 0, version)
}

func TestSQLiteConnector_Migrate_CreatesTables(t *testing.T) {
	tmpDir := t.TempDir()
	connector, err := NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	defer connector.Close()

	ctx := context.Background()
	err = connector.Migrate(ctx)
	require.NoError(t, err)

	db := connector.DB()

	// 验证 raw_dialogues 表存在
	var count int
	err = db.QueryRow(`
		SELECT count(*) FROM sqlite_master 
		WHERE type='table' AND name='raw_dialogues'
	`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// 验证 extracted_facts 表存在
	err = db.QueryRow(`
		SELECT count(*) FROM sqlite_master 
		WHERE type='table' AND name='extracted_facts'
	`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// 验证 semantic_embeddings 表存在
	err = db.QueryRow(`
		SELECT count(*) FROM sqlite_master 
		WHERE type='table' AND name='semantic_embeddings'
	`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// 验证索引存在
	indexes := []string{
		"idx_raw_session_time",
		"idx_raw_extraction_status",
		"idx_fact_confidence",
		"idx_fact_status",
		"idx_embedding_fact",
	}
	for _, idx := range indexes {
		err = db.QueryRow(`
			SELECT count(*) FROM sqlite_master 
			WHERE type='index' AND name=?
		`, idx).Scan(&count)
		require.NoError(t, err, "index %s", idx)
		assert.Equal(t, 1, count, "index %s should exist", idx)
	}
}

func TestSQLiteConnector_Migrate_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	connector, err := NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	defer connector.Close()

	ctx := context.Background()

	// 第一次迁移
	err = connector.Migrate(ctx)
	require.NoError(t, err)

	// 第二次迁移（幂等）
	err = connector.Migrate(ctx)
	require.NoError(t, err)
}

func TestSQLiteConnector_DB_ReturnsConnection(t *testing.T) {
	tmpDir := t.TempDir()
	connector, err := NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	defer connector.Close()

	db := connector.DB()
	require.NotNil(t, db)

	// 验证连接可用
	err = db.Ping()
	assert.NoError(t, err)
}

func TestSQLiteConnector_Close(t *testing.T) {
	tmpDir := t.TempDir()
	connector, err := NewSQLiteConnector(tmpDir)
	require.NoError(t, err)

	err = connector.Close()
	assert.NoError(t, err)

	// 关闭后再操作应报错
	db := connector.DB()
	err = db.Ping()
	assert.Error(t, err)
}

func TestSQLiteConnector_ForeignKeysEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	connector, err := NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	defer connector.Close()

	ctx := context.Background()
	err = connector.Migrate(ctx)
	require.NoError(t, err)

	db := connector.DB()

	var fkEnabled int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	require.NoError(t, err)
	assert.Equal(t, 1, fkEnabled, "foreign keys should be enabled")
}

func TestSQLiteConnector_DataDirCreated(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "nested", "data")

	connector, err := NewSQLiteConnector(subDir)
	require.NoError(t, err)
	defer connector.Close()

	_, err = os.Stat(subDir)
	assert.NoError(t, err, "data directory should be created")
}

func TestMigrateV7_ForeignKeyConstraint(t *testing.T) {
	tmpDir := t.TempDir()
	connector, err := NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	defer connector.Close()

	ctx := context.Background()
	err = connector.Migrate(ctx)
	require.NoError(t, err)

	db := connector.DB()

	// 插入一条事实
	_, err = db.Exec(`
		INSERT INTO extracted_facts (fact_id, subject, predicate, object, confidence, source_msg_ids, created_at)
		VALUES ('fact_001', '用户', '患有', '头痛', 0.8, '["msg_001"]', ?)
	`, timeNowMs())
	require.NoError(t, err)

	// 插入关联的嵌入
	vectorBytes := make([]byte, 384*4)
	_, err = db.Exec(`
		INSERT INTO semantic_embeddings (embedding_id, fact_id, vector, model_version, created_at)
		VALUES ('emb_001', 'fact_001', ?, 'all-MiniLM-L6-v2', ?)
	`, vectorBytes, timeNowMs())
	require.NoError(t, err)

	// 尝试删除事实（应级联删除嵌入）
	_, err = db.Exec("DELETE FROM extracted_facts WHERE fact_id = ?", "fact_001")
	require.NoError(t, err)

	// 验证嵌入已被级联删除
	var count int
	err = db.QueryRow("SELECT count(*) FROM semantic_embeddings WHERE fact_id = ?", "fact_001").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestMigrateV7_CheckConstraints(t *testing.T) {
	tmpDir := t.TempDir()
	connector, err := NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	defer connector.Close()

	ctx := context.Background()
	err = connector.Migrate(ctx)
	require.NoError(t, err)

	db := connector.DB()

	// 测试无效 role
	_, err = db.Exec(`
		INSERT INTO raw_dialogues (message_id, session_id, role, content, timestamp, created_at)
		VALUES ('msg_bad', 's1', 'invalid_role', 'test', ?, ?)
	`, timeNowMs(), timeNowMs())
	assert.Error(t, err, "should reject invalid role")

	// 测试无效 extraction_status
	_, err = db.Exec(`
		INSERT INTO raw_dialogues (message_id, session_id, role, content, timestamp, extraction_status, created_at)
		VALUES ('msg_bad2', 's1', 'user', 'test', ?, 'bad_status', ?)
	`, timeNowMs(), timeNowMs())
	assert.Error(t, err, "should reject invalid extraction_status")

	// 测试无效 confidence 范围
	_, err = db.Exec(`
		INSERT INTO extracted_facts (fact_id, subject, predicate, object, confidence, source_msg_ids, created_at)
		VALUES ('fact_bad', '用户', '患有', '头痛', 1.5, '["msg_001"]', ?)
	`, timeNowMs())
	assert.Error(t, err, "should reject confidence > 1")

	// 测试无效 fact status
	_, err = db.Exec(`
		INSERT INTO extracted_facts (fact_id, subject, predicate, object, confidence, source_msg_ids, status, created_at)
		VALUES ('fact_bad2', '用户', '患有', '头痛', 0.5, '["msg_001"]', 'unknown', ?)
	`, timeNowMs())
	assert.Error(t, err, "should reject invalid fact status")
}

func timeNowMs() int64 {
	return time.Now().UnixMilli()
}
