// Package database 封装 DuckDB / SQLite 连接池、迁移与事务管理。
package database

import (
	"context"
	"fmt"

	"github.com/google/wire"
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

// SQLiteConnector SQLite / SQLCipher 连接管理器，用于事务型数据存储。
type SQLiteConnector struct {
	path string
}

// NewSQLiteConnector 创建 SQLite 连接。
func NewSQLiteConnector(dataDir string) (*SQLiteConnector, error) {
	return &SQLiteConnector{path: dataDir + "/medmemo.db"}, nil
}

// Close 关闭连接。
func (c *SQLiteConnector) Close() error {
	return nil
}

// DatabaseSet 供 Wire 使用的 ProviderSet。
var DatabaseSet = wire.NewSet(
	NewDuckDBConnector,
	NewSQLiteConnector,
)
