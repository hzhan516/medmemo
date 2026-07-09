// Package repository 实现数据持久化适配器。
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/wire"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/viant/sqlite-vec/vector"
)

// EmbeddingRepoSQLite 基于 SQLite 的嵌入向量仓库实现。
type EmbeddingRepoSQLite struct {
	db                 *sql.DB
	vectorSQLAvailable bool
}

// NewEmbeddingRepoSQLite 构造函数。
func NewEmbeddingRepoSQLite(connector database.DBConnector) *EmbeddingRepoSQLite {
	db := connector.DB()
	return &EmbeddingRepoSQLite{
		db:                 db,
		vectorSQLAvailable: detectVectorSQLAvailable(db),
	}
}

// Save 保存嵌入向量。
func (r *EmbeddingRepoSQLite) Save(ctx context.Context, e *entity.SemanticEmbedding) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO semantic_embeddings (embedding_id, fact_id, vector, model_version, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, e.EmbeddingID, e.FactID, e.VectorToBytes(), e.ModelVersion, e.CreatedAt.UnixMilli())
	if err != nil {
		if database.IsSQLiteUniqueConstraintOn(err, "semantic_embeddings.fact_id") {
			return nil
		}
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("embedding not found: %w", entity.ErrEmbeddingNotFound)
		}
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	vec, err := entity.VectorFromBytes(vectorBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode embedding vector: %w", err)
	}
	e.Vector = vec
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

