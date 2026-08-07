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

// createPlaintextLegacyDB 创建指定历史版本的明文 SQLite 数据库，模拟生产环境升级场景。
func createPlaintextLegacyDB(t *testing.T, targetVersion int) (string, *mockSecretStore) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := fmt.Sprintf("%s/medmemo.db", tmpDir)

	plainDB, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, applyMigrationsUpTo(ctx, plainDB, targetVersion))

	now := time.Now().UnixMilli()

	// 写入 providers 旧行：无 provider_type / models，验证迁移后 fallback。
	_, err = plainDB.ExecContext(ctx, `
		INSERT INTO providers (id, name, api_host, api_key, model_id, created_at, updated_at, auth_method, auth_params)
		VALUES ('legacy_prov', 'Legacy', 'https://api.openai.com', X'00', 'gpt-4o', ?, ?, 'api_key', '{}')
	`, now, now)
	require.NoError(t, err)

	_, err = plainDB.ExecContext(ctx, `
		INSERT INTO conversations (id, title, model, created_at, updated_at)
		VALUES ('legacy_conv', 'Legacy', 'gpt-4o', ?, ?)
	`, now, now)
	require.NoError(t, err)

	require.NoError(t, plainDB.Close())

	store := newMockSecretStore()
	return tmpDir, store
}

// TestSQLCipherConnector_MigrateFromPlaintext_LegacyV11 验证生产明文 v11 库迁移为加密 v15。
func TestSQLCipherConnector_MigrateFromPlaintext_LegacyV11(t *testing.T) {
	tmpDir, store := createPlaintextLegacyDB(t, 11)

	conn, err := NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)
	require.NotNil(t, conn)

	ctx := context.Background()
	require.NoError(t, conn.Migrate(ctx))

	var version int
	require.NoError(t, conn.DB().QueryRowContext(ctx, "PRAGMA user_version").Scan(&version))
	assert.Equal(t, 15, version)

	var title string
	require.NoError(t, conn.DB().QueryRowContext(ctx,
		"SELECT title FROM conversations WHERE id = 'legacy_conv'").Scan(&title))
	assert.Equal(t, "Legacy", title)

	var providerType string
	require.NoError(t, conn.DB().QueryRowContext(ctx,
		"SELECT provider_type FROM providers WHERE id = 'legacy_prov'").Scan(&providerType))
	assert.Equal(t, "", providerType)

	require.NoError(t, conn.Close())
}

// TestSQLCipherConnector_MigrateFromPlaintext_LegacyV13 验证生产明文 v13 库迁移为加密 v15。
func TestSQLCipherConnector_MigrateFromPlaintext_LegacyV13(t *testing.T) {
	tmpDir, store := createPlaintextLegacyDB(t, 13)

	conn, err := NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)
	require.NotNil(t, conn)

	ctx := context.Background()
	require.NoError(t, conn.Migrate(ctx))

	var version int
	require.NoError(t, conn.DB().QueryRowContext(ctx, "PRAGMA user_version").Scan(&version))
	assert.Equal(t, 15, version)

	var modelsCol string
	require.NoError(t, conn.DB().QueryRowContext(ctx,
		"SELECT models FROM providers WHERE id = 'legacy_prov'").Scan(&modelsCol))
	assert.Equal(t, "[]", modelsCol)

	require.NoError(t, conn.Close())
}

// TestSQLCipherConnector_ReopenAfterLegacyMigration 验证加密库迁移后可重复打开。
func TestSQLCipherConnector_ReopenAfterLegacyMigration(t *testing.T) {
	tmpDir, store := createPlaintextLegacyDB(t, 13)

	conn1, err := NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)
	require.NoError(t, conn1.Migrate(context.Background()))
	require.NoError(t, conn1.Close())

	conn2, err := NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)
	require.NotNil(t, conn2)

	var count int
	require.NoError(t, conn2.DB().QueryRowContext(context.Background(),
		"SELECT count(*) FROM providers WHERE id = 'legacy_prov'").Scan(&count))
	assert.Equal(t, 1, count)

	require.NoError(t, conn2.Close())
}
