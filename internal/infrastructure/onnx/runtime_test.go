package onnx

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewEngine_NoModel_ReturnsUnavailable 验证无模型时引擎降级为不可用，不 panic。
func TestNewEngine_NoModel_ReturnsUnavailable(t *testing.T) {
	engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
	require.NoError(t, err, "无模型时不应返回错误")
	assert.NotNil(t, engine)
	assert.False(t, engine.IsAvailable(), "模型缺失时引擎应标记为不可用")

	err = engine.Close()
	assert.NoError(t, err)
}

// TestNewEngine_NoLibrary_ReturnsUnavailable 验证无动态库时引擎降级为不可用。
func TestNewEngine_NoLibrary_ReturnsUnavailable(t *testing.T) {
	engine, err := NewEngine(EngineConfig{ResourceDir: "/nonexistent/resources", ModelPath: "resources/models/distilbert-ner"})
	require.NoError(t, err, "无动态库时不应返回错误")
	assert.NotNil(t, engine)
	assert.False(t, engine.IsAvailable(), "动态库缺失时引擎应标记为不可用")

	err = engine.Close()
	assert.NoError(t, err)
}

// TestEngine_Predict_WhenUnavailable_ReturnsError 验证不可用引擎调用 Predict 返回错误。
func TestEngine_Predict_WhenUnavailable_ReturnsError(t *testing.T) {
	engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
	require.NoError(t, err)
	defer engine.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	spans, err := engine.Predict(ctx, "测试文本")
	assert.Error(t, err)
	assert.Nil(t, spans)
	assert.Contains(t, err.Error(), "not available")
}

// TestNormalizeLabel 验证 BIO 标签归一化逻辑。
func TestNormalizeLabel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"B-PER", "PER"},
		{"I-PER", "PER"},
		{"B-ORG", "ORG"},
		{"I-LOC", "LOC"},
		{"PER", "PER"},
		{"O", "O"},
		{"B-DISEASE", "DISEASE"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeLabel(tt.input))
		})
	}
}

// TestPlatformLibPath 验证各平台动态库路径解析。
func TestPlatformLibPath(t *testing.T) {
	path, err := PlatformLibPath("/opt/medmemo")
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "/opt/medmemo/lib/")
}

// TestEntitySpan_Struct 验证 EntitySpan 数据结构。
func TestEntitySpan_Struct(t *testing.T) {
	span := EntitySpan{
		Text:  "张三",
		Label: "PER",
		Start: 0,
		End:   6,
	}
	assert.Equal(t, "张三", span.Text)
	assert.Equal(t, "PER", span.Label)
	assert.Equal(t, 0, span.Start)
	assert.Equal(t, 6, span.End)
}
