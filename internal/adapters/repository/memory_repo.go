// Package repository 实现数据持久化适配器。
// 将 domain 实体映射到具体数据库（DuckDB / SQLite / Kùzǔ）。
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/wire"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/hzhan516/medmemo/pkg/models"
)

// MemoryRepoSQLite 基于 SQLite 的记忆仓库实现（DuckDB 降级方案）。
// 待 DuckDB 驱动引入后，可替换为 DuckDB 实现以支持向量检索 [Issue#025]。
type MemoryRepoSQLite struct {
	db *sql.DB
}

// NewMemoryRepoSQLite 构造函数，复用 DBConnector。
func NewMemoryRepoSQLite(connector database.DBConnector) *MemoryRepoSQLite {
	return &MemoryRepoSQLite{db: connector.DB()}
}

// Save 保存或更新记忆。
func (r *MemoryRepoSQLite) Save(ctx context.Context, mem *entity.HealthMemory) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO memories (id, tier, content, tags, source_conv, confidence, created_at, accessed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, mem.ID, mem.Tier, mem.Content, strings.Join(mem.Tags, ","), mem.SourceConv, mem.Confidence,
		mem.CreatedAt.UnixMilli(), mem.AccessedAt.UnixMilli())
	if err != nil {
		return fmt.Errorf("failed to save memory: %w", err)
	}
	return nil
}

// GetByID 按 ID 查询记忆。
func (r *MemoryRepoSQLite) GetByID(ctx context.Context, id models.MemoryID) (*entity.HealthMemory, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, tier, content, tags, source_conv, confidence, created_at, accessed_at
		FROM memories WHERE id = ?
	`, id)

	var mem entity.HealthMemory
	var tags string
	var createdAt, accessedAt int64
	if err := row.Scan(&mem.ID, &mem.Tier, &mem.Content, &tags, &mem.SourceConv, &mem.Confidence, &createdAt, &accessedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("memory %s not found: %w", id, entity.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get memory: %w", err)
	}
	mem.Tags = splitTags(tags)
	mem.CreatedAt = msToTime(createdAt)
	mem.AccessedAt = msToTime(accessedAt)
	return &mem, nil
}

// Search 关键词搜索记忆（简单 LIKE 匹配）。
func (r *MemoryRepoSQLite) Search(ctx context.Context, query string, limit int) ([]*entity.HealthMemory, error) {
	pattern := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tier, content, tags, source_conv, confidence, created_at, accessed_at
		FROM memories WHERE content LIKE ? OR tags LIKE ?
		ORDER BY accessed_at DESC
		LIMIT ?
	`, pattern, pattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}
	defer rows.Close()
	return scanMemories(rows)
}

// SemanticSearch 语义向量检索。
// 当前 SQLite 降级实现不支持向量索引，直接返回错误。
// 注：项目已迁移至 sqlite-vec 方案，SemanticSearch 在此降级实现中保留为占位。
func (r *MemoryRepoSQLite) SemanticSearch(ctx context.Context, embedding []float32, topK int) ([]*entity.HealthMemory, error) {
	return nil, fmt.Errorf("semantic search not available in SQLite fallback, use sqlite-vec via EmbeddingRepository instead: %w", entity.ErrNotFound)
}

// ListByTier 按记忆层级查询。
func (r *MemoryRepoSQLite) ListByTier(ctx context.Context, tier entity.MemoryTier, limit int) ([]*entity.HealthMemory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, tier, content, tags, source_conv, confidence, created_at, accessed_at
		FROM memories WHERE tier = ?
		ORDER BY accessed_at DESC
		LIMIT ?
	`, tier, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list memories by tier: %w", err)
	}
	defer rows.Close()
	return scanMemories(rows)
}

// Delete 删除记忆。
func (r *MemoryRepoSQLite) Delete(ctx context.Context, id models.MemoryID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM memories WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}
	return nil
}

// EnsureMemorySchema 确保 memories 表存在（供数据库迁移调用）。
func EnsureMemorySchema(db *sql.DB) error {
	_, err := db.Exec(`
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
	`)
	if err != nil {
		return fmt.Errorf("failed to ensure memory schema: %w", err)
	}
	return nil
}

// scanMemories 扫描查询结果为记忆列表。
func scanMemories(rows *sql.Rows) ([]*entity.HealthMemory, error) {
	var result []*entity.HealthMemory
	for rows.Next() {
		var mem entity.HealthMemory
		var tags string
		var createdAt, accessedAt int64
		if err := rows.Scan(&mem.ID, &mem.Tier, &mem.Content, &tags, &mem.SourceConv, &mem.Confidence, &createdAt, &accessedAt); err != nil {
			return nil, fmt.Errorf("failed to scan memory: %w", err)
		}
		mem.Tags = splitTags(tags)
		mem.CreatedAt = msToTime(createdAt)
		mem.AccessedAt = msToTime(accessedAt)
		result = append(result, &mem)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate memories: %w", err)
	}
	return result, nil
}

func splitTags(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}

func msToTime(ms int64) time.Time {
	return time.UnixMilli(ms)
}

// DuckDBConnection 占位类型，待 infrastructure/database 完成后替换 [Issue#025]。
type DuckDBConnection struct{}

// RepositorySet 供 Wire 使用的 ProviderSet。
// 当前使用 SQLite 降级实现 MemoryRepo。
var RepositorySet = wire.NewSet(
	NewMemoryRepoSQLite,
	NewConversationRepoSQLite,
	NewMessageRepoSQLite,
	NewDisclaimerRepoSQLite,
	NewProviderRepoSQLite,
)
