package detector

import (
	"context"
	"testing"

	"github.com/hzhan516/medmemo/internal/infrastructure/onnx"
	"github.com/hzhan516/medmemo/pkg/models"
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

// TestStrictConfidenceThreshold_Constant 验证严格级阈值常量为 0.5。
func TestStrictConfidenceThreshold_Constant(t *testing.T) {
	assert.Equal(t, float32(0.5), strictConfidenceThreshold)
	assert.Less(t, strictConfidenceThreshold, defaultConfidenceThreshold, "严格级阈值应低于标准级以提升召回")
}

// TestNewStrictONNXNERDetector 验证严格级检测器使用更低阈值且复用引擎。
func TestNewStrictONNXNERDetector(t *testing.T) {
	det := NewStrictONNXNERDetector(nil)
	require.NotNil(t, det)
	require.NotNil(t, det.ONNXNERDetector)
	assert.Equal(t, strictConfidenceThreshold, det.threshold)
	assert.False(t, det.IsAvailable())
}

// TestFilterSpansByThreshold 验证按阈值过滤与类型映射的纯函数逻辑。
func TestFilterSpansByThreshold(t *testing.T) {
	spans := []onnx.EntitySpan{
		{Text: "张三", Label: "PER", Start: 0, End: 6, Score: 0.60},
		{Text: "北京", Label: "LOC", Start: 9, End: 15, Score: 0.90},
		{Text: "某某", Label: "MISC", Start: 20, End: 26, Score: 0.99}, // 无法映射类型，丢弃
	}

	// 标准阈值 0.75：仅保留北京（0.90），张三(0.60)被过滤，MISC 丢弃。
	std := filterSpansByThreshold(spans, defaultConfidenceThreshold)
	require.Len(t, std, 1)
	assert.Equal(t, "北京", std[0].Text)

	// 严格阈值 0.5：保留张三(0.60)与北京(0.90)，MISC 仍因类型丢弃。
	strict := filterSpansByThreshold(spans, strictConfidenceThreshold)
	require.Len(t, strict, 2)
	texts := []string{strict[0].Text, strict[1].Text}
	assert.Contains(t, texts, "张三")
	assert.Contains(t, texts, "北京")
	for _, e := range strict {
		assert.Equal(t, models.P3Confidential, e.Level)
	}
}
