package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// SourceType / KnowledgeSource 测试
// =============================================================================

func TestSourceType_ConfidenceValues(t *testing.T) {
	tests := []struct {
		name     string
		source   SourceType
		expected float64
	}{
		{"医学指南", SourceMedicalGuideline, 0.95},
		{"循证数据库", SourceEvidenceDB, 0.85},
		{"医学知识图谱", SourceKnowledgeGraph, 0.75},
		{"LLM内部推理", SourceLLMInternal, 0.60},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.source.BaseConfidence()
			assert.InDelta(t, tt.expected, actual, 0.001)
		})
	}
}

func TestKnowledgeSource_Valid(t *testing.T) {
	ks := KnowledgeSource{
		Type:       SourceMedicalGuideline,
		Confidence: 0.95,
		Citation:   "中华医学会消化指南2023",
	}
	assert.Equal(t, SourceMedicalGuideline, ks.Type)
	assert.InDelta(t, 0.95, ks.Confidence, 0.001)
	assert.Equal(t, "中华医学会消化指南2023", ks.Citation)
}

// =============================================================================
// ReasoningChain / Evaluate 测试
// =============================================================================

func TestReasoningChain_Evaluate_Perfect(t *testing.T) {
	rc := ReasoningChain{
		HasSymptomAnalysis: true,
		HasDifferentialDx:  true,
		HasRecommendation:  true,
		HasUncertaintyAck:  true,
		HasEmergencyCheck:  true,
		MissingInfoList:    []string{},
	}
	score := rc.Evaluate()
	// 20+25+25+15+15 = 100
	assert.InDelta(t, 100.0, score, 0.001)
}

func TestReasoningChain_Evaluate_Partial(t *testing.T) {
	rc := ReasoningChain{
		HasSymptomAnalysis: true,
		HasDifferentialDx:  false,
		HasRecommendation:  true,
		HasUncertaintyAck:  false,
		HasEmergencyCheck:  true,
		MissingInfoList:    []string{"疼痛持续时间", "既往病史"},
	}
	score := rc.Evaluate()
	// 20+0+25+0+15 - 2*10 = 40
	assert.InDelta(t, 40.0, score, 0.001)
}

func TestReasoningChain_Evaluate_Incomplete(t *testing.T) {
	rc := ReasoningChain{
		HasSymptomAnalysis: false,
		HasDifferentialDx:  false,
		HasRecommendation:  false,
		HasUncertaintyAck:  false,
		HasEmergencyCheck:  false,
		MissingInfoList:    []string{"a", "b", "c", "d", "e", "f"},
	}
	score := rc.Evaluate()
	// 0 - 6*10 = -60 → max(0, ...) = 0
	assert.InDelta(t, 0.0, score, 0.001)
}

func TestReasoningChain_Evaluate_ShortAnswer(t *testing.T) {
	rc := ReasoningChain{
		HasSymptomAnalysis: false,
		HasDifferentialDx:  false,
		HasRecommendation:  false,
		HasUncertaintyAck:  false,
		HasEmergencyCheck:  false,
		MissingInfoList:    []string{},
	}
	score := rc.Evaluate()
	assert.InDelta(t, 0.0, score, 0.001)
}

func TestReasoningChain_Evaluate_ComfortOnly(t *testing.T) {
	rc := ReasoningChain{
		HasSymptomAnalysis: false,
		HasDifferentialDx:  false,
		HasRecommendation:  false,
		HasUncertaintyAck:  true,
		HasEmergencyCheck:  false,
		MissingInfoList:    []string{},
	}
	score := rc.Evaluate()
	// 仅不确定性承认 15 分
	assert.InDelta(t, 15.0, score, 0.001)
}

// =============================================================================
// ConfidenceLevel 测试
// =============================================================================

