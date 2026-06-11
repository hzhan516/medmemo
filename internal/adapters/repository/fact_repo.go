// Package repository 实现数据持久化适配器。
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/wire"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
)

// FactRepoSQLite 基于 SQLite 的事实仓库实现。
type FactRepoSQLite struct {
	db *sql.DB
}

// NewFactRepoSQLite 构造函数。
func NewFactRepoSQLite(connector database.DBConnector) *FactRepoSQLite {
	return &FactRepoSQLite{db: connector.DB()}
}

// Save 保存事实记录。
func (r *FactRepoSQLite) Save(ctx context.Context, f *entity.ExtractedFact) error {
	sourceIDs, err := json.Marshal(f.SourceMsgIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal source_msg_ids: %w", err)
	}

	var scoredAt, reviewedAt *int64
	if f.ScoredAt != nil {
		v := f.ScoredAt.UnixMilli()
		scoredAt = &v
	}
	if f.ReviewedAt != nil {
		v := f.ReviewedAt.UnixMilli()
		reviewedAt = &v
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO extracted_facts (fact_id, subject, predicate, object, confidence, source_msg_ids, status, is_sensitive, scored_at, reviewed_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, f.FactID, f.Subject, f.Predicate, f.Object, f.Confidence, sourceIDs, f.Status, boolToInt(f.IsSensitive), scoredAt, reviewedAt, f.CreatedAt.UnixMilli())
	if err != nil {
		if database.IsSQLitePrimaryOrUniqueConstraintOn(err, "extracted_facts.fact_id") {
			return nil
		}
		return fmt.Errorf("failed to save fact: %w", err)
	}
	return nil
}

// FindLatestApprovedByPredicates 按 subject 和多个 predicate 查找最新已审批事实。
func (r *FactRepoSQLite) FindLatestApprovedByPredicates(ctx context.Context, subject string, predicates []string) (*entity.ExtractedFact, error) {
	if len(predicates) == 0 {
		return nil, entity.ErrFactNotFound
	}

	placeholders := make([]string, len(predicates))
	args := make([]any, 0, len(predicates)+1)
	args = append(args, subject)
	for i, p := range predicates {
		placeholders[i] = "?"
		args = append(args, p)
	}

	query := fmt.Sprintf(`
		SELECT fact_id, subject, predicate, object, confidence, source_msg_ids, status, is_sensitive, scored_at, reviewed_at, created_at
		FROM extracted_facts
		WHERE status = 'approved' AND subject = ? AND predicate IN (%s)
		ORDER BY created_at DESC
		LIMIT 1
	`, strings.Join(placeholders, ","))

	row := r.db.QueryRowContext(ctx, query, args...)
	return scanFact(row)
}

// GetByID 按 ID 查询事实。
func (r *FactRepoSQLite) GetByID(ctx context.Context, factID string) (*entity.ExtractedFact, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT fact_id, subject, predicate, object, confidence, source_msg_ids, status, is_sensitive, scored_at, reviewed_at, created_at
		FROM extracted_facts WHERE fact_id = ?
	`, factID)

	return scanFact(row)
}

// ListByStatus 按审核状态分页查询。
func (r *FactRepoSQLite) ListByStatus(ctx context.Context, status entity.FactStatus, offset, limit int) ([]*entity.ExtractedFact, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT fact_id, subject, predicate, object, confidence, source_msg_ids, status, is_sensitive, scored_at, reviewed_at, created_at
		FROM extracted_facts
		WHERE status = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list facts by status: %w", err)
	}
	defer rows.Close()
	return scanFacts(rows)
}

// ListPending 获取待审核列表。
func (r *FactRepoSQLite) ListPending(ctx context.Context, offset, limit int) ([]*entity.ExtractedFact, error) {
	return r.ListByStatus(ctx, entity.FactStatusPending, offset, limit)
}

// UpdateStatus 更新审核状态。
func (r *FactRepoSQLite) UpdateStatus(ctx context.Context, factID string, status entity.FactStatus) error {
	reviewedAt := time.Now().UTC().UnixMilli()
	_, err := r.db.ExecContext(ctx, `
		UPDATE extracted_facts SET status = ?, reviewed_at = ? WHERE fact_id = ?
	`, status, reviewedAt, factID)
	if err != nil {
		return fmt.Errorf("failed to update fact status: %w", err)
	}
	return nil
}

// Delete 删除事实及关联嵌入向量。
// 使用事务显式删除 semantic_embeddings 和 extracted_facts，不依赖外键级联作为唯一保障，
// 避免连接池场景或外键未启用时留下 stale embedding 影响召回。
func (r *FactRepoSQLite) Delete(ctx context.Context, factID string) (err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin delete transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(ctx, `DELETE FROM semantic_embeddings WHERE fact_id = ?`, factID); err != nil {
		return fmt.Errorf("failed to delete embedding: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM extracted_facts WHERE fact_id = ?`, factID); err != nil {
		return fmt.Errorf("failed to delete fact: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit delete transaction: %w", err)
	}
	return nil
}

