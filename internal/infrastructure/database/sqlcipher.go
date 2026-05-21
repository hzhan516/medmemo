// Package database 封装 DuckDB / SQLite / SQLCipher 连接池、迁移与事务管理。
package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/wire"
	"github.com/medmemo/medmemo/internal/infrastructure/secret"
	sqlite3 "github.com/mutecomm/go-sqlcipher"
)

const dbKeyName = "db_key"

// SQLCipherConnector SQLCipher 加密数据库连接管理器。
// AES-256-GCM page-level 透明加密，密钥通过 secret.Store 管理。
type SQLCipherConnector struct {
	db   *sql.DB
	path string
}

// NewSQLCipherConnector 创建 SQLCipher 加密数据库连接。
// 流程：获取主密钥 → 检测明文迁移 → 打开加密库 → 验证密钥 → 配置连接池。
func NewSQLCipherConnector(dataDir string, store secret.Store) (*SQLCipherConnector, error) {
	if dataDir == "" {
		dataDir = ".medmemo/data"
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "medmemo.db")

	// 获取或生成 32 字节主密钥
	key, err := getOrCreateKey(store)
	if err != nil {
		return nil, fmt.Errorf("failed to get database key: %w", err)
	}

	// 检测是否需要明文迁移
	needsMigrate, err := isPlaintextDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check database encryption status: %w", err)
	}
	if needsMigrate {
		if err := migrateFromPlaintext(dbPath, key); err != nil {
			return nil, fmt.Errorf("failed to migrate plaintext database: %w", err)
		}
	}

	// 打开加密数据库（DSN 中通过 _pragma_key 传递密钥）
	dsn := fmt.Sprintf("%s?_pragma_key=x'%x'", dbPath, key)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlcipher database: %w", err)
	}

	// 验证密钥有效性
	if _, err := db.Exec("SELECT count(*) FROM sqlite_master"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database key verification failed: %w", err)
	}

	// 连接池配置
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// 启用外键约束
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// DELETE journal 模式，避免 WAL 空主文件导致的密钥验证问题
	if _, err := db.Exec("PRAGMA journal_mode = DELETE"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to set journal mode: %w", err)
	}

	return &SQLCipherConnector{db: db, path: dbPath}, nil
}

// DB 返回底层 *sql.DB，供 repository 层使用。
func (c *SQLCipherConnector) DB() *sql.DB {
	return c.db
}

// Close 关闭连接。
func (c *SQLCipherConnector) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// Migrate 执行版本化数据库迁移，复用 SQLiteConnector 的 schema 逻辑。
func (c *SQLCipherConnector) Migrate(ctx context.Context) error {
	// SQLCipher 的 schema 和 SQLite 完全一致，直接复用现有迁移
	return migrateSQLiteSchema(ctx, c.db)
}

// getOrCreateKey 从 secret.Store 获取数据库密钥，不存在则生成并存储。
// 密钥长度非 32 字节视为损坏，直接返回错误。
func getOrCreateKey(store secret.Store) ([]byte, error) {
	key, err := store.Get(dbKeyName)
	if err == nil {
		if len(key) == 32 {
			return key, nil
		}
		return nil, fmt.Errorf("database key in store has invalid length %d, expected 32", len(key))
	}

	// 生成 256 位随机密钥
	key = make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate database key: %w", err)
	}

	if err := store.Set(dbKeyName, key); err != nil {
		return nil, fmt.Errorf("failed to store database key: %w", err)
	}

	return key, nil
}

// isPlaintextDB 检测数据库是否为明文 SQLite。
func isPlaintextDB(dbPath string) (bool, error) {
	// 文件不存在 = 新建数据库，无需迁移
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return false, nil
	}

	// 获取主文件信息
	info, err := os.Stat(dbPath)
	if err != nil {
		return false, fmt.Errorf("failed to stat database file: %w", err)
	}

	// 主文件为空但存在 WAL：活跃的 WAL 模式数据库，不是明文
	if info.Size() == 0 {
		if _, err := os.Stat(dbPath + "-wal"); err == nil {
			return false, nil
		}
		// 空文件且无 WAL，保守视为无需迁移
		return false, nil
	}

	encrypted, err := sqlite3.IsEncrypted(dbPath)
	if err != nil {
		return false, fmt.Errorf("failed to check encryption status: %w", err)
	}

	// 未加密 = 明文，需要迁移
	return !encrypted, nil
}

