# Infrastructure Layer

> 🌐 [中文版本](../../docs/i18n/zh-Hans-CN/internal/infrastructure/README.md)

## Why This Layer Exists

The Infrastructure Layer is the outermost ring of Clean Architecture. It encapsulates all technical frameworks and third-party libraries.

**Core principle**: this layer knows nothing about business logic. It only provides technical capabilities — database connections, HTTP clients, configuration loading, and secret management — which the adapter layer consumes as needed.

By isolating framework code here, we ensure that:
- Upgrading DuckDB or ONNX Runtime requires changes only in this layer
- Business logic remains free of framework-specific types
- Platform differences (macOS keychain vs. Windows Credential Manager) are handled in one place

---

## Directory Structure

```
internal/infrastructure/
├── onnx/       # Hugot ONNX Runtime inference runtime wrapper
├── database/   # DuckDB / SQLite connection pool, migrations, and transaction management
├── config/     # Viper configuration loading and validation
└── secret/     # System keychain wrapper (macOS Keychain / Windows Credential / Linux Secret Service)
```

| Package | Purpose | Example Types |
|---------|---------|--------------|
| `onnx/` | Local AI model inference | `ONNXRuntime`, `InferenceWorker` |
| `database/` | Database connectivity | `DuckDBConnector`, `SQLiteConnector` |
| `config/` | Application configuration | `AppConfig`, `Load()` |
| `secret/` | Secure credential storage | `KeychainStore` |

---

## Import Constraints (Iron Rule)

| Allowed Imports | Forbidden Imports |
|-----------------|-------------------|
| Go standard library | `github.com/hzhan516/medmemo/internal/domain/*` |
| Third-party frameworks (Wails, DuckDB, Viper, Hugot...) | `github.com/hzhan516/medmemo/internal/application/*` |
| — | `github.com/hzhan516/medmemo/internal/adapters/*` |

> ⚠️ If this layer imports any business package, it breaks Clean Architecture's dependency direction.

---

## Core Responsibilities

1. **Framework Initialization** — Create database connection pools, load ONNX Runtime, parse configuration files
2. **Resource Management** — Provide `Close()` / `Shutdown()` methods for graceful release
3. **Platform Abstraction** — Shield cross-platform differences (keychain APIs, dynamic library paths) from upper layers

---

## Design Principles

- **Expose concrete types**. This layer returns concrete types (e.g., `*sql.DB`, `*onnx.Runtime`), not interfaces. Interfaces are defined in `application/port/`.
- **Configuration-driven**. All tunables (timeouts, connection counts, paths) are loaded through `config/`. No hard-coded values.
- **Fail fast**. Validate dependencies at initialization time (database connectivity, model file existence) to avoid runtime surprises.

---

## Example

```go
// database/duckdb.go
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

---

## Related Layers

- [Adapters Layer](../adapters/README.md) — Consumes the technical capabilities this layer provides
- [Application Layer](../application/README.md) — Defines the workflows that ultimately use these capabilities
- [Domain Layer](../domain/README.md) — Holds the business rules completely independent of this layer

---

*Last updated: 2026-05-19*
