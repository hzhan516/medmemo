package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/hzhan516/medmemo/internal/domain/entity"
)

func TestConfidenceAggregator_Calculate_FullScore(t *testing.T) {
	agg := NewConfidenceAggregator()
	sources := []entity.KnowledgeSource{
		{Type: entity.SourceMedicalGuideline, Confidence: 0.95, Citation: "指南"},
	}
	reasoning := entity.ReasoningChain{
		HasSymptomAnalysis: true,
		HasDifferentialDx:  true,
		HasRecommendation:  true,
		HasUncertaintyAck:  true,
		HasEmergencyCheck:  true,
		MissingInfoList:    []string{},
	}
	contextScore := 90.0
	historyAccuracy := 0.80
	uncertainty := 90.0

	result := agg.Calculate(sources, reasoning, contextScore, historyAccuracy, uncertainty)

	// source: 0.95*0.30=28.5, reasoning: 100*0.25=25, context: 90*0.20=18, history: 80*0.15=12, uncertainty: 90*0.10=9
	// total = 28.5+25+18+12+9 = 92.5
	assert.InDelta(t, 92.5, result.OverallScore, 0.1)
	assert.Equal(t, entity.ConfidenceLevelA, result.Level)
	assert.Equal(t, "可作为参考", result.Suggestion)
}

func TestConfidenceAggregator_Calculate_MediumScore(t *testing.T) {
	agg := NewConfidenceAggregator()
	sources := []entity.KnowledgeSource{
		{Type: entity.SourceEvidenceDB, Confidence: 0.85, Citation: "PubMed"},
	}
	reasoning := entity.ReasoningChain{
		HasSymptomAnalysis: true,
		HasDifferentialDx:  false,
		HasRecommendation:  true,
		HasUncertaintyAck:  true,
		HasEmergencyCheck:  false,
		MissingInfoList:    []string{"疼痛持续时间"},
	}
	contextScore := 60.0
	historyAccuracy := 0.70
	uncertainty := 50.0

	result := agg.Calculate(sources, reasoning, contextScore, historyAccuracy, uncertainty)

	// reasoning: (20+0+25+15+0-10)=50*0.25=12.5
	// source: 0.85*0.30=25.5, context: 60*0.20=12, history: 70*0.15=10.5, uncertainty: 50*0.10=5
	// total ≈ 25.5+12.5+12+10.5+5 = 65.5
	assert.InDelta(t, 65.5, result.OverallScore, 1.0)
	assert.Equal(t, entity.ConfidenceLevelC, result.Level)
	assert.Equal(t, "仅供参考，强烈建议咨询医生", result.Suggestion)
}

func TestConfidenceAggregator_Calculate_LowScore(t *testing.T) {
	agg := NewConfidenceAggregator()
	sources := []entity.KnowledgeSource{
		{Type: entity.SourceLLMInternal, Confidence: 0.60, Citation: ""},
	}
	reasoning := entity.ReasoningChain{
		HasSymptomAnalysis: false,
		HasDifferentialDx:  false,
		HasRecommendation:  false,
		HasUncertaintyAck:  false,
		HasEmergencyCheck:  false,
		MissingInfoList:    []string{"症状", "持续时间", "既往病史"},
	}
	contextScore := 20.0
	historyAccuracy := 0.50
	uncertainty := 10.0

	result := agg.Calculate(sources, reasoning, contextScore, historyAccuracy, uncertainty)

	// 各项均低分，应得到 D/E 级
	assert.Less(t, result.OverallScore, 30.0)
	assert.Equal(t, entity.ConfidenceLevelE, result.Level)
	assert.Equal(t, "必须就医", result.Suggestion)
}

func TestConfidenceAggregator_Calculate_EmergencyOverride(t *testing.T) {
	agg := NewConfidenceAggregator()
	// 紧急症状兜底：无论各维度如何，固定 A 级
	sources := []entity.KnowledgeSource{
		{Type: entity.SourceLLMInternal, Confidence: 0.60, Citation: ""},
	}
	reasoning := entity.ReasoningChain{
		HasSymptomAnalysis: false,
		HasDifferentialDx:  false,
		HasRecommendation:  false,
		HasUncertaintyAck:  false,
		HasEmergencyCheck:  true,
		MissingInfoList:    []string{},
	}
	contextScore := 10.0
	historyAccuracy := 0.50
	uncertainty := 10.0

	result := agg.CalculateEmergency(sources, reasoning, contextScore, historyAccuracy, uncertainty)

	assert.Equal(t, entity.ConfidenceLevelA, result.Level)
	assert.InDelta(t, 100.0, result.OverallScore, 0.1)
	assert.Contains(t, result.Explanation, "明确医学指南")
}

