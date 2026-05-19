package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultComplianceChecker_Check 验证默认合规检查器始终放行。
func TestDefaultComplianceChecker_Check(t *testing.T) {
	checker := NewDefaultComplianceChecker()
	ctx := context.Background()

	result, err := checker.Check(ctx, "这是一条测试内容")
	require.NoError(t, err)
	assert.False(t, result.Blocked)
	assert.Equal(t, "L4", result.Level)
	assert.Equal(t, "这是一条测试内容", result.SafeText)
}

// TestDefaultComplianceChecker_Check_EmptyText 验证空文本不报错。
func TestDefaultComplianceChecker_Check_EmptyText(t *testing.T) {
	checker := NewDefaultComplianceChecker()
	ctx := context.Background()

	result, err := checker.Check(ctx, "")
	require.NoError(t, err)
	assert.False(t, result.Blocked)
}
