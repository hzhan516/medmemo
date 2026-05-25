// Package repository 实现数据持久化适配器。
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/wire"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
)

// EmbeddingRepoSQLite 基于 SQLite 的嵌入向量仓库实现。
type EmbeddingRepoSQLite struct {
	db *sql.DB
}

// NewEmbeddingRepoSQLite 构造函数。
func NewEmbeddingRepoSQLite(connector database.DBConnector) *EmbeddingRepoSQLite {
	return &EmbeddingRepoSQLite{db: connector.DB()}
}

// Save 保存嵌入向量。
func (r *EmbeddingRepoSQLite) Save(ctx context.Context, e *entity.SemanticEmbedding) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO semantic_embeddings (embedding_id, fact_id, vector, model_version, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, e.EmbeddingID, e.FactID, e.VectorToBytes(), e.ModelVersion, e.CreatedAt.UnixMilli())
	if err != nil {
		return fmt.Errorf("failed to save embedding: %w", err)
	}
	return nil
}

// GetByFactID 按事实 ID 查询嵌入。
func (r *EmbeddingRepoSQLite) GetByFactID(ctx context.Context, factID string) (*entity.SemanticEmbedding, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT embedding_id, fact_id, vector, model_version, created_at
		FROM semantic_embeddings WHERE fact_id = ?
	`, factID)

	var e entity.SemanticEmbedding
	var vectorBytes []byte
	var created int64

	if err := row.Scan(&e.EmbeddingID, &e.FactID, &vectorBytes, &e.ModelVersion, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("embedding not found: %w", entity.ErrEmbeddingNotFound)
		}
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	vector, err := entity.VectorFromBytes(vectorBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode embedding vector: %w", err)
	}
	e.Vector = vector
	e.CreatedAt = time.UnixMilli(created).UTC()
	return &e, nil
}

// DeleteByFactID 按事实 ID 删除嵌入。
func (r *EmbeddingRepoSQLite) DeleteByFactID(ctx context.Context, factID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM semantic_embeddings WHERE fact_id = ?`, factID)
	if err != nil {
		return fmt.Errorf("failed to delete embedding: %w", err)
	}
	return nil
}

// EmbeddingRepoSet 供 Wire 使用的 ProviderSet。
var EmbeddingRepoSet = wire.NewSet(
	NewEmbeddingRepoSQLite,
)
