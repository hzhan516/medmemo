// Package database 封装 DuckDB / SQLite 连接池、迁移与事务管理。
package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/wire"
	_ "modernc.org/sqlite"
)

// DuckDBConnector DuckDB 嵌入式数据库连接管理器。
type DuckDBConnector struct {
	path string
	// TODO(作者): 接入 DuckDB Go 驱动 [Issue#020]
}

// NewDuckDBConnector 创建 DuckDB 连接。
func NewDuckDBConnector(dataDir string) (*DuckDBConnector, error) {
	return &DuckDBConnector{path: dataDir + "/medmemo.duckdb"}, nil
}

// Close 关闭数据库连接。
func (c *DuckDBConnector) Close() error {
	return nil
}

// Migrate 执行数据库迁移脚本。
func (c *DuckDBConnector) Migrate(ctx context.Context) error {
	// TODO(作者): 执行 schema 迁移 [Issue#021]
	return fmt.Errorf("DuckDBConnector.Migrate not implemented")
}

// SQLiteConnector SQLite 连接管理器，用于事务型数据存储。
type SQLiteConnector struct {
	db   *sql.DB
	path string
}

// NewSQLiteConnector 创建 SQLite 连接并配置连接池。
func NewSQLiteConnector(dataDir string) (*SQLiteConnector, error) {
	if dataDir == "" {
		dataDir = ".medmemo/data"
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "medmemo.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// 启用外键约束
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return &SQLiteConnector{db: db, path: dbPath}, nil
}

// DB 返回底层的 *sql.DB，供 repository 层使用。
func (c *SQLiteConnector) DB() *sql.DB {
	return c.db
}

// Close 关闭连接。
func (c *SQLiteConnector) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// Migrate 执行版本化数据库迁移。
func (c *SQLiteConnector) Migrate(ctx context.Context) error {
	var version int
	if err := c.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
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
	}

	for _, m := range migrations {
		if version >= m.version {
			continue
		}
		if _, err := c.db.ExecContext(ctx, m.sql); err != nil {
			return fmt.Errorf("failed to apply migration v%d: %w", m.version, err)
		}
		if _, err := c.db.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", m.version)); err != nil {
			return fmt.Errorf("failed to update schema version to v%d: %w", m.version, err)
		}
		version = m.version
	}

	return nil
}

// DatabaseSet 供 Wire 使用的 ProviderSet。
var DatabaseSet = wire.NewSet(
	NewDuckDBConnector,
	NewSQLiteConnector,
)
