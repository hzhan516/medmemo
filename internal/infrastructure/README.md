# Infrastructure Layer（基础设施层）

## 定位

Infrastructure Layer 是 Clean Architecture 的最外层，封装所有技术框架和第三方库。

**核心原则**：该层不知道业务逻辑的存在。它只提供技术能力（数据库连接、HTTP 客户端、配置加载、密钥管理），由 adapter 层按需调用。

## 目录结构

```
internal/infrastructure/
├── onnx/       # Hugot ONNX Runtime 推理运行时封装
├── database/   # DuckDB / SQLite 连接池、迁移、事务管理
├── config/     # Viper 配置加载与校验
├── secret/     # 系统密钥环封装（macOS Keychain / Windows Credential / Linux Secret Service）
└── network/    # HTTP 客户端：重试、超时、断路器
```

## 导入约束（铁律）

| 允许导入                                   | 禁止导入                                                |
|----------------------------------------|-----------------------------------------------------|
| Go 标准库                                 | `github.com/medmemo/medmemo/internal/domain/*`      |
| 第三方框架库（Wails, DuckDB, Viper, Hugot...） | `github.com/medmemo/medmemo/internal/application/*` |
| —                                      | `github.com/medmemo/medmemo/internal/adapters/*`    |

> ⚠️ 基础设施层如果导入了任何业务包，将破坏 Clean Architecture 的依赖方向。

## 核心职责

1. **框架初始化**：数据库连接池创建、ONNX Runtime 加载、配置文件解析
2. **资源管理**：提供 `Close()` / `Shutdown()` 方法，确保优雅释放
3. **平台抽象**：跨平台差异（如密钥环、动态库路径）在此层屏蔽

## 设计原则

- **具体类型暴露**：基础设施层直接暴露具体类型（如 `*sql.DB`、`*onnx.Runtime`），不包装为接口——接口由 application/port 定义
- **配置驱动**：所有可配置项（超时、连接数、路径）通过 `config/` 加载，禁止硬编码
- **失败快速**：初始化时即验证依赖可用性（如数据库连接、模型文件存在性），避免运行时才发现问题

## 示例

```go
// Package database database/duckdb.go
package database

import (
	"database/sql"

	_ "github.com/marcboeker/go-duckdb"
)

type DuckDBConnector struct {
	db *sql.DB
}

func NewDuckDBConnector(dataSourceName string) (*DuckDBConnector, error) {
	db, err := sql.Open("duckdb", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &DuckDBConnector{db: db}, nil
}

func (c *DuckDBConnector) Close() error {
	return c.db.Close()
}
```
