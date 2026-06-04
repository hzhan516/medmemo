package repository

import (
	"context"

	"github.com/hzhan516/medmemo/internal/domain/entity"
)

// AuditLogRepository 定义审计日志的持久化接口。
type AuditLogRepository interface {
	// Save 保存一条审计日志记录。
	Save(ctx context.Context, entry *entity.AuditLogEntry) error
	// ListByTarget 按目标实体查询相关审计日志。
	ListByTarget(ctx context.Context, targetType, targetID string, limit int) ([]*entity.AuditLogEntry, error)
}
