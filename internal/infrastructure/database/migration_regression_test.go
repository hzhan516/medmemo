package database

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// createLegacyDBAtVersion 在临时目录中创建一个指定历史版本的明文 SQLite 数据库，
// 用于模拟从 v1.1.7 (v11) 或 v1.1.9 (v13) 升级上来的存量库。
func createLegacyDBAtVersion(t *testing.T, targetVersion int) (*sql.DB, string) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := fmt.Sprintf("%s/medmemo.db", tmpDir)

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, applyMigrationsUpTo(ctx, db, targetVersion))

	var version int
	require.NoError(t, db.QueryRow("PRAGMA user_version").Scan(&version))
	require.Equal(t, targetVersion, version)

	return db, dbPath
}

// seedLegacyV11Data 向 v11 数据库写入跨表测试数据。
func seedLegacyV11Data(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UnixMilli()

	_, err := db.ExecContext(ctx, `
		INSERT INTO conversations (id, title, model, created_at, updated_at)
		VALUES ('conv_v11', 'Legacy Conversation', 'gpt-4o', ?, ?)
	`, now, now)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO messages (id, conversation_id, role, content, tokens, created_at)
		VALUES ('msg_v11', 'conv_v11', 'user', 'legacy content', 10, ?)
	`, now)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO providers (id, name, api_host, api_key, model_id, created_at, updated_at)
		VALUES ('prov_v11', 'Legacy Provider', 'https://api.openai.com', X'00', 'gpt-4o', ?, ?)
	`, now, now)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO extracted_facts (fact_id, subject, predicate, object, confidence, source_msg_ids, created_at)
		VALUES ('fact_v11', '用户', '主诉', '头痛', 0.8, '["msg_v11"]', ?)
	`, now)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO semantic_embeddings (embedding_id, fact_id, vector, model_version, created_at)
		VALUES ('emb_v11', 'fact_v11', X'00000000', 'all-MiniLM-L6-v2', ?)
	`, now)
	require.NoError(t, err)
}

// seedLegacyV13Data 向 v13 数据库写入跨表测试数据，包含知识库表。
func seedLegacyV13Data(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UnixMilli()

	seedLegacyV11Data(t, db)

	_, err := db.ExecContext(ctx, `
		INSERT INTO answer_feedback (message_id, answer_type, feedback, created_at)
		VALUES ('msg_v11', 'summary', 'helpful', ?)
	`, now)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO knowledge_documents (document_id, title, source_type, checksum, created_at, updated_at)
		VALUES ('doc_v13', 'Legacy Doc', 'manual', 'sha256:abc', ?, ?)
	`, now, now)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO knowledge_chunks (chunk_id, document_id, chunk_index, content, created_at)
		VALUES ('chunk_v13', 'doc_v13', 0, 'legacy chunk', ?)
	`, now)
	require.NoError(t, err)
}

// TestMigrate_Upgrade_FromV11_ToV15 验证 v1.1.7 (v11) 存量库可升级到 v15。
func TestMigrate_Upgrade_FromV11_ToV15(t *testing.T) {
	db, _ := createLegacyDBAtVersion(t, 11)
	defer func() { _ = db.Close() }()
	seedLegacyV11Data(t, db)

	ctx := context.Background()
	require.NoError(t, migrateSQLiteSchema(ctx, db))

	var version int
	require.NoError(t, db.QueryRow("PRAGMA user_version").Scan(&version))
	assert.Equal(t, 15, version)

	assertTableColumnExists(t, db, "providers", "provider_type")
	assertTableColumnExists(t, db, "providers", "models")
	assertTableColumnExists(t, db, "messages", "confidence_score")
	assertTableColumnExists(t, db, "extracted_facts", "is_sensitive")
}

// TestMigrate_Upgrade_FromV13_ToV15 验证 v1.1.9 (v13) 存量库可升级到 v15。
func TestMigrate_Upgrade_FromV13_ToV15(t *testing.T) {
	db, _ := createLegacyDBAtVersion(t, 13)
	defer func() { _ = db.Close() }()
	seedLegacyV13Data(t, db)

	ctx := context.Background()
	require.NoError(t, migrateSQLiteSchema(ctx, db))

	var version int
	require.NoError(t, db.QueryRow("PRAGMA user_version").Scan(&version))
	assert.Equal(t, 15, version)

	assertTableExists(t, db, "knowledge_documents")
	assertTableExists(t, db, "knowledge_embeddings")
	assertTableColumnExists(t, db, "providers", "models")
}

