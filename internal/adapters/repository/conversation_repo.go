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

// Save 保存或更新会话。
func (r *ConversationRepoSQLite) Save(ctx context.Context, conv *entity.Conversation) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO conversations (id, title, model, created_at, updated_at, archived_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, conv.ID, conv.Title, conv.Model, conv.CreatedAt.UnixMilli(), conv.UpdatedAt.UnixMilli(), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to save conversation: %w", err)
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

// ListRecent 查询最近的会话列表（不包含已软删除的）。
func (r *ConversationRepoSQLite) ListRecent(ctx context.Context, limit int) ([]*entity.Conversation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, model, created_at, updated_at
		FROM conversations
		WHERE deleted_at IS NULL
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
		if err := rows.Scan(&conv.ID, &conv.Title, &conv.Model, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		conv.CreatedAt = time.UnixMilli(createdAt)
		conv.UpdatedAt = time.UnixMilli(updatedAt)
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
