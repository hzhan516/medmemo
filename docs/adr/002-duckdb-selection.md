# ADR-002: Deferring DuckDB/Kùzǔ to v2+ Planning

> 🌐 [中文版本](../i18n/zh-Hans-CN/adr/002-duckdb-selection.md)

- **Status**: Superseded for v1.x
- **Date**: 2026-05
- **Last updated**: 2026-07-09
- **Deciders**: Backend Tech Lead, AI Engineer

## Context

The original storage design considered DuckDB/Kùzǔ for v2+ analytical and graph workloads: DuckDB for future family-health analytics and vector experiments, and Kùzǔ for future family relationship graph traversal.

The shipped v1.x product uses a smaller local-first storage stack:

- SQLCipher/SQLite for conversations, messages, provider settings, facts, audit logs, and knowledge documents.
- sqlite-vec for semantic vector search over approved facts and local knowledge documents.
- Frozen DuckDB/Kùzǔ stubs only for v2+ planning; they are not active in v1.x runtime.

## Decision

Supersede the original active DuckDB/Kùzǔ storage decision for v1.x. Keep DuckDB/Kùzǔ as v2+ planning candidates only, and treat SQLCipher/SQLite + sqlite-vec as the v1.x source of truth.

## Consequences

- v1.x packaging stays smaller and avoids additional CGO/runtime libraries for DuckDB/Kùzǔ.
- The storage and migration surface is simpler: encrypted SQLite plus sqlite-vec.
- Future v2+ family graph or analytics work must reopen this ADR with implementation evidence, migration design, and package-size impact.

## Related Documents

- [docs/ARCHITECTURE.md](../ARCHITECTURE.md) — Current storage architecture and ADR index.
- `internal/infrastructure/database/` — SQLCipher/SQLite connectors.
- `internal/adapters/repository/` — SQLite repository implementations.
