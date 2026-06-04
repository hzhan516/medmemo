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

// DialogueRepoSQLite 基于 SQLite 的原始对话仓库实现。
type DialogueRepoSQLite struct {
	db *sql.DB
}

// NewDialogueRepoSQLite 构造函数。
func NewDialogueRepoSQLite(connector database.DBConnector) *DialogueRepoSQLite {
	return &DialogueRepoSQLite{db: connector.DB()}
}

// Insert 保存单条原始对话记录。
func (r *DialogueRepoSQLite) Insert(ctx context.Context, d *entity.RawDialogue) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO raw_dialogues (message_id, session_id, role, content, model_name, timestamp, extraction_status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, d.MessageID, d.SessionID, d.Role, d.Content, d.ModelName, d.Timestamp.UnixMilli(), d.ExtractionStatus, d.CreatedAt.UnixMilli())
	if err != nil {
		return fmt.Errorf("failed to insert raw dialogue: %w", err)
	}
	return nil
}

// InsertBatch 批量保存原始对话记录（事务内）。
func (r *DialogueRepoSQLite) InsertBatch(ctx context.Context, dialogues []*entity.RawDialogue) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin batch insert transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO raw_dialogues (message_id, session_id, role, content, model_name, timestamp, extraction_status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare batch insert statement: %w", err)
	}
	defer stmt.Close()

	for _, d := range dialogues {
		if _, err := stmt.ExecContext(ctx, d.MessageID, d.SessionID, d.Role, d.Content, d.ModelName, d.Timestamp.UnixMilli(), d.ExtractionStatus, d.CreatedAt.UnixMilli()); err != nil {
			return fmt.Errorf("failed to insert dialogue %s: %w", d.MessageID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch insert: %w", err)
	}
	return nil
}

// GetBySession 按会话分页查询。
func (r *DialogueRepoSQLite) GetBySession(ctx context.Context, sessionID string, offset, limit int) ([]*entity.RawDialogue, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT message_id, session_id, role, content, model_name, timestamp, extraction_status, created_at
		FROM raw_dialogues
		WHERE session_id = ?
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`, sessionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query dialogues by session: %w", err)
	}
	defer rows.Close()
	return scanDialogues(rows)
}

// GetRecent 获取会话最近 N 分钟内的消息。
func (r *DialogueRepoSQLite) GetRecent(ctx context.Context, sessionID string, minutes int) ([]*entity.RawDialogue, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(minutes) * time.Minute).UnixMilli()
	rows, err := r.db.QueryContext(ctx, `
		SELECT message_id, session_id, role, content, model_name, timestamp, extraction_status, created_at
		FROM raw_dialogues
		WHERE session_id = ? AND timestamp >= ?
		ORDER BY timestamp DESC
	`, sessionID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent dialogues: %w", err)
	}
	defer rows.Close()
	return scanDialogues(rows)
}

// GetUnprocessed 获取尚未进行事实提取的消息。
func (r *DialogueRepoSQLite) GetUnprocessed(ctx context.Context, limit int) ([]*entity.RawDialogue, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT message_id, session_id, role, content, model_name, timestamp, extraction_status, created_at
		FROM raw_dialogues
		WHERE extraction_status = 'unprocessed'
		ORDER BY timestamp ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query unprocessed dialogues: %w", err)
	}
	defer rows.Close()
	return scanDialogues(rows)
}

// MarkProcessing 标记为处理中。
func (r *DialogueRepoSQLite) MarkProcessing(ctx context.Context, messageID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE raw_dialogues SET extraction_status = 'processing' WHERE message_id = ?
	`, messageID)
	if err != nil {
		return fmt.Errorf("failed to mark dialogue processing: %w", err)
	}
	return nil
}

// MarkProcessed 标记为已处理。
func (r *DialogueRepoSQLite) MarkProcessed(ctx context.Context, messageID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE raw_dialogues SET extraction_status = 'processed' WHERE message_id = ?
	`, messageID)
	if err != nil {
		return fmt.Errorf("failed to mark dialogue processed: %w", err)
	}
	return nil
}

// MarkFailed 标记为处理失败。
func (r *DialogueRepoSQLite) MarkFailed(ctx context.Context, messageID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE raw_dialogues SET extraction_status = 'failed' WHERE message_id = ?
	`, messageID)
	if err != nil {
		return fmt.Errorf("failed to mark dialogue failed: %w", err)
	}
	return nil
}

func scanDialogues(rows *sql.Rows) ([]*entity.RawDialogue, error) {
	var result []*entity.RawDialogue
	for rows.Next() {
		var d entity.RawDialogue
		var ts, created int64
		if err := rows.Scan(&d.MessageID, &d.SessionID, &d.Role, &d.Content, &d.ModelName, &ts, &d.ExtractionStatus, &created); err != nil {
			return nil, fmt.Errorf("failed to scan dialogue: %w", err)
		}
		d.Timestamp = time.UnixMilli(ts).UTC()
		d.CreatedAt = time.UnixMilli(created).UTC()
		result = append(result, &d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate dialogues: %w", err)
	}
	return result, nil
}

// DialogueRepoSet 供 Wire 使用的 ProviderSet。
var DialogueRepoSet = wire.NewSet(
	NewDialogueRepoSQLite,
)
