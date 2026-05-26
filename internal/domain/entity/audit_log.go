package entity

import (
	"fmt"
	"time"
)

// AuditAction 定义审核日志的操作类型。
type AuditAction string

const (
	AuditActionCreate  AuditAction = "CREATE"
	AuditActionApprove AuditAction = "APPROVE"
	AuditActionReject  AuditAction = "REJECT"
	AuditActionDelete  AuditAction = "DELETE"
)

// AuditLogEntry 记录记忆相关操作的审计日志。
type AuditLogEntry struct {
	ID         string
	Action     AuditAction // 操作类型
	TargetType string      // 目标实体类型：fact | memory
	TargetID   string      // 目标实体 ID
	OldValue   string      // 变更前值（JSON，可选）
	NewValue   string      // 变更后值（JSON，可选）
	Actor      string      // 操作者：system | user
	Timestamp  time.Time
}

// NewAuditLogEntry 创建新的审计日志记录。
func NewAuditLogEntry(action AuditAction, targetType, targetID, actor string) *AuditLogEntry {
	now := time.Now().UTC()
	return &AuditLogEntry{
		ID:         fmt.Sprintf("audit_%d", now.UnixNano()),
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Actor:      actor,
		Timestamp:  now,
	}
}
