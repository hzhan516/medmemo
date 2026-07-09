package pipeline

import (
	"context"
	"fmt"
	"testing"

	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNERDetector 用于测试的 NER 检测器 mock。
type mockNERDetector struct {
	available bool
	entities  []models.SensitiveEntity
	err       error
}

func (m *mockNERDetector) Predict(_ context.Context, _ string) ([]models.SensitiveEntity, error) {
	return m.entities, m.err
}

func (m *mockNERDetector) IsAvailable() bool {
	return m.available
}

// TestDeidentifyPipeline_L1Only 验证只有 L1 时流水线正常工作。
func TestDeidentifyPipeline_L1Only(t *testing.T) {
	t.Parallel()
	p := NewDeidentifyPipeline(NewL1RuleStage())
	ctx := context.Background()

	result, err := p.Execute(ctx, "我的身份证号是110101199001011234", models.DesensitizationStandard)
	require.NoError(t, err)
	assert.Contains(t, result.SafeText, "{{ID_CARD_")
	assert.NotContains(t, result.SafeText, "110101199001011234")
}

// TestL2NERStage_Process_NoNER 验证 NER 不可用时降级透传。
func TestL2NERStage_Process_NoNER(t *testing.T) {
	t.Parallel()
	stage := NewL2NERStage(&mockNERDetector{available: false}, &mockNERDetector{available: false})
	ctx := context.Background()

	input := Input{
		Text:     "张三住北京",
		Metadata: map[string]any{"original_text": "张三住北京"},
	}
	output, err := stage.Process(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "张三住北京", output.Text)
}

// TestL2NERStage_Process_EmptyNER 验证 NER 返回空实体时透传。
func TestL2NERStage_Process_EmptyNER(t *testing.T) {
	t.Parallel()
	stage := NewL2NERStage(&mockNERDetector{available: true, entities: nil}, &mockNERDetector{available: true, entities: nil})
	ctx := context.Background()

	input := Input{
		Text:     "张三住北京",
		Metadata: map[string]any{"original_text": "张三住北京"},
	}
	output, err := stage.Process(ctx, input)
	require.NoError(t, err)
	assert.Equal(t, "张三住北京", output.Text)
}

// TestL2NERStage_Process_WithMockNER 验证 mock NER 结果正确替换。
// 使用 UTF-8 byte offset（中文字符每字符 3 bytes）。
func TestL2NERStage_Process_WithMockNER(t *testing.T) {
	t.Parallel()
	// 原始文本 "张三住北京" 的 byte offset：
	// "张"=[0,3), "三"=[3,6), "住"=[6,9), "北"=[9,12), "京"=[12,15)
	entities := []models.SensitiveEntity{
		{Text: "张三", Type: "姓名", StartPos: 0, EndPos: 6, Score: 0.95},
		{Text: "北京", Type: "地点", StartPos: 9, EndPos: 15, Score: 0.88},
	}
	stage := NewL2NERStage(&mockNERDetector{available: true, entities: entities}, &mockNERDetector{available: true, entities: entities})
	ctx := context.Background()

	input := Input{
		Text:     "张三住北京",
		Metadata: map[string]any{"original_text": "张三住北京"},
	}
	output, err := stage.Process(ctx, input)
	require.NoError(t, err)
	assert.NotContains(t, output.Text, "张三")
	assert.NotContains(t, output.Text, "北京")
	assert.Contains(t, output.Text, "{{per_")
	assert.Contains(t, output.Text, "{{loc_")
	assert.Contains(t, output.Text, "住") // 非实体文本保留
}

// TestL2NERStage_Process_FilterOverlap 验证 NER 与 L1 实体重叠时被过滤。
func TestL2NERStage_Process_FilterOverlap(t *testing.T) {
	t.Parallel()
	// 原始文本："张三电话13800138000"
	// byte offset: "张"=[0,3), "三"=[3,6), "电"=[6,9), "话"=[9,12), "1"=[12,13)...
	// L1 实体：手机号 [12, 23)
	l1Entities := []models.SensitiveEntity{
		{
			Text: "13800138000", Type: "手机号", Level: models.P3Confidential,
			StartPos: 12, EndPos: 23,
			Placeholder: "{{phone_a3f7b2d1}}",
		},
	}
	// mock NER 返回：张三 [0,6) + 13800138000 [12,23)（模拟 NER 对数字的误报）
	nerEntities := []models.SensitiveEntity{
		{Text: "张三", Type: "姓名", StartPos: 0, EndPos: 6, Score: 0.95},
		{Text: "13800138000", Type: "姓名", StartPos: 12, EndPos: 23, Score: 0.82},
	}
	stage := NewL2NERStage(&mockNERDetector{available: true, entities: nerEntities}, &mockNERDetector{available: true, entities: nerEntities})
	ctx := context.Background()

	input := Input{
		Text: "张三电话{{phone_a3f7b2d1}}",
		Metadata: map[string]any{
			"original_text":   "张三电话13800138000",
			"l1_entities":     l1Entities,
			"l1_placeholders": map[string]string{"{{phone_a3f7b2d1}}": "13800138000"},
		},
	}
	output, err := stage.Process(ctx, input)
	require.NoError(t, err)
	// 张三应被替换，手机号不应被重复替换
	assert.NotContains(t, output.Text, "张三")
	assert.Contains(t, output.Text, "{{phone_a3f7b2d1}}") // L1 占位符保留
}

// TestL2NERStage_Process_PositionMapping 验证 L1 占位符长度变化后的偏移映射。
func TestL2NERStage_Process_PositionMapping(t *testing.T) {
	t.Parallel()
	// 原始文本："张三电话13800138000住北京"
	// L1 替换 "13800138000"(11 bytes) → "{{phone_a3f7b2d1}}"(18 bytes)，delta = +7
	// NER："北京" 原始位置 [23, 29)，映射后 [30, 36)
	l1Entities := []models.SensitiveEntity{
		{
			Text: "13800138000", Type: "手机号", Level: models.P3Confidential,
			StartPos: 12, EndPos: 23,
			Placeholder: "{{phone_a3f7b2d1}}",
		},
	}
	nerEntities := []models.SensitiveEntity{
		{Text: "北京", Type: "地点", StartPos: 23, EndPos: 29, Score: 0.90},
	}
	stage := NewL2NERStage(&mockNERDetector{available: true, entities: nerEntities}, &mockNERDetector{available: true, entities: nerEntities})
	ctx := context.Background()

	input := Input{
		Text: "张三电话{{phone_a3f7b2d1}}住北京",
		Metadata: map[string]any{
			"original_text": "张三电话13800138000住北京",
			"l1_entities":   l1Entities,
		},
	}
	output, err := stage.Process(ctx, input)
	require.NoError(t, err)
	// 验证 "北京" 被替换，且 L1 占位符完整保留
	assert.NotContains(t, output.Text, "北京")
	assert.Contains(t, output.Text, "{{phone_a3f7b2d1}}")
	assert.Contains(t, output.Text, "{{loc_")
}

// TestL2NERStage_Process_EndToEnd 验证 L1+L2 端到端协同。
func TestL2NERStage_Process_EndToEnd(t *testing.T) {
	t.Parallel()
	// 原始文本："张三住北京市朝阳区，电话13800138000"
	// L1 已替换手机号
	// L2 应替换人名和地点
	l1Entities := []models.SensitiveEntity{
		{
			Text: "13800138000", Type: "手机号", Level: models.P3Confidential,
			StartPos: 33, EndPos: 44,
			Placeholder: "{{phone_a3f7b2d1}}",
		},
	}
	nerEntities := []models.SensitiveEntity{
		{Text: "张三", Type: "姓名", StartPos: 0, EndPos: 6, Score: 0.95},
		{Text: "北京市朝阳区", Type: "地点", StartPos: 9, EndPos: 27, Score: 0.88},
	}
	stage := NewL2NERStage(&mockNERDetector{available: true, entities: nerEntities}, &mockNERDetector{available: true, entities: nerEntities})
	ctx := context.Background()

	input := Input{
		Text: "张三住北京市朝阳区，电话{{phone_a3f7b2d1}}",
		Metadata: map[string]any{
			"original_text": "张三住北京市朝阳区，电话13800138000",
			"l1_entities":   l1Entities,
		},
	}
	output, err := stage.Process(ctx, input)
	require.NoError(t, err)
	assert.NotContains(t, output.Text, "张三")
	assert.NotContains(t, output.Text, "北京市朝阳区")
	assert.NotContains(t, output.Text, "13800138000")
	assert.Contains(t, output.Text, "{{per_")
	assert.Contains(t, output.Text, "{{loc_")
	assert.Contains(t, output.Text, "{{phone_")
	assert.Contains(t, output.Text, "住")   // 非实体保留
	assert.Contains(t, output.Text, "，电话") // 非实体保留

	// 验证 Metadata 中记录了 l2_entities
	l2Entities, ok := output.Metadata["l2_entities"].([]models.SensitiveEntity)
	require.True(t, ok)
	assert.Len(t, l2Entities, 2)
}

// TestL2NERStage_Process_NERError 验证 NER 推理失败时降级透传。
func TestL2NERStage_Process_NERError(t *testing.T) {
	t.Parallel()
	stage := NewL2NERStage(&mockNERDetector{available: true, err: assert.AnError}, &mockNERDetector{available: true, err: assert.AnError})
	ctx := context.Background()

	input := Input{
		Text:     "张三住北京",
		Metadata: map[string]any{"original_text": "张三住北京"},
	}
	output, err := stage.Process(ctx, input)
	// 降级：出错时透传，不返回 error
	require.NoError(t, err)
	assert.Equal(t, "张三住北京", output.Text)
}

// TestFilterOverlappingEntities 验证重叠过滤逻辑。
func TestFilterOverlappingEntities(t *testing.T) {
	t.Parallel()
	l1 := []models.SensitiveEntity{
		{StartPos: 10, EndPos: 20},
	}
	tests := []struct {
		name     string
		ner      []models.SensitiveEntity
		expected int
	}{
		{
			name:     "完全包含在 L1 内",
			ner:      []models.SensitiveEntity{{StartPos: 12, EndPos: 18}},
			expected: 0,
		},
		{
			name:     "部分重叠",
			ner:      []models.SensitiveEntity{{StartPos: 5, EndPos: 15}},
			expected: 0,
		},
		{
			name:     "无重叠-在 L1 之前",
			ner:      []models.SensitiveEntity{{StartPos: 0, EndPos: 5}},
			expected: 1,
		},
		{
			name:     "无重叠-在 L1 之后",
			ner:      []models.SensitiveEntity{{StartPos: 25, EndPos: 30}},
			expected: 1,
		},
		{
			name:     "边界接触-不重叠",
			ner:      []models.SensitiveEntity{{StartPos: 20, EndPos: 25}},
			expected: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterOverlappingEntities(tt.ner, l1)
			assert.Len(t, result, tt.expected)
		})
	}
}

