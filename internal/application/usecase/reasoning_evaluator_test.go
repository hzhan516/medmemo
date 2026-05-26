package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/hzhan516/medmemo/internal/domain/entity"
)

func TestReasoningChainEvaluator_Evaluate_PerfectAnswer(t *testing.T) {
	evaluator := NewReasoningChainEvaluator()
	chain := entity.ReasoningChain{
		HasSymptomAnalysis: true,
		HasDifferentialDx:  true,
		HasRecommendation:  true,
		HasUncertaintyAck:  true,
		HasEmergencyCheck:  true,
		MissingInfoList:    []string{},
	}
	score := evaluator.Evaluate(chain)
	// 20+25+25+15+15 = 100
	assert.InDelta(t, 100.0, score, 0.001)
}

func TestReasoningChainEvaluator_Evaluate_PartialAnswer(t *testing.T) {
	evaluator := NewReasoningChainEvaluator()
	chain := entity.ReasoningChain{
		HasSymptomAnalysis: true,
		HasDifferentialDx:  false,
		HasRecommendation:  true,
		HasUncertaintyAck:  true,
		HasEmergencyCheck:  false,
		MissingInfoList:    []string{"疼痛持续时间"},
	}
	score := evaluator.Evaluate(chain)
	// 20+0+25+15+0 - 1*10 = 50
	assert.InDelta(t, 50.0, score, 0.001)
}

func TestReasoningChainEvaluator_Evaluate_IncompleteAnswer(t *testing.T) {
	evaluator := NewReasoningChainEvaluator()
	chain := entity.ReasoningChain{
		HasSymptomAnalysis: false,
		HasDifferentialDx:  false,
		HasRecommendation:  false,
		HasUncertaintyAck:  false,
		HasEmergencyCheck:  false,
		MissingInfoList:    []string{"症状持续时间", "既往病史", "用药情况", "过敏史"},
	}
	score := evaluator.Evaluate(chain)
	// 0 - 4*10 = -40 → max(0, ...) = 0
	assert.InDelta(t, 0.0, score, 0.001)
}

func TestReasoningChainEvaluator_Evaluate_ShortAnswer(t *testing.T) {
	evaluator := NewReasoningChainEvaluator()
	chain := entity.ReasoningChain{
		HasSymptomAnalysis: false,
		HasDifferentialDx:  false,
		HasRecommendation:  false,
		HasUncertaintyAck:  false,
		HasEmergencyCheck:  false,
		MissingInfoList:    []string{},
	}
	score := evaluator.Evaluate(chain)
	assert.InDelta(t, 0.0, score, 0.001)
}

func TestReasoningChainEvaluator_Evaluate_ComfortOnly(t *testing.T) {
	evaluator := NewReasoningChainEvaluator()
	chain := entity.ReasoningChain{
		HasSymptomAnalysis: false,
		HasDifferentialDx:  false,
		HasRecommendation:  false,
		HasUncertaintyAck:  true,
		HasEmergencyCheck:  false,
		MissingInfoList:    []string{},
	}
	score := evaluator.Evaluate(chain)
	// 仅不确定性承认 15 分
	assert.InDelta(t, 15.0, score, 0.001)
}

func TestReasoningChainEvaluator_Evaluate_MissingInfoPenalty(t *testing.T) {
	evaluator := NewReasoningChainEvaluator()
	chain := entity.ReasoningChain{
		HasSymptomAnalysis: true,
		HasDifferentialDx:  true,
		HasRecommendation:  true,
		HasUncertaintyAck:  true,
		HasEmergencyCheck:  true,
		MissingInfoList:    []string{"a", "b", "c", "d", "e", "f", "g", "eight", "nine", "ten", "eleven"},
	}
	score := evaluator.Evaluate(chain)
	// 100 - 11*10 = -10 → max(0, ...) = 0
	assert.InDelta(t, 0.0, score, 0.001)
}

func TestReasoningChainEvaluator_DetectMissingInfo(t *testing.T) {
	evaluator := NewReasoningChainEvaluator()
	answer := "根据您的描述，胃痛可能与胃炎有关。"
	missing := evaluator.DetectMissingInfo(answer)

	// 应识别出缺失的关键信息
	assert.NotEmpty(t, missing)
	// 至少包含"疼痛持续时间"
	assert.Contains(t, missing, "疼痛持续时间")
}

func TestReasoningChainEvaluator_DetectMissingInfo_Complete(t *testing.T) {
	evaluator := NewReasoningChainEvaluator()
	answer := "根据您描述的胃痛症状（饭后加重、有反酸、持续3天），这种情况可能与胃酸反流或轻度胃炎有关。建议您考虑就诊消化内科。同时请注意，如果疼痛持续加重或出现呕血、黑便等情况，请立即就医。"
	missing := evaluator.DetectMissingInfo(answer)

	// 完整回答应识别较少缺失信息
	assert.LessOrEqual(t, len(missing), 2)
}

func TestReasoningChainEvaluator_ExtractReasoningChain(t *testing.T) {
	evaluator := NewReasoningChainEvaluator()
	answer := "您描述的症状包括头痛和发热。可能的原因有上呼吸道感染或流感。建议您多休息、多喝水。但请注意，如果症状持续加重，请及时就医。"
	chain := evaluator.ExtractReasoningChain(answer)

	assert.True(t, chain.HasSymptomAnalysis)
	assert.True(t, chain.HasDifferentialDx)
	assert.True(t, chain.HasRecommendation)
	assert.True(t, chain.HasUncertaintyAck)
	assert.False(t, chain.HasEmergencyCheck)
}
