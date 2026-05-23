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
	// TODO(作者): 执行 schema 迁移 [Issue#023]
	return fmt.Errorf("DuckDBConnector.Migrate not implemented")
}

// SQLiteConnector SQLite 连接管理器，用于事务型数据存储。
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

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// 启用外键约束
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close() // 外键启用失败后的清理关闭，关闭错误非关键（上方已返回主错误）
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
	return migrateSQLiteSchema(ctx, c.db)
}

// DatabaseSet 供 Wire 使用的 ProviderSet。
var DatabaseSet = wire.NewSet(
	NewDuckDBConnector,
	NewSQLiteConnector,
	NewSQLCipherConnector,
)
