package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// 领域实体补充测试（testcase-reviewer / test-master 阶段）
// =============================================================================

// TestSourceType_BaseConfidence_Unknown 验证未知来源类型降级为 llm_internal。
func TestSourceType_BaseConfidence_Unknown(t *testing.T) {
	unknown := SourceType("unknown_xyz")
	assert.InDelta(t, 0.60, unknown.BaseConfidence(), 0.001)
}

// TestSourceType_BaseConfidence_EmptyString 验证空字符串降级为 llm_internal。
func TestSourceType_BaseConfidence_EmptyString(t *testing.T) {
	empty := SourceType("")
	assert.InDelta(t, 0.60, empty.BaseConfidence(), 0.001)
}

// TestConfidenceLevel_Color_Invalid 验证无效等级的颜色降级。
func TestConfidenceLevel_Color_Invalid(t *testing.T) {
	invalid := ConfidenceLevel("Z")
	assert.Equal(t, "#e74c3c", invalid.Color())
}

// TestConfidenceLevel_Label_Invalid 验证无效等级的标签降级。
func TestConfidenceLevel_Label_Invalid(t *testing.T) {
	invalid := ConfidenceLevel("Z")
	assert.Equal(t, "不确定", invalid.Label())
}

// TestConfidenceLevel_Icon_Invalid 验证无效等级的图标降级。
func TestConfidenceLevel_Icon_Invalid(t *testing.T) {
	invalid := ConfidenceLevel("Z")
	assert.Equal(t, "🚫", invalid.Icon())
}

// TestConfidenceLevel_Suggestion_Invalid 验证无效等级的建议降级。
func TestConfidenceLevel_Suggestion_Invalid(t *testing.T) {
	invalid := ConfidenceLevel("Z")
	assert.Equal(t, "必须就医", invalid.Suggestion())
}

// TestConfidenceLevel_String 验证 String() 方法。
func TestConfidenceLevel_String(t *testing.T) {
	assert.Equal(t, "A", ConfidenceLevelA.String())
	assert.Equal(t, "B", ConfidenceLevelB.String())
	assert.Equal(t, "C", ConfidenceLevelC.String())
	assert.Equal(t, "D", ConfidenceLevelD.String())
	assert.Equal(t, "E", ConfidenceLevelE.String())
}

// TestScoreToLevel_Boundary_At100 验证 100 分为 A 级。
func TestScoreToLevel_Boundary_At100(t *testing.T) {
	assert.Equal(t, ConfidenceLevelA, ScoreToLevel(100.0))
}

// TestScoreToLevel_Boundary_At89_9 验证 89.9 分为 B 级。
func TestScoreToLevel_Boundary_At89_9(t *testing.T) {
	assert.Equal(t, ConfidenceLevelB, ScoreToLevel(89.9))
}

// TestScoreToLevel_Boundary_At69_9 验证 69.9 分为 C 级。
func TestScoreToLevel_Boundary_At69_9(t *testing.T) {
	assert.Equal(t, ConfidenceLevelC, ScoreToLevel(69.9))
}

// TestScoreToLevel_Boundary_At49_9 验证 49.9 分为 D 级。
func TestScoreToLevel_Boundary_At49_9(t *testing.T) {
	assert.Equal(t, ConfidenceLevelD, ScoreToLevel(49.9))
}

// TestScoreToLevel_Boundary_At29_9 验证 29.9 分为 E 级。
func TestScoreToLevel_Boundary_At29_9(t *testing.T) {
	assert.Equal(t, ConfidenceLevelE, ScoreToLevel(29.9))
}

// TestScoreToLevel_Boundary_At0 验证 0 分为 E 级。
func TestScoreToLevel_Boundary_At0(t *testing.T) {
	assert.Equal(t, ConfidenceLevelE, ScoreToLevel(0.0))
}

// TestScoreToLevel_Negative 验证负分为 E 级。
func TestScoreToLevel_Negative(t *testing.T) {
	assert.Equal(t, ConfidenceLevelE, ScoreToLevel(-10.0))
}

// TestKnowledgeSource_EmptyCitation 验证空引用。
func TestKnowledgeSource_EmptyCitation(t *testing.T) {
	ks := KnowledgeSource{
		Type:       SourceLLMInternal,
		Confidence: 0.60,
		Citation:   "",
	}
	assert.Equal(t, "", ks.Citation)
	assert.Equal(t, SourceLLMInternal, ks.Type)
}

// TestReasoningChain_Evaluate_Exact100 验证恰好 100 分。
func TestReasoningChain_Evaluate_Exact100(t *testing.T) {
	rc := ReasoningChain{
		HasSymptomAnalysis: true,
		HasDifferentialDx:  true,
		HasRecommendation:  true,
		HasUncertaintyAck:  true,
		HasEmergencyCheck:  true,
		MissingInfoList:    []string{},
	}
	assert.InDelta(t, 100.0, rc.Evaluate(), 0.001)
}

// TestReasoningChain_Evaluate_MaxMissing 验证最多缺失信息扣分不超过基础分。
func TestReasoningChain_Evaluate_MaxMissing(t *testing.T) {
	rc := ReasoningChain{
		HasSymptomAnalysis: true,
		HasDifferentialDx:  true,
		HasRecommendation:  true,
		HasUncertaintyAck:  true,
		HasEmergencyCheck:  true,
		MissingInfoList:    make([]string, 100), // 100 个缺失项
	}
	// 100 - 100*10 = -900 → max(0, ...) = 0
	assert.InDelta(t, 0.0, rc.Evaluate(), 0.001)
}

// TestAccuracyStats_Accuracy_FullCorrect 验证全正确时 1.0。
func TestAccuracyStats_Accuracy_FullCorrect(t *testing.T) {
	stats := AccuracyStats{
		AnswerType:   AnswerTypeSymptomAnalysis,
		TotalCount:   10,
		CorrectCount: 10,
	}
	assert.InDelta(t, 1.0, stats.Accuracy(), 0.001)
}

// TestAccuracyStats_Accuracy_FullWrong 验证全错误时 0.0。
func TestAccuracyStats_Accuracy_FullWrong(t *testing.T) {
	stats := AccuracyStats{
		AnswerType:   AnswerTypeRecommendation,
		TotalCount:   10,
		CorrectCount: 0,
	}
	assert.InDelta(t, 0.0, stats.Accuracy(), 0.001)
}

// TestAccuracyStats_Accuracy_Half 验证 50% 准确率。
func TestAccuracyStats_Accuracy_Half(t *testing.T) {
	stats := AccuracyStats{
		AnswerType:   AnswerTypeHealthInfo,
		TotalCount:   100,
		CorrectCount: 50,
	}
	assert.InDelta(t, 0.50, stats.Accuracy(), 0.001)
}

// TestAccuracyStats_Accuracy_LargeNumbers 验证大数据量精度。
func TestAccuracyStats_Accuracy_LargeNumbers(t *testing.T) {
	stats := AccuracyStats{
		AnswerType:   AnswerTypeEmergency,
		TotalCount:   1000000,
		CorrectCount: 750000,
	}
	assert.InDelta(t, 0.75, stats.Accuracy(), 0.0001)
}