// SearchSimilar 执行向量相似度搜索，返回与查询向量最相似的 topK 个嵌入（含相似度分数）。
func (r *EmbeddingRepoSQLite) SearchSimilar(ctx context.Context, queryVector []float32, topK int) ([]*entity.ScoredEmbedding, error) {
	if topK <= 0 {
		return nil, nil
	}
	if !r.vectorSQLAvailable {
		return r.searchSimilarInGo(ctx, queryVector, topK, "")
	}

	queryBlob, err := vector.EncodeEmbedding(queryVector)
	if err != nil {
		return nil, fmt.Errorf("failed to encode query vector: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT embedding_id, fact_id, vector, model_version, created_at,
		       vec_cosine(?, vector) as similarity
		FROM semantic_embeddings
		ORDER BY similarity DESC
		LIMIT ?
	`, queryBlob, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar embeddings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return r.scanScoredEmbeddings(rows)
}

// SearchSimilarFiltered 执行带 model_version 过滤的向量相似度搜索。
// 当 modelVersion 为空时退化为 SearchSimilar（搜索所有版本）。
func (r *EmbeddingRepoSQLite) SearchSimilarFiltered(ctx context.Context, queryVector []float32, topK int, modelVersion string) ([]*entity.ScoredEmbedding, error) {
	if topK <= 0 {
		return nil, nil
	}
	if modelVersion == "" {
		return r.SearchSimilar(ctx, queryVector, topK)
	}
	if !r.vectorSQLAvailable {
		return r.searchSimilarInGo(ctx, queryVector, topK, modelVersion)
	}

	queryBlob, err := vector.EncodeEmbedding(queryVector)
	if err != nil {
		return nil, fmt.Errorf("failed to encode query vector: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT embedding_id, fact_id, vector, model_version, created_at,
		       vec_cosine(?, vector) as similarity
		FROM semantic_embeddings
		WHERE model_version = ?
		ORDER BY similarity DESC
		LIMIT ?
	`, queryBlob, modelVersion, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar embeddings with filter: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return r.scanScoredEmbeddings(rows)
}

// CountByVersionNot 统计 model_version 不等于指定版本的 embedding 数量。
func (r *EmbeddingRepoSQLite) CountByVersionNot(ctx context.Context, version string) (int64, error) {
	var count int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM semantic_embeddings WHERE model_version != ?
	`, version).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count embeddings by version: %w", err)
	}
	return count, nil
}

// UpdateEmbedding 原子更新已有 embedding 的向量、版本和时间戳（保留 embedding_id）。
func (r *EmbeddingRepoSQLite) UpdateEmbedding(ctx context.Context, e *entity.SemanticEmbedding) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE semantic_embeddings 
		SET vector = ?, model_version = ?, created_at = ?
		WHERE fact_id = ?
	`, e.VectorToBytes(), e.ModelVersion, e.CreatedAt.UnixMilli(), e.FactID)
	if err != nil {
		return fmt.Errorf("failed to update embedding: %w", err)
	}
	return nil
}

func detectVectorSQLAvailable(db *sql.DB) bool {
	a := make([]float32, entity.EmbeddingDimension)
	a[0] = 1
	b := make([]float32, entity.EmbeddingDimension)
	b[0] = 1

	aBlob, err := vector.EncodeEmbedding(a)
	if err != nil {
		return false
	}
	bBlob, err := vector.EncodeEmbedding(b)
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var similarity float64
	if err := db.QueryRowContext(ctx, `SELECT vec_cosine(?, ?)`, aBlob, bBlob).Scan(&similarity); err != nil {
		return false
	}
	return true
}

// scanScoredEmbeddings 扫描 scored embedding 行并反序列化向量。
func (r *EmbeddingRepoSQLite) scanScoredEmbeddings(rows *sql.Rows) ([]*entity.ScoredEmbedding, error) {
	var results []*entity.ScoredEmbedding
	for rows.Next() {
		var e entity.SemanticEmbedding
		var vectorBytes []byte
		var created int64
		var similarity float64

		if err := rows.Scan(&e.EmbeddingID, &e.FactID, &vectorBytes, &e.ModelVersion, &created, &similarity); err != nil {
			return nil, fmt.Errorf("failed to scan embedding row: %w", err)
		}

		vec, err := entity.VectorFromBytes(vectorBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to decode embedding vector: %w", err)
		}
		e.Vector = vec
		e.CreatedAt = time.UnixMilli(created).UTC()
		results = append(results, &entity.ScoredEmbedding{
			SemanticEmbedding: &e,
			Similarity:        similarity,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate embedding rows: %w", err)
	}

	return results, nil
}

// searchSimilarInGo 在 Go 中完成相似度计算；modelVersion 为空时搜索全部版本。
func (r *EmbeddingRepoSQLite) searchSimilarInGo(ctx context.Context, queryVector []float32, topK int, modelVersion string) ([]*entity.ScoredEmbedding, error) {
	query := `SELECT embedding_id, fact_id, vector, model_version, created_at FROM semantic_embeddings`
	var args []any
	if modelVersion != "" {
		query += " WHERE model_version = ?"
		args = append(args, modelVersion)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list embeddings for fallback search: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []*entity.ScoredEmbedding
	for rows.Next() {
		var e entity.SemanticEmbedding
		var vectorBytes []byte
		var created int64

		if err := rows.Scan(&e.EmbeddingID, &e.FactID, &vectorBytes, &e.ModelVersion, &created); err != nil {
			return nil, fmt.Errorf("failed to scan embedding row for fallback search: %w", err)
		}

		vec, err := entity.VectorFromBytes(vectorBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to decode embedding vector for fallback search: %w", err)
		}
		similarity, err := cosineSimilarity(queryVector, vec)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate fallback cosine similarity: %w", err)
		}

		e.Vector = vec
		e.CreatedAt = time.UnixMilli(created).UTC()
		results = append(results, &entity.ScoredEmbedding{
			SemanticEmbedding: &e,
			Similarity:        similarity,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate embedding rows for fallback search: %w", err)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

func cosineSimilarity(a, b []float32) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("dimension mismatch %d vs %d", len(a), len(b))
	}
	if len(a) == 0 {
		return 0, fmt.Errorf("empty vectors")
	}

	var dot, normA, normB float64
	for i := range a {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		normA += av * av
		normB += bv * bv
	}
	if normA == 0 || normB == 0 {
		return 0, fmt.Errorf("zero-magnitude vector")
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB)), nil
}

// EmbeddingRepoSet 供 Wire 使用的 ProviderSet。
var EmbeddingRepoSet = wire.NewSet(
	NewEmbeddingRepoSQLite,
)