// ListAllSubjects 获取所有已审批事实的不重复 subject 列表。
func (r *FactRepoSQLite) ListAllSubjects(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT subject FROM extracted_facts WHERE status = 'approved' AND subject != ''
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list subjects: %w", err)
	}
	defer rows.Close()

	var subjects []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("failed to scan subject: %w", err)
		}
		subjects = append(subjects, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate subjects: %w", err)
	}
	return subjects, nil
}

// FindBySubject 按 subject 查找已审批事实。
func (r *FactRepoSQLite) FindBySubject(ctx context.Context, subject string) ([]*entity.ExtractedFact, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT fact_id, subject, predicate, object, confidence, source_msg_ids, status, is_sensitive, scored_at, reviewed_at, created_at
		FROM extracted_facts
		WHERE status = 'approved' AND subject = ?
		ORDER BY created_at DESC
	`, subject)
	if err != nil {
		return nil, fmt.Errorf("failed to find facts by subject: %w", err)
	}
	defer rows.Close()
	return scanFacts(rows)
}

// FindBySession 按原始对话会话 ID 查找关联的已审批事实。
// 通过 source_msg_ids 与 raw_dialogues 关联。
func (r *FactRepoSQLite) FindBySession(ctx context.Context, sessionID string) ([]*entity.ExtractedFact, error) {
	// 先获取该会话的所有 message_id
	msgRows, err := r.db.QueryContext(ctx, `
		SELECT message_id FROM raw_dialogues WHERE session_id = ?
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list session messages: %w", err)
	}
	var msgIDs []string
	for msgRows.Next() {
		var id string
		if err := msgRows.Scan(&id); err != nil {
			msgRows.Close()
			return nil, fmt.Errorf("failed to scan message id: %w", err)
		}
		msgIDs = append(msgIDs, id)
	}
	msgRows.Close()
	if err := msgRows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate message ids: %w", err)
	}
	if len(msgIDs) == 0 {
		return nil, nil
	}

	// 查询所有已审批事实并在应用层过滤
	rows, err := r.db.QueryContext(ctx, `
		SELECT fact_id, subject, predicate, object, confidence, source_msg_ids, status, is_sensitive, scored_at, reviewed_at, created_at
		FROM extracted_facts
		WHERE status = 'approved'
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list facts: %w", err)
	}
	defer rows.Close()

	facts, err := scanFacts(rows)
	if err != nil {
		return nil, err
	}

	// 构建 message_id 集合用于快速查找
	msgIDSet := make(map[string]struct{}, len(msgIDs))
	for _, id := range msgIDs {
		msgIDSet[id] = struct{}{}
	}

	var result []*entity.ExtractedFact
	for _, f := range facts {
		for _, sid := range f.SourceMsgIDs {
			if _, ok := msgIDSet[sid]; ok {
				result = append(result, f)
				break
			}
		}
	}
	return result, nil
}

// CountApprovedFactsNeedingEmbedding 统计需要（重新）生成 embedding 的已审批事实数。
func (r *FactRepoSQLite) CountApprovedFactsNeedingEmbedding(ctx context.Context, targetVersion string) (int64, error) {
	var count int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM extracted_facts f
		LEFT JOIN semantic_embeddings e ON f.fact_id = e.fact_id
		WHERE f.status = 'approved'
		  AND (e.fact_id IS NULL OR e.model_version <> ?)
	`, targetVersion).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count approved facts needing embedding: %w", err)
	}
	return count, nil
}

