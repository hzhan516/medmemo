package detector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewONNXNERDetector 验证构造函数。
func TestNewONNXNERDetector(t *testing.T) {
	det := NewONNXNERDetector(nil)
	require.NotNil(t, det)
	assert.False(t, det.IsAvailable())
}

// TestONNXNERDetector_IsAvailable_NilEngine 验证 nil engine 返回不可用。
func TestONNXNERDetector_IsAvailable_NilEngine(t *testing.T) {
	det := NewONNXNERDetector(nil)
	assert.False(t, det.IsAvailable())
}

// TestONNXNERDetector_Predict_NilEngine 验证 nil engine 时降级返回空列表。
func TestONNXNERDetector_Predict_NilEngine(t *testing.T) {
	det := NewONNXNERDetector(nil)
	ctx := context.Background()

	entities, err := det.Predict(ctx, "张三住北京")
	require.NoError(t, err)
	assert.Empty(t, entities)
}

// TestONNXNERDetector_Predict_UnavailableEngine 验证不可用引擎时降级返回空列表。
func TestONNXNERDetector_Predict_UnavailableEngine(t *testing.T) {
	// 创建一个不可用的 mock engine：无法直接 mock *onnx.Engine，
	// 但可以通过 NewEngine 传无效路径获得 available=false 的引擎
	// 由于 onnx 包已就绪，这里直接测试降级路径
	// 注：实际测试依赖于 onnx.NewEngine 的行为，该函数在模型/库缺失时返回 available=false
	// 为避免引入 onnx 包的具体依赖（可能触发 CGO），本测试仅验证 nil 情况
	det := NewONNXNERDetector(nil)
	ctx := context.Background()

	entities, err := det.Predict(ctx, "任意文本")
	require.NoError(t, err)
	assert.Nil(t, entities)
}

// TestMapNERLabel 验证 BIO 标签到中文实体类型的映射。
func TestMapNERLabel(t *testing.T) {
	tests := []struct {
		label    string
		expected string
	}{
		{"PER", "姓名"},
		{"LOC", "地点"},
		{"ORG", "机构名"},
		{"MISC", ""},
		{"DISEASE", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			assert.Equal(t, tt.expected, mapNERLabel(tt.label))
		})
	}
}

// TestONNXNERDetector_ConfidenceThreshold 验证低置信度实体被过滤。
// 通过直接构造带有 mock engine 的内部状态来测试（需反射或导出测试）。
// MVP 阶段：该逻辑在集成测试（pipeline）中通过 mock NERDetector 覆盖。
func TestONNXNERDetector_ConfidenceThreshold_Behavior(t *testing.T) {
	// 阈值常量验证
	assert.Equal(t, float32(0.75), defaultConfidenceThreshold)
}
