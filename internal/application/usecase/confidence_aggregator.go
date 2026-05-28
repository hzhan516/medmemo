package usecase

import (
	"fmt"
	"strings"

	"github.com/hzhan516/medmemo/internal/domain/entity"
)

// ConfidenceAggregator 置信度聚合计算器，五维度加权输出最终置信度（TASK-064）。
type ConfidenceAggregator struct {
	tagger    *KnowledgeSourceTagger
	evaluator *ReasoningChainEvaluator
}

// NewConfidenceAggregator 创建新的置信度聚合计算器。
func NewConfidenceAggregator() *ConfidenceAggregator {
	return &ConfidenceAggregator{
		tagger:    NewKnowledgeSourceTagger(),
		evaluator: NewReasoningChainEvaluator(),
	}
}

// Calculate 聚合五维度分数，计算最终置信度结果。
// 权重：知识来源 30% / 推理链 25% / 上下文 20% / 历史 15% / 不确定性 10%
func (ca *ConfidenceAggregator) Calculate(
	sources []entity.KnowledgeSource,
	reasoning entity.ReasoningChain,
	contextScore float64,
	historyAccuracy float64,
	uncertainty float64,
) *entity.ConfidenceResult {
	sourceScore := ca.tagger.CalculateSourceScore(sources) * 100 * 0.30
	reasoningScore := ca.evaluator.Evaluate(reasoning) * 0.25
	contextWeighted := contextScore * 0.20
	historyWeighted := historyAccuracy * 100 * 0.15
	uncertaintyWeighted := uncertainty * 0.10

	total := sourceScore + reasoningScore + contextWeighted + historyWeighted + uncertaintyWeighted
	total = clampScore(total)

	level := entity.ScoreToLevel(total)
	missing := reasoning.MissingInfoList

	return &entity.ConfidenceResult{
		OverallScore: total,
		Level:        level,
		Breakdown: map[string]float64{
			"knowledge_source": ca.tagger.CalculateSourceScore(sources) * 100,
			"reasoning":        ca.evaluator.Evaluate(reasoning),
			"context":          contextScore,
			"history":          historyAccuracy * 100,
			"uncertainty":      uncertainty,
		},
		Explanation: generateExplanation(total, missing),
		Suggestion:  level.Suggestion(),
		MissingInfo: missing,
	}
}

// CalculateEmergency 紧急症状兜底：固定返回 A 级（高度确信）。
func (ca *ConfidenceAggregator) CalculateEmergency(
	sources []entity.KnowledgeSource,
	reasoning entity.ReasoningChain,
	contextScore float64,
	historyAccuracy float64,
	uncertainty float64,
) *entity.ConfidenceResult {
	_ = sources
	_ = reasoning
	_ = contextScore
	_ = historyAccuracy
	_ = uncertainty
	return &entity.ConfidenceResult{
		OverallScore: 100.0,
		Level:        entity.ConfidenceLevelA,
		Breakdown: map[string]float64{
			"knowledge_source": 100.0,
			"reasoning":        100.0,
			"context":          100.0,
			"history":          100.0,
			"uncertainty":      100.0,
		},
		Explanation: "该判断基于明确医学指南，属于紧急症状强制提醒，置信度为最高等级。",
		Suggestion:  entity.ConfidenceLevelA.Suggestion(),
		MissingInfo: []string{},
	}
}

// CalculateWithRawScore 直接根据原始总分和缺失信息生成结果。
func (ca *ConfidenceAggregator) CalculateWithRawScore(score float64, missing []string) *entity.ConfidenceResult {
	score = clampScore(score)
	level := entity.ScoreToLevel(score)
	return &entity.ConfidenceResult{
		OverallScore: score,
		Level:        level,
		Breakdown:    map[string]float64{},
		Explanation:  generateExplanation(score, missing),
		Suggestion:   level.Suggestion(),
		MissingInfo:  missing,
	}
}

// clampScore 将分数限制在 0-100 范围内。
func clampScore(s float64) float64 {
	if s < 0 {
		return 0
	}
	if s > 100 {
		return 100
	}
	return s
}

// generateExplanation 根据分数和缺失信息生成解释文本。
func generateExplanation(score float64, missing []string) string {
	if score >= 90 {
		return "该回答基于可靠的医学知识来源，逻辑完整，可作为参考。"
	}
	if score >= 70 {
		return "该回答基于合理推断，存在少量不确定性，建议与医生讨论。"
	}
	if len(missing) > 0 {
		items := strings.Join(missing, "、")
		if score >= 50 {
			return fmt.Sprintf("您未提供%s，补充这些信息可以提高判断准确性。", items)
		}
		return fmt.Sprintf("信息不足：缺少%s，强烈建议补充后再次咨询或尽快就医。", items)
	}
	if score >= 30 {
		return "信息不足或情况复杂，建议尽快就医，不要自行判断。"
	}
	return "AI 无法提供有效建议，请立即就医。"
}
