package detector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRuleDetector_Detect 验证规则检测器占位实现返回空结果。
func TestRuleDetector_Detect(t *testing.T) {
	det := NewRuleDetector()
	ctx := context.Background()

	entities, err := det.Detect(ctx, "我的身份证号是 110101199001011234")
	require.NoError(t, err)
	assert.Empty(t, entities)
}

// TestRuleDetector_Detect_EmptyInput 验证空输入不 panic。
func TestRuleDetector_Detect_EmptyInput(t *testing.T) {
	det := NewRuleDetector()
	ctx := context.Background()

	entities, err := det.Detect(ctx, "")
	require.NoError(t, err)
	assert.Empty(t, entities)
}
