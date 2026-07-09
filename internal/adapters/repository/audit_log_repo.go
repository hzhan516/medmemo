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

// AuditLogRepoSQLite 基于 SQLite 的审计日志仓库实现。
type AuditLogRepoSQLite struct {
	db *sql.DB
}

// NewAuditLogRepoSQLite 构造函数。
func NewAuditLogRepoSQLite(connector database.DBConnector) *AuditLogRepoSQLite {
	return &AuditLogRepoSQLite{db: connector.DB()}
}

// Save 保存审计日志记录。
func (r *AuditLogRepoSQLite) Save(ctx context.Context, entry *entity.AuditLogEntry) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_logs (id, action, target_type, target_id, old_value, new_value, actor, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.ID, entry.Action, entry.TargetType, entry.TargetID, entry.OldValue, entry.NewValue, entry.Actor, entry.Timestamp.UnixMilli())
	if err != nil {
		return fmt.Errorf("failed to save audit log: %w", err)
	}
	return nil
}

// ListByTarget 按目标实体查询审计日志。
func (r *AuditLogRepoSQLite) ListByTarget(ctx context.Context, targetType, targetID string, limit int) ([]*entity.AuditLogEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, action, target_type, target_id, old_value, new_value, actor, timestamp
		FROM audit_logs
		WHERE target_type = ? AND target_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, targetType, targetID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var result []*entity.AuditLogEntry
	for rows.Next() {
		var e entity.AuditLogEntry
		var ts int64
		if err := rows.Scan(&e.ID, &e.Action, &e.TargetType, &e.TargetID, &e.OldValue, &e.NewValue, &e.Actor, &ts); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		e.Timestamp = time.UnixMilli(ts).UTC()
		result = append(result, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate audit logs: %w", err)
	}
	return result, nil
}

// AuditLogRepoSet 供 Wire 使用的 ProviderSet。
var AuditLogRepoSet = wire.NewSet(
	NewAuditLogRepoSQLite,
)
