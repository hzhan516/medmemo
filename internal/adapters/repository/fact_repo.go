// Package repository 实现数据持久化适配器。
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

// Delete 删除事实（级联删除关联嵌入，由外键约束处理）。
func (r *FactRepoSQLite) Delete(ctx context.Context, factID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM extracted_facts WHERE fact_id = ?`, factID)
	if err != nil {
		return fmt.Errorf("failed to delete fact: %w", err)
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
