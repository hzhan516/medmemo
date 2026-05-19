package detector

import (
	"context"
	"testing"

	"github.com/medmemo/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRuleDetector_Detect_IDCard 验证身份证号检测。
func TestRuleDetector_Detect_IDCard(t *testing.T) {
	det := NewRuleDetector()
	ctx := context.Background()

	entities, err := det.Detect(ctx, "我的身份证号是 110101199001011234")
	require.NoError(t, err)
	assert.Len(t, entities, 1)
	assert.Equal(t, "身份证号", entities[0].Type)
	assert.Equal(t, models.P3Confidential, entities[0].Level)
}

// TestRuleDetector_Detect_Phone 验证手机号检测。
func TestRuleDetector_Detect_Phone(t *testing.T) {
	det := NewRuleDetector()
	ctx := context.Background()

	entities, err := det.Detect(ctx, "联系我 13800138000")
	require.NoError(t, err)
	assert.Len(t, entities, 1)
	assert.Equal(t, "手机号", entities[0].Type)
}

// TestRuleDetector_Detect_Multiple 验证多类型混合检测。
func TestRuleDetector_Detect_Multiple(t *testing.T) {
	det := NewRuleDetector()
	ctx := context.Background()

	entities, err := det.Detect(ctx, "电话 13800138000，邮箱 test@example.com")
	require.NoError(t, err)
	assert.Len(t, entities, 2)
}

// TestRuleDetector_Detect_EmptyInput 验证空输入不 panic。
func TestRuleDetector_Detect_EmptyInput(t *testing.T) {
	det := NewRuleDetector()
	ctx := context.Background()

	entities, err := det.Detect(ctx, "")
	require.NoError(t, err)
	assert.Empty(t, entities)
}

// TestRuleDetector_Detect_NoSensitive 验证无敏感信息返回空。
func TestRuleDetector_Detect_NoSensitive(t *testing.T) {
	det := NewRuleDetector()
	ctx := context.Background()

	entities, err := det.Detect(ctx, "今天天气不错")
	require.NoError(t, err)
	assert.Empty(t, entities)
}
