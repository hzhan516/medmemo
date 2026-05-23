// Package database 封装 DuckDB / SQLite / SQLCipher 连接池、迁移与事务管理。
package database

import (
	"context"
	"database/sql"
)

// DBConnector 定义所有数据库连接器的统一接口。
// Repository 层通过此接口消费连接器，屏蔽底层具体实现（SQLite / SQLCipher / DuckDB）。
type DBConnector interface {
	// DB 返回底层的 *sql.DB，供 repository 层使用。
	DB() *sql.DB
	// Close 关闭数据库连接。
	Close() error
	// Migrate 执行版本化数据库迁移。
	Migrate(ctx context.Context) error
}
