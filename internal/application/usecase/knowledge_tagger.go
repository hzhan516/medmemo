package usecase

import "github.com/hzhan516/medmemo/internal/domain/entity"

// KnowledgeSourceTagger 知识源标记器，为每个知识片段标记来源类型和可信度（TASK-062）。
type KnowledgeSourceTagger struct{}

// NewKnowledgeSourceTagger 创建新的知识源标记器。
func NewKnowledgeSourceTagger() *KnowledgeSourceTagger {
	return &KnowledgeSourceTagger{}
}

// Tag 根据来源类型生成 KnowledgeSource，未知类型降级为 llm_internal。
func (kt *KnowledgeSourceTagger) Tag(sourceType entity.SourceType, citation string) entity.KnowledgeSource {
	baseConf := sourceType.BaseConfidence()
	// 未知类型时 BaseConfidence 已返回 llm_internal 的 0.60
	if baseConf == entity.SourceLLMInternal.BaseConfidence() && sourceType != entity.SourceLLMInternal {
		sourceType = entity.SourceLLMInternal
	}
	return entity.KnowledgeSource{
		Type:       sourceType,
		Confidence: baseConf,
		Citation:   citation,
	}
}

// CalculateSourceScore 计算多来源的平均可信度，空列表时降级为 llm_internal 基准值。
func (kt *KnowledgeSourceTagger) CalculateSourceScore(sources []entity.KnowledgeSource) float64 {
	if len(sources) == 0 {
		return entity.SourceLLMInternal.BaseConfidence()
	}
	var sum float64
	for _, s := range sources {
		sum += s.Confidence
	}
	return sum / float64(len(sources))
}
