// Package repository 实现数据持久化适配器。
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/hzhan516/medmemo/pkg/models"
)

// MessageRepoSQLite 基于 SQLite 的消息仓库实现。
type MessageRepoSQLite struct {
	db *sql.DB
}

// NewMessageRepoSQLite 构造函数。
func NewMessageRepoSQLite(connector database.DBConnector) *MessageRepoSQLite {
	return &MessageRepoSQLite{db: connector.DB()}
}

// Save 保存单条消息。
func (r *MessageRepoSQLite) Save(ctx context.Context, convID models.ConversationID, msg *entity.Message) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO messages (
			id, conversation_id, role, content, tokens,
			prompt_tokens, completion_tokens,
			confidence_score, confidence_level, confidence_json,
			created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		msg.ID, convID, msg.Role, msg.Content,
		msg.PromptTokens+msg.CompletionTokens, // tokens 保持兼容
		msg.PromptTokens, msg.CompletionTokens,
		msg.ConfidenceScore, msg.ConfidenceLevel, msg.ConfidenceJSON,
		msg.Timestamp.UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	return nil
}

// ListByConversation 按会话查询消息，支持 cursor-based 分页。
func (r *MessageRepoSQLite) ListByConversation(ctx context.Context, convID models.ConversationID, cursor string, limit int) ([]*entity.Message, string, error) {
	var args []interface{}
	args = append(args, convID)

	query := `
		SELECT
			id, role, content, created_at,
			prompt_tokens, completion_tokens,
			confidence_score, confidence_level, confidence_json
		FROM messages
		WHERE conversation_id = ? AND deleted_at IS NULL
	`

	if cursor != "" {
		cursorTs, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor: %w", err)
		}
		query += " AND created_at < ?"
		args = append(args, cursorTs)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit+1) // 多取一条用于判断是否有下一页

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list messages: %w", err)
	}
	defer rows.Close()

	var result []*entity.Message
	for rows.Next() {
		var msg entity.Message
		var createdAt int64
		var confidenceScore sql.NullFloat64
		var confidenceLevel sql.NullString
		var confidenceJSON sql.NullString
		if err := rows.Scan(
			&msg.ID, &msg.Role, &msg.Content, &createdAt,
			&msg.PromptTokens, &msg.CompletionTokens,
			&confidenceScore, &confidenceLevel, &confidenceJSON,
		); err != nil {
			return nil, "", fmt.Errorf("failed to scan message: %w", err)
		}
		msg.Timestamp = time.UnixMilli(createdAt)
		if confidenceScore.Valid {
			msg.ConfidenceScore = confidenceScore.Float64
		}
		if confidenceLevel.Valid {
			msg.ConfidenceLevel = confidenceLevel.String
		}
		if confidenceJSON.Valid {
			msg.ConfidenceJSON = confidenceJSON.String
		}
		result = append(result, &msg)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("failed to iterate messages: %w", err)
	}

	var nextCursor string
	if len(result) > limit {
		// 有多余的一条，截断并设置下一页 cursor
		nextCursor = strconv.FormatInt(result[limit-1].Timestamp.UnixMilli(), 10)
		result = result[:limit]
	}

	return result, nextCursor, nil
}

// SoftDelete 软删除消息。
func (r *MessageRepoSQLite) SoftDelete(ctx context.Context, msgID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE messages SET deleted_at = ? WHERE id = ?
	`, time.Now().UnixMilli(), msgID)
	if err != nil {
		return fmt.Errorf("failed to soft delete message: %w", err)
	}
	return nil
}

// Restore 恢复软删除的消息。
func (r *MessageRepoSQLite) Restore(ctx context.Context, msgID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE messages SET deleted_at = NULL WHERE id = ?
	`, msgID)
	if err != nil {
		return fmt.Errorf("failed to restore message: %w", err)
	}
	return nil
}