// TestMapOriginalToDeidPos 验证偏移量映射。
func TestMapOriginalToDeidPos(t *testing.T) {
	t.Parallel()
	// 模拟两个 L1 实体：
	// [5, 10) "12345" → "{{id_abc}}" (长度 10)，delta = +5
	// [20, 25) "67890" → "{{phone_def}}" (长度 13)，delta = +8
	l1 := []models.SensitiveEntity{
		{StartPos: 5, EndPos: 10, Placeholder: "{{id_abc}}"},
		{StartPos: 20, EndPos: 25, Placeholder: "{{phone_def}}"},
	}

	tests := []struct {
		origPos  int
		expected int
	}{
		{origPos: 0, expected: 0},   // 在第一个 L1 之前，无偏移
		{origPos: 5, expected: 5},   // 恰好在第一个 L1 起点，无偏移
		{origPos: 7, expected: 12},  // 在第一个 L1 内部，delta=+5
		{origPos: 15, expected: 20}, // 在两个 L1 之间，delta=+5
		{origPos: 22, expected: 35}, // 在第二个 L1 内部，delta=5+8=+13
		{origPos: 30, expected: 43}, // 在第二个 L1 之后，delta=+13
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("pos_%d", tt.origPos), func(t *testing.T) {
			result := mapOriginalToDeidPos(tt.origPos, l1)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestMapTypeToPlaceholderPrefix 验证占位符前缀映射。
func TestMapTypeToPlaceholderPrefix(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "per", mapTypeToPlaceholderPrefix("姓名"))
	assert.Equal(t, "loc", mapTypeToPlaceholderPrefix("地点"))
	assert.Equal(t, "org", mapTypeToPlaceholderPrefix("机构名"))
	assert.Equal(t, "ent", mapTypeToPlaceholderPrefix("未知"))
}

// TestL2NERStage_SelectsDetectorByLevel 验证按脱敏级别选择对应的 NER 检测器。
func TestL2NERStage_SelectsDetectorByLevel(t *testing.T) {
	t.Parallel()
	// 标准级检测器不返回实体；严格级检测器返回一个实体（模拟低阈值召回）。
	standard := &mockNERDetector{available: true, entities: nil}
	strict := &mockNERDetector{available: true, entities: []models.SensitiveEntity{
		{Text: "张三", Type: "姓名", StartPos: 0, EndPos: 6, Score: 0.6},
	}}
	stage := NewL2NERStage(standard, strict)
	ctx := context.Background()

	mkInput := func(level models.DesensitizationLevel) Input {
		return Input{Text: "张三住北京", Level: level, Metadata: map[string]any{"original_text": "张三住北京"}}
	}

	outStd, err := stage.Process(ctx, mkInput(models.DesensitizationStandard))
	require.NoError(t, err)
	assert.Contains(t, outStd.Text, "张三", "标准级检测器未召回，姓名保留")

	outStrict, err := stage.Process(ctx, mkInput(models.DesensitizationStrict))
	require.NoError(t, err)
	assert.NotContains(t, outStrict.Text, "张三", "严格级检测器召回，姓名被遮蔽")
	assert.Contains(t, outStrict.Text, "{{per_")
}