func TestConfidenceAggregator_Calculate_Boundary_AtoB(t *testing.T) {
	agg := NewConfidenceAggregator()
	// 89.9 分应为 B 级
	result := agg.CalculateWithRawScore(89.9, []string{})
	assert.Equal(t, entity.ConfidenceLevelB, result.Level)
}

func TestConfidenceAggregator_Calculate_Boundary_BtoC(t *testing.T) {
	agg := NewConfidenceAggregator()
	// 69.9 分应为 C 级
	result := agg.CalculateWithRawScore(69.9, []string{})
	assert.Equal(t, entity.ConfidenceLevelC, result.Level)
}

func TestConfidenceAggregator_Calculate_Boundary_CtoD(t *testing.T) {
	agg := NewConfidenceAggregator()
	// 49.9 分应为 D 级
	result := agg.CalculateWithRawScore(49.9, []string{})
	assert.Equal(t, entity.ConfidenceLevelD, result.Level)
}

func TestConfidenceAggregator_Calculate_Boundary_DtoE(t *testing.T) {
	agg := NewConfidenceAggregator()
	// 29.9 分应为 E 级
	result := agg.CalculateWithRawScore(29.9, []string{})
	assert.Equal(t, entity.ConfidenceLevelE, result.Level)
}

func TestConfidenceAggregator_Calculate_Breakdown(t *testing.T) {
	agg := NewConfidenceAggregator()
	sources := []entity.KnowledgeSource{
		{Type: entity.SourceMedicalGuideline, Confidence: 0.95, Citation: "指南"},
	}
	reasoning := entity.ReasoningChain{
		HasSymptomAnalysis: true,
		HasDifferentialDx:  true,
		HasRecommendation:  true,
		HasUncertaintyAck:  true,
		HasEmergencyCheck:  true,
		MissingInfoList:    []string{},
	}

	result := agg.Calculate(sources, reasoning, 80.0, 0.75, 85.0)

	assert.NotNil(t, result.Breakdown)
	assert.Contains(t, result.Breakdown, "knowledge_source")
	assert.Contains(t, result.Breakdown, "reasoning")
	assert.Contains(t, result.Breakdown, "context")
	assert.Contains(t, result.Breakdown, "history")
	assert.Contains(t, result.Breakdown, "uncertainty")
}

func TestConfidenceAggregator_Calculate_MissingInfoExplanation(t *testing.T) {
	agg := NewConfidenceAggregator()
	missing := []string{"疼痛持续时间", "既往病史"}
	result := agg.CalculateWithRawScore(65.0, missing)

	assert.Contains(t, result.Explanation, "疼痛持续时间")
	assert.Contains(t, result.Explanation, "既往病史")
	assert.Equal(t, missing, result.MissingInfo)
}

func TestConfidenceAggregator_Calculate_EmptySources(t *testing.T) {
	agg := NewConfidenceAggregator()
	sources := []entity.KnowledgeSource{}
	reasoning := entity.ReasoningChain{
		HasSymptomAnalysis: true,
		HasDifferentialDx:  true,
		HasRecommendation:  true,
		HasUncertaintyAck:  true,
		HasEmergencyCheck:  true,
		MissingInfoList:    []string{},
	}

	result := agg.Calculate(sources, reasoning, 80.0, 0.75, 85.0)
	// 空来源时默认使用 llm_internal(0.60)
	assert.Less(t, result.OverallScore, 85.0)
	assert.Greater(t, result.OverallScore, 50.0)
}

func TestConfidenceAggregator_Calculate_Latency(t *testing.T) {
	agg := NewConfidenceAggregator()
	sources := []entity.KnowledgeSource{
		{Type: entity.SourceMedicalGuideline, Confidence: 0.95, Citation: "指南"},
	}
	reasoning := entity.ReasoningChain{
		HasSymptomAnalysis: true,
		HasDifferentialDx:  true,
		HasRecommendation:  true,
		HasUncertaintyAck:  true,
		HasEmergencyCheck:  true,
		MissingInfoList:    []string{},
	}

	// 10000 次计算延迟应 < 50ms
	var result *entity.ConfidenceResult
	for i := 0; i < 10000; i++ {
		result = agg.Calculate(sources, reasoning, 80.0, 0.75, 85.0)
	}
	_ = result
	// 实际基准测试在 benchmark 文件中
}
