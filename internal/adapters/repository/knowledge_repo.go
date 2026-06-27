// Package repository 实现数据持久化适配器。
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
)

// KnowledgeRepoSQLite 基于 SQLite/SQLCipher 的知识库实现。
type KnowledgeRepoSQLite struct {
	db *sql.DB
}

// NewKnowledgeRepoSQLite 构造函数。
func NewKnowledgeRepoSQLite(connector database.DBConnector) *KnowledgeRepoSQLite {
	return &KnowledgeRepoSQLite{db: connector.DB()}
}

// SaveDocument 保存或更新文档。
func (r *KnowledgeRepoSQLite) SaveDocument(ctx context.Context, doc *entity.KnowledgeDocument) error {
	now := doc.UpdatedAt.UnixMilli()
	_, err := r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO knowledge_documents
		(document_id, title, source_type, citation, url, language, checksum, metadata_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, doc.DocumentID, doc.Title, string(doc.SourceType), doc.Citation, doc.URL, doc.Language, doc.Checksum, doc.MetadataJSON, doc.CreatedAt.UnixMilli(), now)
	if err != nil {
		return fmt.Errorf("failed to save knowledge document: %w", err)
	}
	return nil
}

// SaveChunks 批量保存片段。
func (r *KnowledgeRepoSQLite) SaveChunks(ctx context.Context, chunks []*entity.KnowledgeChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin chunks transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR REPLACE INTO knowledge_chunks
		(chunk_id, document_id, chunk_index, content, token_count, metadata_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare chunks statement: %w", err)
	}
	defer stmt.Close()

	for _, c := range chunks {
		if _, err := stmt.ExecContext(ctx, c.ChunkID, c.DocumentID, c.ChunkIndex, c.Content, c.TokenCount, c.MetadataJSON, c.CreatedAt.UnixMilli()); err != nil {
			return fmt.Errorf("failed to save chunk %s: %w", c.ChunkID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit chunks transaction: %w", err)
	}
	return nil
}

// FindDocument 按 ID 查询文档。
func (r *KnowledgeRepoSQLite) FindDocument(ctx context.Context, id string) (*entity.KnowledgeDocument, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT document_id, title, source_type, citation, url, language, checksum, metadata_json, created_at, updated_at
		FROM knowledge_documents
		WHERE document_id = ?
	`, id)
	return r.scanDocument(row)
}

// ListDocuments 列出全部文档。
func (r *KnowledgeRepoSQLite) ListDocuments(ctx context.Context) ([]*entity.KnowledgeDocument, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT document_id, title, source_type, citation, url, language, checksum, metadata_json, created_at, updated_at
		FROM knowledge_documents
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list knowledge documents: %w", err)
	}
	defer rows.Close()

	var result []*entity.KnowledgeDocument
	for rows.Next() {
		doc, err := r.scanDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan knowledge document: %w", err)
		}
		result = append(result, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate knowledge documents: %w", err)
	}
	return result, nil
}

// DeleteDocument 级联删除文档及其关联数据。
func (r *KnowledgeRepoSQLite) DeleteDocument(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin delete transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM knowledge_terms WHERE document_id = ?`, id); err != nil {
		return fmt.Errorf("failed to delete knowledge terms: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM knowledge_embeddings WHERE chunk_id IN (SELECT chunk_id FROM knowledge_chunks WHERE document_id = ?)`, id); err != nil {
		return fmt.Errorf("failed to delete knowledge embeddings: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM knowledge_chunks WHERE document_id = ?`, id); err != nil {
		return fmt.Errorf("failed to delete knowledge chunks: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM knowledge_documents WHERE document_id = ?`, id); err != nil {
		return fmt.Errorf("failed to delete knowledge document: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit delete transaction: %w", err)
	}
	return nil
}

// FindDocumentByChecksum 按 checksum 查询文档。
func (r *KnowledgeRepoSQLite) FindDocumentByChecksum(ctx context.Context, checksum string) (*entity.KnowledgeDocument, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT document_id, title, source_type, citation, url, language, checksum, metadata_json, created_at, updated_at
		FROM knowledge_documents
		WHERE checksum = ?
	`, checksum)
	return r.scanDocument(row)
}

// CountChunks 返回片段总数。
func (r *KnowledgeRepoSQLite) CountChunks(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT chunk_id) FROM knowledge_chunks`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count chunks: %w", err)
	}
	return count, nil
}

// CountTermDF 返回词项的文档频率。
func (r *KnowledgeRepoSQLite) CountTermDF(ctx context.Context, term string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT chunk_id) FROM knowledge_terms WHERE term = ?`, term).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count term df: %w", err)
	}
	return count, nil
}

// AverageChunkTokenCount 返回平均 token 数。
func (r *KnowledgeRepoSQLite) AverageChunkTokenCount(ctx context.Context) (float64, error) {
	var avg sql.NullFloat64
	err := r.db.QueryRowContext(ctx, `SELECT AVG(token_count) FROM knowledge_chunks`).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("failed to get average chunk token count: %w", err)
	}
	if !avg.Valid {
		return 0, nil
	}
	return avg.Float64, nil
}

// GetChunkTokenCount 返回指定片段的 token 数。
func (r *KnowledgeRepoSQLite) GetChunkTokenCount(ctx context.Context, chunkID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT token_count FROM knowledge_chunks WHERE chunk_id = ?`, chunkID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get chunk token count: %w", err)
	}
	return count, nil
}

// SaveTerms 保存片段词项频率。
func (r *KnowledgeRepoSQLite) SaveTerms(ctx context.Context, chunkID, documentID string, termFreq map[string]int) error {
	if len(termFreq) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin terms transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM knowledge_terms WHERE chunk_id = ?`, chunkID); err != nil {
		return fmt.Errorf("failed to clear old terms: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO knowledge_terms (term, chunk_id, tf, document_id)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare terms statement: %w", err)
	}
	defer stmt.Close()

	for term, tf := range termFreq {
		if _, err := stmt.ExecContext(ctx, term, chunkID, tf, documentID); err != nil {
			return fmt.Errorf("failed to save term %s: %w", term, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit terms transaction: %w", err)
	}
	return nil
}

// SaveImportJob 保存导入任务。
func (r *KnowledgeRepoSQLite) SaveImportJob(ctx context.Context, job *entity.KnowledgeImportJob) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO knowledge_import_jobs
		(job_id, status, total, processed, error_message, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, job.JobID, string(job.Status), job.Total, job.Processed, job.ErrorMessage, job.CreatedAt.UnixMilli(), job.UpdatedAt.UnixMilli())
	if err != nil {
		return fmt.Errorf("failed to save knowledge import job: %w", err)
	}
	return nil
}

// GetImportJob 按 ID 查询导入任务。
func (r *KnowledgeRepoSQLite) GetImportJob(ctx context.Context, id string) (*entity.KnowledgeImportJob, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT job_id, status, total, processed, error_message, created_at, updated_at
		FROM knowledge_import_jobs
		WHERE job_id = ?
	`, id)
	var job entity.KnowledgeImportJob
	var status string
	var createdAt, updatedAt int64
	if err := row.Scan(&job.JobID, &status, &job.Total, &job.Processed, &job.ErrorMessage, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("knowledge import job %s not found: %w", id, entity.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get knowledge import job: %w", err)
	}
	job.Status = entity.KnowledgeImportStatus(status)
	job.CreatedAt = time.UnixMilli(createdAt)
	job.UpdatedAt = time.UnixMilli(updatedAt)
	return &job, nil
}

func (r *KnowledgeRepoSQLite) scanDocument(scanner interface {
	Scan(dest ...interface{}) error
}) (*entity.KnowledgeDocument, error) {
	var doc entity.KnowledgeDocument
	var sourceType string
	var createdAt, updatedAt int64
	if err := scanner.Scan(&doc.DocumentID, &doc.Title, &sourceType, &doc.Citation, &doc.URL, &doc.Language, &doc.Checksum, &doc.MetadataJSON, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	doc.SourceType = entity.KnowledgeSourceType(sourceType)
	doc.CreatedAt = time.UnixMilli(createdAt)
	doc.UpdatedAt = time.UnixMilli(updatedAt)
	return &doc, nil
}

// Compile-time interface check.
var _ repository.KnowledgeRepository = (*KnowledgeRepoSQLite)(nil)
