package main

import (
	"testing"

	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
)

// TestStrictDeidNeedsConfirm 验证严格级 fail-closed 确认判定的纯逻辑。
func TestStrictDeidNeedsConfirm(t *testing.T) {
	t.Parallel()
	failed := &usecase.PreparedPrompt{DeidFailed: true}
	ok := &usecase.PreparedPrompt{DeidFailed: false}

	// 严格级 + 降级 + 未强制 → 需确认
	assert.True(t, strictDeidNeedsConfirm(models.DesensitizationStrict, failed, false))
	// 严格级 + 降级 + 强制发送 → 不需确认（用户已确认）
	assert.False(t, strictDeidNeedsConfirm(models.DesensitizationStrict, failed, true))
	// 严格级 + 未降级 → 不需确认
	assert.False(t, strictDeidNeedsConfirm(models.DesensitizationStrict, ok, false))
	// 标准级即使降级也不走确认流
	assert.False(t, strictDeidNeedsConfirm(models.DesensitizationStandard, failed, false))
	// off 级不走确认流
	assert.False(t, strictDeidNeedsConfirm(models.DesensitizationOff, failed, false))
	// prepared 为 nil → 不需确认
	assert.False(t, strictDeidNeedsConfirm(models.DesensitizationStrict, nil, false))
}

// TestDeidDegradedPreview 验证降级预览从消息内容构造。
func TestDeidDegradedPreview(t *testing.T) {
	t.Parallel()
	msgs := []models.Message{
		{Role: models.RoleUser, Content: "第一条已脱敏"},
		{Role: models.RoleUser, Content: "第二条已脱敏"},
	}
	preview := deidDegradedPreview(msgs)
	assert.Equal(t, []string{"第一条已脱敏", "第二条已脱敏"}, preview)
	assert.Empty(t, deidDegradedPreview(nil))
}
