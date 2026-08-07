package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// TestProviderRepo_ReadAfterUpgrade_ProviderTypeFallback 验证升级后空 provider_type
// 行能按 api_host 正确回退推断类型。
func TestProviderRepo_ReadAfterUpgrade_ProviderTypeFallback(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	conn, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	require.NoError(t, conn.Migrate(ctx))

	store := newMockSecretStore()
	repo, err := NewProviderRepoSQLite(conn, store)
	require.NoError(t, err)

	p := &models.ProviderConfig{
		ID:          "legacy-type-upgrade",
		Name:        "Legacy Type Provider",
		APIHost:     "https://api.moonshot.cn",
		APIKey:      "sk-legacy",
		ModelID:     "moonshot-v1-8k",
		Temperature: 0.7,
		TimeoutMs:   30000,
		MaxRetries:  3,
		GroupName:   "default",
		Enabled:     true,
	}
	require.NoError(t, repo.Create(ctx, p))

	// 模拟 v13 迁移后的旧行：清空 provider_type，读取时应回退推断。
	_, err = repo.db.ExecContext(ctx,
		"UPDATE providers SET provider_type = '' WHERE id = ?", p.ID)
	require.NoError(t, err)

	got, err := repo.Get(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ProviderKimi, got.Type, "空 provider_type 应按 api_host 回退推断为 kimi")

	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, models.ProviderKimi, list[0].Type)
}

// TestProviderRepo_ReadAfterUpgrade_ModelsFallback 验证升级后空 models 行能
// 从 model_id 合成单模型列表。
func TestProviderRepo_ReadAfterUpgrade_ModelsFallback(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	conn, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	require.NoError(t, conn.Migrate(ctx))

	store := newMockSecretStore()
	repo, err := NewProviderRepoSQLite(conn, store)
	require.NoError(t, err)

	p := &models.ProviderConfig{
		ID:          "legacy-models-upgrade",
		Name:        "Legacy Models Provider",
		APIHost:     "https://api.openai.com",
		APIKey:      "sk-legacy",
		ModelID:     "gpt-4o-mini",
		Temperature: 0.7,
		TimeoutMs:   30000,
		MaxRetries:  3,
		GroupName:   "default",
		Enabled:     true,
	}
	require.NoError(t, repo.Create(ctx, p))

	// 模拟 v13 迁移后的旧行：清空 models 列，读取时应合成单模型。
	_, err = repo.db.ExecContext(ctx,
		"UPDATE providers SET models = '[]' WHERE id = ?", p.ID)
	require.NoError(t, err)

	got, err := repo.Get(ctx, p.ID)
	require.NoError(t, err)
	require.Len(t, got.Models, 1)
	assert.Equal(t, "gpt-4o-mini", got.Models[0].ID)
	assert.Equal(t, "gpt-4o-mini", got.Models[0].Name)
	assert.True(t, got.Models[0].Enabled)
}

// TestProviderRepo_ReadAfterUpgrade_FromPlaintextV13 模拟生产环境从 v1.1.9 (v13)
// 明文库升级：先构造 v13 明文库并写入旧行，再经 SQLCipher 加密并迁移到 v15，
// 验证 schema 版本、旧行数据及新增列的默认值均被正确保留。
func TestProviderRepo_ReadAfterUpgrade_FromPlaintextV13(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbPath := fmt.Sprintf("%s/medmemo.db", tmpDir)
	now := time.Now().UnixMilli()

	// 构造 v13 明文库：含 providers 表核心列，无 provider_type / models。
	plainDB, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = plainDB.ExecContext(ctx, v13SchemaFixture)
	require.NoError(t, err)
	_, err = plainDB.ExecContext(ctx, `
		INSERT INTO providers (id, name, api_host, api_key, model_id, created_at, updated_at, auth_method, auth_params)
		VALUES ('v13-prov', 'V13 Provider', 'https://api.openai.com', X'00', 'gpt-4o', ?, ?, 'api_key', '{}')
	`, now, now)
	require.NoError(t, err)
	require.NoError(t, plainDB.Close())

	// SQLCipher 打开会自动将明文库迁移为加密库，再执行 Migrate 到 v15。
	store := newMockSecretStore()
	conn, err := database.NewSQLCipherConnector(tmpDir, store)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	// 验证 sqlcipher_export 后的加密库保留了原明文库的 user_version。
	var userVersion int
	require.NoError(t, conn.DB().QueryRowContext(ctx, "PRAGMA user_version").Scan(&userVersion))
	require.Equal(t, 13, userVersion)

	require.NoError(t, conn.Migrate(ctx))

	var version int
	require.NoError(t, conn.DB().QueryRowContext(ctx, "PRAGMA user_version").Scan(&version))
	require.Equal(t, 15, version)

	// 直接查询底层 DB 验证 provider_type / models 已按默认值补齐。
	var providerType, modelsJSON string
	require.NoError(t, conn.DB().QueryRowContext(ctx,
		"SELECT provider_type, models FROM providers WHERE id = 'v13-prov'").Scan(&providerType, &modelsJSON))
	assert.Equal(t, "", providerType)
	assert.Equal(t, "[]", modelsJSON)

	// 验证旧行数据完整保留。
	var name string
	require.NoError(t, conn.DB().QueryRowContext(ctx,
		"SELECT name FROM providers WHERE id = 'v13-prov'").Scan(&name))
	assert.Equal(t, "V13 Provider", name)
}

// v13SchemaFixture 是 v1.1.9 (schema v13) 的精简 SQL fixture，仅包含本测试所需的表。
// 完整迁移历史由 database 包测试覆盖，此处不重复全部知识库表。
const v13SchemaFixture = `
PRAGMA user_version = 13;

CREATE TABLE IF NOT EXISTS providers (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	api_host TEXT NOT NULL,
	api_key BLOB NOT NULL,
	model_id TEXT NOT NULL,
	temperature REAL DEFAULT 0.7,
	timeout_ms INTEGER DEFAULT 30000,
	max_retries INTEGER DEFAULT 3,
	group_name TEXT DEFAULT '',
	enabled INTEGER DEFAULT 1,
	sort_order INTEGER DEFAULT 0,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	auth_method TEXT DEFAULT 'api_key',
	auth_params TEXT DEFAULT '{}'
);
`
