// Package database 封装 SQLite / SQLCipher 连接池、迁移与事务管理。
package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// SQLiteConnector SQLite 连接管理器，用于事务型数据存储。
// 当前主要供测试与降级场景使用，生产环境使用 SQLCipherConnector。
type SQLiteConnector struct {
	db   *sql.DB
	path string
}

// NewSQLiteConnector 创建 SQLite 连接并配置连接池。
func NewSQLiteConnector(dataDir string) (*SQLiteConnector, error) {
	dataDir = resolveDataDir(dataDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "medmemo.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// 连接池配置：桌面端并发场景需要更大池子
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)

	// 设置 busy_timeout：锁冲突时自动重试最多 5 秒
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// 启用外键约束
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// WAL 模式提升并发读写性能
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to set WAL journal mode: %w", err)
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
	return migrateSQLiteSchema(ctx, c.db)
}
