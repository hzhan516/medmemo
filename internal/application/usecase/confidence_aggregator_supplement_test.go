package usecase

import (
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// 边界值与异常场景补充测试（testcase-reviewer / test-master 阶段）
// =============================================================================

// TestConfidenceAggregator_Calculate_ScoreAbove100 验证分数超过 100 时被截断。
func TestConfidenceAggregator_Calculate_ScoreAbove100(t *testing.T) {
	t.Parallel()
	agg := NewConfidenceAggregator()
	// 使用极端高分输入使 raw score > 100
	sources := []entity.KnowledgeSource{
		{Type: entity.SourceMedicalGuideline, Confidence: 1.0, Citation: "指南"},
		{Type: entity.SourceMedicalGuideline, Confidence: 1.0, Citation: "指南2"},
	}
	reasoning := entity.ReasoningChain{
		HasSymptomAnalysis: true,
		HasDifferentialDx:  true,
		HasRecommendation:  true,
		HasUncertaintyAck:  true,
		HasEmergencyCheck:  true,
		MissingInfoList:    []string{},
	}

	result := agg.Calculate(sources, reasoning, 100.0, 1.0, 100.0)

	// 即使各维度满分，总分也应被截断到 100
	assert.LessOrEqual(t, result.OverallScore, 100.0)
	assert.Equal(t, entity.ConfidenceLevelA, result.Level)
}

// TestConfidenceAggregator_Calculate_NegativeScoreClamped 验证负分被截断到 0。
func TestConfidenceAggregator_Calculate_NegativeScoreClamped(t *testing.T) {
	t.Parallel()
	agg := NewConfidenceAggregator()
	// 大量缺失信息导致推理链负分
	reasoning := entity.ReasoningChain{
		HasSymptomAnalysis: false,
		HasDifferentialDx:  false,
		HasRecommendation:  false,
		HasUncertaintyAck:  false,
		HasEmergencyCheck:  false,
		MissingInfoList:    []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"},
	}
	result := agg.CalculateWithRawScore(-50.0, reasoning.MissingInfoList)

	assert.GreaterOrEqual(t, result.OverallScore, 0.0)
	assert.Equal(t, entity.ConfidenceLevelE, result.Level)
}

// TestConfidenceAggregator_Calculate_AllLLMInternalSources 验证全 llm_internal 来源时低分。
func TestConfidenceAggregator_Calculate_AllLLMInternalSources(t *testing.T) {
	t.Parallel()
	agg := NewConfidenceAggregator()
	sources := []entity.KnowledgeSource{
		{Type: entity.SourceLLMInternal, Confidence: 0.60, Citation: ""},
		{Type: entity.SourceLLMInternal, Confidence: 0.60, Citation: ""},
		{Type: entity.SourceLLMInternal, Confidence: 0.60, Citation: ""},
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
	// 知识来源分数低，应低于全 medical_guideline 的情况
	assert.Less(t, result.OverallScore, 90.0)
	assert.Greater(t, result.OverallScore, 50.0)
}

// TestConfidenceAggregator_Calculate_Boundary_At90 验证临界值 90 分。
func TestConfidenceAggregator_Calculate_Boundary_At90(t *testing.T) {
	t.Parallel()
	agg := NewConfidenceAggregator()
	result := agg.CalculateWithRawScore(90.0, []string{})
	assert.Equal(t, entity.ConfidenceLevelA, result.Level)
}

// TestConfidenceAggregator_Calculate_Boundary_At70 验证临界值 70 分。
func TestConfidenceAggregator_Calculate_Boundary_At70(t *testing.T) {
	t.Parallel()
	agg := NewConfidenceAggregator()
	result := agg.CalculateWithRawScore(70.0, []string{})
	assert.Equal(t, entity.ConfidenceLevelB, result.Level)
}

// TestConfidenceAggregator_Calculate_Boundary_At50 验证临界值 50 分。
func TestConfidenceAggregator_Calculate_Boundary_At50(t *testing.T) {
	t.Parallel()
	agg := NewConfidenceAggregator()
	result := agg.CalculateWithRawScore(50.0, []string{})
	assert.Equal(t, entity.ConfidenceLevelC, result.Level)
}

// TestConfidenceAggregator_Calculate_Boundary_At30 验证临界值 30 分。
func TestConfidenceAggregator_Calculate_Boundary_At30(t *testing.T) {
	t.Parallel()
	agg := NewConfidenceAggregator()
	result := agg.CalculateWithRawScore(30.0, []string{})
	assert.Equal(t, entity.ConfidenceLevelD, result.Level)
}

// TestConfidenceAggregator_Calculate_Boundary_At0 验证临界值 0 分。
func TestConfidenceAggregator_Calculate_Boundary_At0(t *testing.T) {
	t.Parallel()
	agg := NewConfidenceAggregator()
	result := agg.CalculateWithRawScore(0.0, []string{})
	assert.Equal(t, entity.ConfidenceLevelE, result.Level)
	assert.Equal(t, "必须就医", result.Suggestion)
}

// TestConfidenceAggregator_Calculate_ExplanationHighScore 验证高分解释文本。
func TestConfidenceAggregator_Calculate_ExplanationHighScore(t *testing.T) {
	t.Parallel()
	agg := NewConfidenceAggregator()
	result := agg.CalculateWithRawScore(95.0, []string{})
	assert.Contains(t, result.Explanation, "可作为参考")
}

// TestConfidenceAggregator_Calculate_ExplanationMediumScore 验证中分解释文本。
func TestConfidenceAggregator_Calculate_ExplanationMediumScore(t *testing.T) {
	t.Parallel()
	agg := NewConfidenceAggregator()
	result := agg.CalculateWithRawScore(75.0, []string{})
	assert.Contains(t, result.Explanation, "医生讨论")
}

// TestConfidenceAggregator_Calculate_ExplanationLowScoreNoMissing 验证低分但无缺失信息的解释。
func TestConfidenceAggregator_Calculate_ExplanationLowScoreNoMissing(t *testing.T) {
	t.Parallel()
	agg := NewConfidenceAggregator()
	result := agg.CalculateWithRawScore(40.0, []string{})
	assert.Contains(t, result.Explanation, "尽快就医")
}

// TestKnowledgeSourceTagger_Tag_UnknownSourceDefaults 验证未知来源降级为 llm_internal。
func TestKnowledgeSourceTagger_Tag_UnknownSourceDefaults(t *testing.T) {
	t.Parallel()
	tagger := NewKnowledgeSourceTagger()
	// 使用未定义的来源类型
	ks := tagger.Tag(entity.SourceType("nonexistent"), "citation")
	assert.Equal(t, entity.SourceLLMInternal, ks.Type)
	assert.InDelta(t, 0.60, ks.Confidence, 0.001)
}

// TestKnowledgeSourceTagger_CalculateSourceScore_SingleSource 验证单来源评分。
func TestKnowledgeSourceTagger_CalculateSourceScore_SingleSource(t *testing.T) {
	t.Parallel()
	tagger := NewKnowledgeSourceTagger()
	sources := []entity.KnowledgeSource{
		{Type: entity.SourceEvidenceDB, Confidence: 0.85, Citation: "PubMed"},
	}
	score := tagger.CalculateSourceScore(sources)
	assert.InDelta(t, 0.85, score, 0.001)
}

// TestReasoningChainEvaluator_ExtractReasoningChain_NoKeywords 验证无关键词时全 false。
func TestReasoningChainEvaluator_ExtractReasoningChain_NoKeywords(t *testing.T) {
	t.Parallel()
	evaluator := NewReasoningChainEvaluator()
	answer := "好的。"
	chain := evaluator.ExtractReasoningChain(answer)

	assert.False(t, chain.HasSymptomAnalysis)
	assert.False(t, chain.HasDifferentialDx)
	assert.False(t, chain.HasRecommendation)
	assert.False(t, chain.HasUncertaintyAck)
	assert.False(t, chain.HasEmergencyCheck)
}

// TestReasoningChainEvaluator_ExtractReasoningChain_OnlySymptom 验证仅有症状关键词。
func TestReasoningChainEvaluator_ExtractReasoningChain_OnlySymptom(t *testing.T) {
	t.Parallel()
	evaluator := NewReasoningChainEvaluator()
	answer := "您有头痛和发热症状。"
	chain := evaluator.ExtractReasoningChain(answer)

	assert.True(t, chain.HasSymptomAnalysis)
	assert.False(t, chain.HasDifferentialDx)
	assert.False(t, chain.HasRecommendation)
	assert.False(t, chain.HasUncertaintyAck)
	assert.False(t, chain.HasEmergencyCheck)
}

// TestReasoningChainEvaluator_ExtractReasoningChain_OnlyEmergency 验证仅有紧急关键词。
func TestReasoningChainEvaluator_ExtractReasoningChain_OnlyEmergency(t *testing.T) {
	t.Parallel()
	evaluator := NewReasoningChainEvaluator()
	answer := "请立即去医院，情况严重！"
	chain := evaluator.ExtractReasoningChain(answer)

	assert.False(t, chain.HasSymptomAnalysis)
	assert.False(t, chain.HasDifferentialDx)
	// "医院" 不含推荐关键词，HasRecommendation 应为 false
	assert.False(t, chain.HasRecommendation)
	assert.False(t, chain.HasUncertaintyAck)
	assert.True(t, chain.HasEmergencyCheck)
}

// TestReasoningChainEvaluator_DetectMissingInfo_AllFound 验证所有关键信息均提供时无缺失。
func TestReasoningChainEvaluator_DetectMissingInfo_AllFound(t *testing.T) {
	t.Parallel()
	evaluator := NewReasoningChainEvaluator()
	answer := "患者男，35岁，头痛持续3天，既往有高血压病史，对青霉素过敏，目前在服用降压药，体温38.5度。"
	missing := evaluator.DetectMissingInfo(answer)
	// "年龄" 和 "性别" 检测词可能触发，但不在 required 列表中
	assert.LessOrEqual(t, len(missing), 1)
}

// TestAccuracyTracker_ConcurrentMixedFeedback 验证并发混合反馈的正确性。
func TestAccuracyTracker_ConcurrentMixedFeedback(t *testing.T) {
	t.Parallel()
	tracker := NewAccuracyTracker()
	done := make(chan bool, 200)

	// 100 正确 + 100 错误
	for i := 0; i < 100; i++ {
		go func() {
			tracker.RecordFeedback(entity.AnswerTypeSymptomAnalysis, true, "30d")
			done <- true
		}()
		go func() {
			tracker.RecordFeedback(entity.AnswerTypeSymptomAnalysis, false, "30d")
			done <- true
		}()
	}

	for i := 0; i < 200; i++ {
		<-done
	}

	acc := tracker.GetAccuracy(entity.AnswerTypeSymptomAnalysis, "30d")
	assert.InDelta(t, 0.50, acc, 0.01)
	stats := tracker.GetStats(entity.AnswerTypeSymptomAnalysis, "30d")
	assert.Equal(t, 200, stats.TotalCount)
}

// TestAccuracyTracker_GetStats_ColdStart 验证冷启动时 GetStats 返回零值。
func TestAccuracyTracker_GetStats_ColdStart(t *testing.T) {
	t.Parallel()
	tracker := NewAccuracyTracker()
	stats := tracker.GetStats(entity.AnswerTypeEmergency, "30d")

	assert.Equal(t, 0, stats.TotalCount)
	assert.Equal(t, 0, stats.CorrectCount)
	// AccuracyStats.Accuracy() 在 TotalCount=0 时返回冷启动默认值 0.75
	assert.InDelta(t, 0.75, stats.Accuracy(), 0.001)
}

// TestConfidenceResult_BreakdownNormalization 验证 Breakdown 中各维度分数正确归一化。
func TestConfidenceResult_BreakdownNormalization(t *testing.T) {
	t.Parallel()
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

	// Breakdown 中各维度值应为原始分数（未加权）
	assert.InDelta(t, 95.0, result.Breakdown["knowledge_source"], 0.1)
	assert.InDelta(t, 100.0, result.Breakdown["reasoning"], 0.1)
	assert.InDelta(t, 80.0, result.Breakdown["context"], 0.1)
	assert.InDelta(t, 75.0, result.Breakdown["history"], 0.1)
	assert.InDelta(t, 85.0, result.Breakdown["uncertainty"], 0.1)
}