// migrateFromPlaintext 将明文 SQLite 迁移为 SQLCipher 加密数据库。
// 使用 sqlcipher_export() 保证 schema、数据、索引完整复制。
// 原始文件保留为 .backup。
func migrateFromPlaintext(dbPath string, key []byte) error {
	// 用 SQLCipher 打开明文数据库（不设置密钥即可打开明文 db）
	plainDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open plaintext database: %w", err)
	}

	// 验证是有效的 SQLite 数据库
	var count int
	if err := plainDB.QueryRow("SELECT count(*) FROM sqlite_master").Scan(&count); err != nil {
		_ = plainDB.Close()
		return fmt.Errorf("plaintext database verification failed: %w", err)
	}

	// ATTACH 新的加密数据库
	newPath := dbPath + ".new"
	_ = os.Remove(newPath) // 清理可能残留的临时文件

	// 对路径中的单引号做 SQL 转义，防止注入
	escapedPath := strings.ReplaceAll(newPath, "'", "''")
	attachSQL := fmt.Sprintf("ATTACH DATABASE '%s' AS encrypted KEY \"x'%x'\"", escapedPath, key)
	if _, err := plainDB.Exec(attachSQL); err != nil {
		_ = plainDB.Close()
		_ = os.Remove(newPath)
		return fmt.Errorf("failed to attach encrypted database: %w", err)
	}

	// 执行 SQLCipher 原生导出
	if _, err := plainDB.Exec("SELECT sqlcipher_export('encrypted')"); err != nil {
		_ = plainDB.Close()
		_ = os.Remove(newPath)
		return fmt.Errorf("sqlcipher_export failed: %w", err)
	}

	// DETACH
	if _, err := plainDB.Exec("DETACH DATABASE encrypted"); err != nil {
		_ = plainDB.Close()
		_ = os.Remove(newPath)
		return fmt.Errorf("failed to detach encrypted database: %w", err)
	}

	// 关闭明文连接
	if err := plainDB.Close(); err != nil {
		_ = os.Remove(newPath)
		return fmt.Errorf("failed to close plaintext database: %w", err)
	}

	// 验证加密文件是否生成
	encrypted, err := sqlite3.IsEncrypted(newPath)
	if err != nil || !encrypted {
		_ = os.Remove(newPath)
		return fmt.Errorf("encrypted database verification failed")
	}

	// 原子替换：明文 → backup，加密 → 主路径
	backupPath := dbPath + ".backup"
	if err := os.Rename(dbPath, backupPath); err != nil {
		_ = os.Remove(newPath)
		return fmt.Errorf("failed to backup plaintext database: %w", err)
	}
	if err := os.Rename(newPath, dbPath); err != nil {
		// 尝试恢复
		_ = os.Rename(backupPath, dbPath)
		_ = os.Remove(newPath)
		return fmt.Errorf("failed to replace with encrypted database: %w", err)
	}

	return nil
}

// migrateSQLiteSchema 执行 SQLite/SQLCipher 共用的 schema 迁移。
func migrateSQLiteSchema(ctx context.Context, db *sql.DB) error {
	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("failed to read schema version: %w", err)
	}

	migrations := []struct {
		version int
		sql     string
	}{
		{
			version: 1,
			sql: `
			CREATE TABLE IF NOT EXISTS conversations (
				id TEXT PRIMARY KEY,
				title TEXT NOT NULL DEFAULT '',
				model TEXT NOT NULL DEFAULT '',
				created_at INTEGER NOT NULL,
				updated_at INTEGER NOT NULL,
				archived_at INTEGER,
				deleted_at INTEGER
			);
			CREATE INDEX IF NOT EXISTS idx_conversations_updated ON conversations(updated_at);
			CREATE INDEX IF NOT EXISTS idx_conversations_deleted ON conversations(deleted_at);
			`,
		},
		{
			version: 2,
			sql: `
			CREATE TABLE IF NOT EXISTS messages (
				id TEXT PRIMARY KEY,
				conversation_id TEXT NOT NULL,
				role TEXT NOT NULL,
				content TEXT NOT NULL,
				tokens INTEGER DEFAULT 0,
				created_at INTEGER NOT NULL,
				deleted_at INTEGER,
				FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_messages_conv ON messages(conversation_id, created_at);
			CREATE INDEX IF NOT EXISTS idx_messages_deleted ON messages(deleted_at);
			`,
		},
		{
			version: 3,
			sql: `
			CREATE TABLE IF NOT EXISTS memories (
				id TEXT PRIMARY KEY,
				tier INTEGER NOT NULL,
				content TEXT NOT NULL,
				tags TEXT,
				source_conv TEXT,
				confidence REAL DEFAULT 1.0,
				created_at INTEGER NOT NULL,
				accessed_at INTEGER NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_memories_tier ON memories(tier);
			CREATE INDEX IF NOT EXISTS idx_memories_accessed ON memories(accessed_at);
			`,
		},
		{
			version: 4,
			sql: `
			CREATE TABLE IF NOT EXISTS disclaimer_acceptance (
				version TEXT PRIMARY KEY,
				accepted_at INTEGER NOT NULL,
				text_hash TEXT
			);
			`,
		},
		{
			version: 5,
			sql: `
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
				updated_at INTEGER NOT NULL
			);
			CREATE INDEX IF NOT EXISTS idx_providers_enabled ON providers(enabled);
			CREATE INDEX IF NOT EXISTS idx_providers_group ON providers(group_name, sort_order);
			`,
		},
		{
			version: 6,
			sql: `
			ALTER TABLE providers ADD COLUMN auth_method TEXT DEFAULT 'api_key';
			ALTER TABLE providers ADD COLUMN auth_params TEXT DEFAULT '{}';
			`,
		},
	}

	for _, m := range migrations {
		if version >= m.version {
			continue
		}
		if _, err := db.ExecContext(ctx, m.sql); err != nil {
			return fmt.Errorf("failed to apply migration v%d: %w", m.version, err)
		}
		// PRAGMA 不支持参数化，使用 strconv 避免 fmt.Sprintf 拼接
		pragmaSQL := "PRAGMA user_version = " + strconv.Itoa(m.version)
		if _, err := db.ExecContext(ctx, pragmaSQL); err != nil {
			return fmt.Errorf("failed to update schema version to v%d: %w", m.version, err)
		}
		version = m.version
	}

	return nil
}

// SQLCipherSet 供 Wire 使用的 ProviderSet。
var SQLCipherSet = wire.NewSet(
	NewSQLCipherConnector,
)
