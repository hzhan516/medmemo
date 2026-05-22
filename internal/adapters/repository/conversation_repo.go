// Package repository 实现数据持久化适配器。
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/internal/infrastructure/database"
	"github.com/medmemo/medmemo/pkg/models"
)

// ConversationRepoSQLite 基于 SQLite 的会话仓库实现。
type ConversationRepoSQLite struct {
	db *sql.DB
}

// NewConversationRepoSQLite 构造函数。
func NewConversationRepoSQLite(connector database.DBConnector) *ConversationRepoSQLite {
	return &ConversationRepoSQLite{db: connector.DB()}
}

// Save 保存新会话。若会话已存在则报错，避免 INSERT OR REPLACE 覆盖已有字段。
func (r *ConversationRepoSQLite) Save(ctx context.Context, conv *entity.Conversation) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO conversations (id, title, model, created_at, updated_at, archived_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, conv.ID, conv.Title, conv.Model, conv.CreatedAt.UnixMilli(), conv.UpdatedAt.UnixMilli(), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to save conversation: %w", err)
	}
	return nil
}

// UpdateTimestamp 仅更新会话的 updated_at 时间戳。
func (r *ConversationRepoSQLite) UpdateTimestamp(ctx context.Context, id models.ConversationID, updatedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE conversations SET updated_at = ? WHERE id = ?
	`, updatedAt.UnixMilli(), id)
	if err != nil {
		return fmt.Errorf("failed to update conversation timestamp: %w", err)
	}
	return nil
}

// UpdateTitle 仅更新会话的标题。
func (r *ConversationRepoSQLite) UpdateTitle(ctx context.Context, id models.ConversationID, title string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE conversations SET title = ? WHERE id = ?
	`, title, id)
	if err != nil {
		return fmt.Errorf("failed to update conversation title: %w", err)
	}
	return nil
}

// GetByID 按 ID 查询会话（不包含已软删除的）。
func (r *ConversationRepoSQLite) GetByID(ctx context.Context, id models.ConversationID) (*entity.Conversation, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, title, model, created_at, updated_at
		FROM conversations
		WHERE id = ? AND deleted_at IS NULL
	`, id)

	var conv entity.Conversation
	var createdAt, updatedAt int64
	if err := row.Scan(&conv.ID, &conv.Title, &conv.Model, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("conversation %s not found: %w", id, entity.ErrNotFound)
		}
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}
	conv.CreatedAt = time.UnixMilli(createdAt)
	conv.UpdatedAt = time.UnixMilli(updatedAt)
	conv.Messages = make([]entity.Message, 0)
	return &conv, nil
}

// ListRecent 查询最近的会话列表（包含已软删除的，供前端区分主列表与回收站）。
func (r *ConversationRepoSQLite) ListRecent(ctx context.Context, limit int) ([]*entity.Conversation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, model, created_at, updated_at, deleted_at
		FROM conversations
		ORDER BY updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	var result []*entity.Conversation
	for rows.Next() {
		var conv entity.Conversation
		var createdAt, updatedAt int64
		var deletedAt sql.NullInt64
		if err := rows.Scan(&conv.ID, &conv.Title, &conv.Model, &createdAt, &updatedAt, &deletedAt); err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		conv.CreatedAt = time.UnixMilli(createdAt)
		conv.UpdatedAt = time.UnixMilli(updatedAt)
		if deletedAt.Valid {
			t := time.UnixMilli(deletedAt.Int64)
			conv.DeletedAt = &t
		}
		conv.Messages = make([]entity.Message, 0)
		result = append(result, &conv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate conversations: %w", err)
	}
	return result, nil
}

// Delete 软删除会话。
func (r *ConversationRepoSQLite) Delete(ctx context.Context, id models.ConversationID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE conversations SET deleted_at = ? WHERE id = ?
	`, time.Now().UnixMilli(), id)
	if err != nil {
		return fmt.Errorf("failed to soft delete conversation: %w", err)
	}
	return nil
}

// Restore 恢复软删除的会话。
func (r *ConversationRepoSQLite) Restore(ctx context.Context, id models.ConversationID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE conversations SET deleted_at = NULL WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("failed to restore conversation: %w", err)
	}
	return nil
}

// HardDelete 物理删除会话（级联删除关联消息）。
func (r *ConversationRepoSQLite) HardDelete(ctx context.Context, id models.ConversationID) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM conversations WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("failed to hard delete conversation: %w", err)
	}
	return nil
}

// ArchiveOlderThan 将 updated_at 早于 cutoff 且未删除的会话移入回收站。
func (r *ConversationRepoSQLite) ArchiveOlderThan(ctx context.Context, cutoff time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE conversations SET deleted_at = ?
		WHERE deleted_at IS NULL AND updated_at < ?
	`, time.Now().UnixMilli(), cutoff.UnixMilli())
	if err != nil {
		return fmt.Errorf("failed to archive old conversations: %w", err)
	}
	return nil
}

// PermanentlyDeleteOlderThan 物理删除 deleted_at 早于 cutoff 的会话。
func (r *ConversationRepoSQLite) PermanentlyDeleteOlderThan(ctx context.Context, cutoff time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM conversations
		WHERE deleted_at IS NOT NULL AND deleted_at < ?
	`, cutoff.UnixMilli())
	if err != nil {
		return fmt.Errorf("failed to permanently delete old conversations: %w", err)
	}
	return nil
}
