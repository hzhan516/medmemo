# ADR-002: Adopting DuckDB as the Embedded Analytics Database

> 🌐 [中文版本](../i18n/zh-Hans-CN/adr/002-duckdb-selection.md)

- **Status**: Accepted
- **Date**: 2026-05
- **Deciders**: Backend Tech Lead, AI Engineer

## Context

MedMemo is a desktop health information tool with a **three-tier memory architecture** (L1 working memory, L2 short-term memory, L3 long-term memory) and a **family health graph** that requires both transactional persistence and analytical query capabilities. The core storage requirements are:

1. **Transactional data**: Conversation history, message records, user settings — high write throughput, ACID guarantees, small result sets.
2. **Analytical queries**: Family health statistics (e.g., "what diseases appear most frequently across three generations?"), risk scoring aggregations, time-series analysis — complex GROUP BY, JOINs, and window functions.
3. **Vector retrieval**: L3 long-term memory requires semantic similarity search over embedding vectors (100-dim float32) for memory retrieval-augmented generation (RAG).
4. **Graph queries**: Family relationship network uses property-graph model with Cypher queries — handled by a dedicated graph database.

Before project kickoff, the team evaluated four embedded database options:

| Database | Pros | Cons | Suitability |
|:---------|:-----|:-----|:------------|
| **SQLite** | Mature, battle-tested, tiny footprint (~1MB), excellent transactional performance | No native vector indexing; analytical queries (complex GROUP BY, window functions) significantly slower than columnar stores; fts5 limited to keyword search | ⚠️ Good for transactional layer, insufficient for analytics + vectors |
| **DuckDB** | In-process embedded analytics engine; columnar storage; vector search via `vss` extension; excellent window functions and aggregations; small footprint (~20MB) | Relatively young (v1.7); write-heavy transactional workloads not its primary design target; Go binding via CGO adds build complexity | ✅ Best fit for analytics + vector layers |
| **PostgreSQL + pgvector** | Mature vector support; rich analytical functions; strong ecosystem | Requires external server process; violates offline-first principle; deployment complexity incompatible with desktop single-binary model | ❌ Violates embedded / offline-first constraint |
| **ClickHouse Local** | Extreme analytical performance; vector support via Annoy index | Overkill for desktop scale; large binary (~200MB); steep learning curve | ❌ Too heavy for <150MB total deployment budget |

## Decision

Adopt **DuckDB v1.7+** as the embedded analytics database for MedMemo, used in conjunction with **SQLite** (transactional storage) and **Kùzǔ** (graph database). The three databases are assigned distinct responsibilities:

```
Storage Stack:
├── SQLite / SQLCipher     # Transactional: conversations, messages, settings
├── DuckDB                 # Analytics + Vector: L3 memory, embeddings, aggregations
└── Kùzǔ                   # Graph: family relationships, Cypher queries
```

### Assignment of Responsibilities

| Data Category | Primary Store | Reason |
|:--------------|:--------------|:-------|
| Conversation & message records | SQLite | High-frequency writes, ACID reliability, proven stability |
| L3 long-term memory embeddings | DuckDB | Vector similarity search via `vss` extension; analytical aggregations for memory statistics |
| Family health statistics / risk scores | DuckDB | Complex analytical queries (GROUP BY, window functions) over multi-generation data |
| Family relationship graph | Kùzǔ | Native property-graph model, Cypher query language, efficient graph traversals |

### Key Technical Choices

1. **DuckDB `vss` extension for vector indexing**: HNSW (Hierarchical Navigable Small World) index on `FLOAT[100]` embedding columns, targeting <30ms Top-K retrieval over 100k memory records (PER-06).

2. **DuckDB memory mode for unit tests**: Integration tests use `:memory:` DuckDB instances, avoiding file I/O and test isolation issues.

3. **Write serialization via single Goroutine**: DuckDB write operations are serialized through a single dedicated Goroutine. Read operations leverage DuckDB's MVCC for safe concurrent access (see ADR-004 runtime model).

4. **Go binding via `github.com/marcboeker/go-duckdb`**: CGO-based binding that exposes DuckDB's full SQL interface through the standard `database/sql` driver.

## Consequences

### Positive Impacts

- **Analytical queries run 10–50× faster than SQLite** for aggregation and window-function workloads, directly improving family health statistics response time.
- **Vector search is native**: The `vss` extension eliminates the need for a separate vector database (e.g., Milvus, Pinecone), keeping the deployment footprint under 150MB.
- **Single SQL dialect**: DuckDB speaks standard SQL, so analytical queries can be written and tested directly without learning a new query language.
- **Seamless CSV/JSON/Parquet ingestion**: Future features (e.g., importing health examination reports) can leverage DuckDB's built-in format readers.

### Negative Impacts

- **CGO build complexity**: Cross-compilation for Windows/macOS/Linux requires platform-specific DuckDB static libraries, increasing CI build time.
- **Write contention**: DuckDB is optimized for read-heavy analytical workloads; high-frequency transactional writes (e.g., every keystroke) must be batched or routed to SQLite to avoid contention.
- **Young ecosystem**: DuckDB v1.7 is less mature than SQLite v3.45; edge-case bugs in the Go binding or `vss` extension may require upstream fixes or workarounds.
- **Three databases to maintain**: Operations, migrations, and backups now span SQLite, DuckDB, and Kùzǔ, increasing operational surface area compared to a single-store design.

## Alternatives Considered

| Alternative | Rejection Reason |
|:------------|:-----------------|
| SQLite for everything | Lacks vector indexing and analytical query performance; complex family statistics queries would be unacceptably slow |
| PostgreSQL (embedded via `postgres-lite`) | Still requires a background server process; violates the "single binary, no daemon" desktop deployment constraint |
| Single graph database (Kùzǔ) for everything | Kùzǔ excels at graph traversals but is not designed for high-throughput transactional writes or vector similarity search |

## Related Documents

- [docs/ARCHITECTURE.md](../ARCHITECTURE.md) — System architecture overview and storage stack
- `internal/infrastructure/database/` — DuckDB connection pool and migration code
- `internal/adapters/repository/memory_repo.go` — DuckDB-based memory repository implementation