func TestScoreToLevel(t *testing.T) {
	tests := []struct {
		score    float64
		expected ConfidenceLevel
	}{
		{95.0, ConfidenceLevelA},
		{90.0, ConfidenceLevelA},
		{89.9, ConfidenceLevelB},
		{78.0, ConfidenceLevelB},
		{70.0, ConfidenceLevelB},
		{69.9, ConfidenceLevelC},
		{55.0, ConfidenceLevelC},
		{50.0, ConfidenceLevelC},
		{49.9, ConfidenceLevelD},
		{35.0, ConfidenceLevelD},
		{30.0, ConfidenceLevelD},
		{29.9, ConfidenceLevelE},
		{0.0, ConfidenceLevelE},
		{100.0, ConfidenceLevelA},
	}
	for _, tt := range tests {
		t.Run(tt.expected.String(), func(t *testing.T) {
			actual := ScoreToLevel(tt.score)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestConfidenceLevel_ColorAndLabel(t *testing.T) {
	tests := []struct {
		level         ConfidenceLevel
		expectedColor string
		expectedLabel string
		expectedIcon  string
	}{
		{ConfidenceLevelA, "#27ae60", "高度确信", "✅"},
		{ConfidenceLevelB, "#3498db", "较为确信", "👍"},
		{ConfidenceLevelC, "#f39c12", "中等确信", "⚠️"},
		{ConfidenceLevelD, "#e67e22", "低确信", "❗"},
		{ConfidenceLevelE, "#e74c3c", "不确定", "🚫"},
	}
	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			assert.Equal(t, tt.expectedColor, tt.level.Color())
			assert.Equal(t, tt.expectedLabel, tt.level.Label())
			assert.Equal(t, tt.expectedIcon, tt.level.Icon())
		})
	}
}

func TestConfidenceLevel_Suggestion(t *testing.T) {
	tests := []struct {
		level    ConfidenceLevel
		expected string
	}{
		{ConfidenceLevelA, "可作为参考"},
		{ConfidenceLevelB, "建议与医生讨论"},
		{ConfidenceLevelC, "仅供参考，强烈建议咨询医生"},
		{ConfidenceLevelD, "建议尽快就医"},
		{ConfidenceLevelE, "必须就医"},
	}
	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.Suggestion())
		})
	}
}

// =============================================================================
// ConfidenceResult 测试
// =============================================================================

func TestConfidenceResult_Validate(t *testing.T) {
	cr := ConfidenceResult{
		OverallScore: 78.0,
		Level:        ConfidenceLevelB,
		Breakdown: map[string]float64{
			"knowledge_source": 80.0,
			"reasoning":        85.0,
			"context":          70.0,
			"history":          75.0,
			"uncertainty":      90.0,
		},
		Explanation: "您未提供疼痛持续时间和既往病史",
		Suggestion:  "建议与医生讨论",
		MissingInfo: []string{"疼痛持续时间", "既往病史"},
	}
	assert.Equal(t, ConfidenceLevelB, cr.Level)
	assert.Equal(t, "建议与医生讨论", cr.Suggestion)
	assert.Len(t, cr.MissingInfo, 2)
}

// =============================================================================
// AnswerType 测试
// =============================================================================

func TestAnswerType_Valid(t *testing.T) {
	types := []AnswerType{
		AnswerTypeSymptomAnalysis,
		AnswerTypeRecommendation,
		AnswerTypeHealthInfo,
		AnswerTypeEmergency,
	}
	for _, at := range types {
		t.Run(string(at), func(t *testing.T) {
			assert.NotEmpty(t, string(at))
		})
	}
}

// =============================================================================
// AccuracyStats 测试
// =============================================================================

func TestAccuracyStats_Accuracy(t *testing.T) {
	stats := AccuracyStats{
		AnswerType:   AnswerTypeSymptomAnalysis,
		TotalCount:   100,
		CorrectCount: 75,
	}
	assert.InDelta(t, 0.75, stats.Accuracy(), 0.001)
}

func TestAccuracyStats_Accuracy_ZeroTotal(t *testing.T) {
	stats := AccuracyStats{
		AnswerType:   AnswerTypeRecommendation,
		TotalCount:   0,
		CorrectCount: 0,
	}
	// 冷启动默认基准值 0.75
	assert.InDelta(t, 0.75, stats.Accuracy(), 0.001)
}