// ListApprovedFactsNeedingEmbedding 使用稳定 cursor 分页列出需要重新生成 embedding 的已审批事实。
// 按 created_at ASC, fact_id ASC 排序，支持边处理边更新而不丢失候选。
func (r *FactRepoSQLite) ListApprovedFactsNeedingEmbedding(ctx context.Context, targetVersion string, lastCreatedAt time.Time, lastFactID string, limit int) ([]*entity.ExtractedFact, error) {
	if limit <= 0 {
		limit = 500
	}

	args := []any{targetVersion}
	cursorSQL := ""
	if lastFactID != "" {
		cursorSQL = " AND (f.created_at > ? OR (f.created_at = ? AND f.fact_id > ?))"
		args = append(args, lastCreatedAt.UnixMilli(), lastCreatedAt.UnixMilli(), lastFactID)
	}
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT f.fact_id, f.subject, f.predicate, f.object, f.confidence,
		       f.source_msg_ids, f.status, f.is_sensitive, f.scored_at, f.reviewed_at, f.created_at
		FROM extracted_facts f
		LEFT JOIN semantic_embeddings e ON f.fact_id = e.fact_id
		WHERE f.status = 'approved'
		  AND (e.fact_id IS NULL OR e.model_version <> ?)
		%s
		ORDER BY f.created_at ASC, f.fact_id ASC
		LIMIT ?
	`, cursorSQL), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list approved facts needing embedding: %w", err)
	}
	defer rows.Close()

	return scanFacts(rows)
}

// GetStats 获取审核统计。
func (r *FactRepoSQLite) GetStats(ctx context.Context) (total, approved, rejected, pending int64, err error) {
	err = r.db.QueryRowContext(ctx, `SELECT count(*) FROM extracted_facts`).Scan(&total)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to count total facts: %w", err)
	}
	err = r.db.QueryRowContext(ctx, `SELECT count(*) FROM extracted_facts WHERE status = 'approved'`).Scan(&approved)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to count approved facts: %w", err)
	}
	err = r.db.QueryRowContext(ctx, `SELECT count(*) FROM extracted_facts WHERE status = 'rejected'`).Scan(&rejected)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to count rejected facts: %w", err)
	}
	err = r.db.QueryRowContext(ctx, `SELECT count(*) FROM extracted_facts WHERE status = 'pending'`).Scan(&pending)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to count pending facts: %w", err)
	}
	return total, approved, rejected, pending, nil
}

func scanFact(row *sql.Row) (*entity.ExtractedFact, error) {
	var f entity.ExtractedFact
	var sourceIDsJSON string
	var isSensitive int
	var scoredAt, reviewedAt *int64
	var created int64

	if err := row.Scan(&f.FactID, &f.Subject, &f.Predicate, &f.Object, &f.Confidence, &sourceIDsJSON, &f.Status, &isSensitive, &scoredAt, &reviewedAt, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("fact not found: %w", entity.ErrFactNotFound)
		}
		return nil, fmt.Errorf("failed to scan fact: %w", err)
	}

	if err := json.Unmarshal([]byte(sourceIDsJSON), &f.SourceMsgIDs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal source_msg_ids: %w", err)
	}
	f.IsSensitive = isSensitive != 0
	if scoredAt != nil {
		t := time.UnixMilli(*scoredAt).UTC()
		f.ScoredAt = &t
	}
	if reviewedAt != nil {
		t := time.UnixMilli(*reviewedAt).UTC()
		f.ReviewedAt = &t
	}
	f.CreatedAt = time.UnixMilli(created).UTC()
	return &f, nil
}

func scanFacts(rows *sql.Rows) ([]*entity.ExtractedFact, error) {
	var result []*entity.ExtractedFact
	for rows.Next() {
		var f entity.ExtractedFact
		var sourceIDsJSON string
		var isSensitive int
		var scoredAt, reviewedAt *int64
		var created int64

		if err := rows.Scan(&f.FactID, &f.Subject, &f.Predicate, &f.Object, &f.Confidence, &sourceIDsJSON, &f.Status, &isSensitive, &scoredAt, &reviewedAt, &created); err != nil {
			return nil, fmt.Errorf("failed to scan fact: %w", err)
		}
		if err := json.Unmarshal([]byte(sourceIDsJSON), &f.SourceMsgIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal source_msg_ids: %w", err)
		}
		f.IsSensitive = isSensitive != 0
		if scoredAt != nil {
			t := time.UnixMilli(*scoredAt).UTC()
			f.ScoredAt = &t
		}
		if reviewedAt != nil {
			t := time.UnixMilli(*reviewedAt).UTC()
			f.ReviewedAt = &t
		}
		f.CreatedAt = time.UnixMilli(created).UTC()
		result = append(result, &f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate facts: %w", err)
	}
	return result, nil
}

// FactRepoSet 供 Wire 使用的 ProviderSet。
var FactRepoSet = wire.NewSet(
	NewFactRepoSQLite,
)
