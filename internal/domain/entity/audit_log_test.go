package entity

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuditLogEntry_PopulatesFields(t *testing.T) {
	t.Parallel()

	before := time.Now().UTC()
	entry := NewAuditLogEntry(AuditActionApprove, "fact", "fact-123", "user")
	after := time.Now().UTC()

	require.NotNil(t, entry)
	assert.Equal(t, AuditActionApprove, entry.Action)
	assert.Equal(t, "fact", entry.TargetType)
	assert.Equal(t, "fact-123", entry.TargetID)
	assert.Equal(t, "user", entry.Actor)

	// 未显式设置的可选字段应为零值
	assert.Empty(t, entry.OldValue)
	assert.Empty(t, entry.NewValue)

	// 时间戳应为 UTC，并落在调用前后的区间内
	assert.Equal(t, time.UTC, entry.Timestamp.Location())
	assert.False(t, entry.Timestamp.Before(before), "timestamp must not precede the call start")
	assert.False(t, entry.Timestamp.After(after), "timestamp must not follow the call end")
}

func TestNewAuditLogEntry_IDFormat(t *testing.T) {
	t.Parallel()

	entry := NewAuditLogEntry(AuditActionCreate, "memory", "mem-1", "system")

	// ID 采用 "audit_<纳秒时间戳>" 格式
	assert.True(t, strings.HasPrefix(entry.ID, "audit_"), "ID should have audit_ prefix, got %q", entry.ID)
	suffix := strings.TrimPrefix(entry.ID, "audit_")
	assert.NotEmpty(t, suffix)
	for _, r := range suffix {
		assert.True(t, r >= '0' && r <= '9', "ID suffix must be numeric, got %q", entry.ID)
	}
}

func TestNewAuditLogEntry_GeneratesDistinctIDs(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		entry := NewAuditLogEntry(AuditActionDelete, "fact", "fact-x", "system")
		_, dup := seen[entry.ID]
		require.Falsef(t, dup, "duplicate audit ID generated: %q", entry.ID)
		seen[entry.ID] = struct{}{}
	}
}

func TestAuditActionConstants(t *testing.T) {
	t.Parallel()

	// 固定审计操作常量的字符串值，防止意外变更破坏持久化数据兼容性
	assert.Equal(t, AuditAction("CREATE"), AuditActionCreate)
	assert.Equal(t, AuditAction("APPROVE"), AuditActionApprove)
	assert.Equal(t, AuditAction("REJECT"), AuditActionReject)
	assert.Equal(t, AuditAction("DELETE"), AuditActionDelete)
}