// TestMigrate_Upgrade_DataIntegrity 验证升级后逐表数据完整。
func TestMigrate_Upgrade_DataIntegrity(t *testing.T) {
	db, _ := createLegacyDBAtVersion(t, 13)
	defer func() { _ = db.Close() }()
	seedLegacyV13Data(t, db)

	ctx := context.Background()
	require.NoError(t, migrateSQLiteSchema(ctx, db))

	var title string
	require.NoError(t, db.QueryRow("SELECT title FROM conversations WHERE id = 'conv_v11'").Scan(&title))
	assert.Equal(t, "Legacy Conversation", title)

	var content string
	require.NoError(t, db.QueryRow("SELECT content FROM messages WHERE id = 'msg_v11'").Scan(&content))
	assert.Equal(t, "legacy content", content)

	var modelID string
	require.NoError(t, db.QueryRow("SELECT model_id FROM providers WHERE id = 'prov_v11'").Scan(&modelID))
	assert.Equal(t, "gpt-4o", modelID)

	var subject string
	require.NoError(t, db.QueryRow("SELECT subject FROM extracted_facts WHERE fact_id = 'fact_v11'").Scan(&subject))
	assert.Equal(t, "用户", subject)

	var docTitle string
	require.NoError(t, db.QueryRow("SELECT title FROM knowledge_documents WHERE document_id = 'doc_v13'").Scan(&docTitle))
	assert.Equal(t, "Legacy Doc", docTitle)

	var chunkContent string
	require.NoError(t, db.QueryRow("SELECT content FROM knowledge_chunks WHERE chunk_id = 'chunk_v13'").Scan(&chunkContent))
	assert.Equal(t, "legacy chunk", chunkContent)
}

// TestMigrate_Upgrade_Idempotent 验证重复执行迁移不会破坏数据。
func TestMigrate_Upgrade_Idempotent(t *testing.T) {
	db, _ := createLegacyDBAtVersion(t, 11)
	defer func() { _ = db.Close() }()
	seedLegacyV11Data(t, db)

	ctx := context.Background()
	require.NoError(t, migrateSQLiteSchema(ctx, db))
	require.NoError(t, migrateSQLiteSchema(ctx, db))
	require.NoError(t, migrateSQLiteSchema(ctx, db))

	var version int
	require.NoError(t, db.QueryRow("PRAGMA user_version").Scan(&version))
	assert.Equal(t, 15, version)

	var count int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM conversations").Scan(&count))
	assert.Equal(t, 1, count)

	require.NoError(t, db.QueryRow("SELECT count(*) FROM semantic_embeddings").Scan(&count))
	assert.Equal(t, 1, count)
}

// TestMigrate_Upgrade_Indexes 验证 v11/v13 的索引在升级后仍然存在且新增索引已创建。
func TestMigrate_Upgrade_Indexes(t *testing.T) {
	db, _ := createLegacyDBAtVersion(t, 11)
	defer func() { _ = db.Close() }()

	ctx := context.Background()
	require.NoError(t, migrateSQLiteSchema(ctx, db))

	assertIndexExists(t, db, "idx_embedding_model_version")
	assertIndexExists(t, db, "idx_messages_conv")
	assertIndexExists(t, db, "idx_fact_confidence")
}

func assertTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRow(
		"SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table,
	).Scan(&count))
	assert.Equal(t, 1, count, "table %s should exist", table)
}

func assertTableColumnExists(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRow(
		"SELECT count(*) FROM pragma_table_info(?) WHERE name = ?", table, column,
	).Scan(&count))
	assert.Equal(t, 1, count, "column %s.%s should exist", table, column)
}

func assertIndexExists(t *testing.T, db *sql.DB, index string) {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRow(
		"SELECT count(*) FROM sqlite_master WHERE type = 'index' AND name = ?", index,
	).Scan(&count))
	assert.Equal(t, 1, count, "index %s should exist", index)
}
