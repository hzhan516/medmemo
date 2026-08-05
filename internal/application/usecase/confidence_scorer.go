// Package usecase 应用用例层，编排领域对象完成完整业务流程。
package usecase

import "github.com/hzhan516/medmemo/internal/domain/entity"

// ConfidenceScorer 事实提取结果的多维度置信度评分器。
type ConfidenceScorer struct {
	// 评分权重
	SubjectWeight       float64
	PredicateWeight     float64
	ObjectWeight        float64
	LLMConfidenceWeight float64
	MultiSourceWeight   float64

	// 自动审核阈值
	AutoApproveThreshold float64
	AutoRejectThreshold  float64

	// 敏感信息检测器
	sensitiveDetector *SensitiveDetector
}

// NewConfidenceScorer 构造函数，使用默认权重。
func NewConfidenceScorer() *ConfidenceScorer {
	return &ConfidenceScorer{
		SubjectWeight:       0.2,
		PredicateWeight:     0.2,
		ObjectWeight:        0.2,
		LLMConfidenceWeight: 0.3,
		MultiSourceWeight:   0.1,

		AutoApproveThreshold: 0.8,
		AutoRejectThreshold:  0.5,
		sensitiveDetector:    NewSensitiveDetector(),
	}
}

// Score 对事实进行多维度评分，返回 0-1 的置信度分数。
// 评分完成后自动检测并标记事实是否包含敏感信息。
func (s *ConfidenceScorer) Score(f *entity.ExtractedFact) float64 {
	var score float64

	if f.Subject != "" {
		score += s.SubjectWeight
	}
	if f.Predicate != "" {
		score += s.PredicateWeight
	}
	if f.Object != "" {
		score += s.ObjectWeight
	}

	// LLM confidence 作为乘法因子
	score += f.Confidence * s.LLMConfidenceWeight

	// 多源验证加分
	if len(f.SourceMsgIDs) > 1 {
		score += s.MultiSourceWeight
	}

	// 仅命中 PII 时标记敏感，不覆盖已有标记；医学关键词不驱动 IsSensitive（见 M04 TODO#040）
	if s.sensitiveDetector != nil && s.sensitiveDetector.Detect(f) {
		f.IsSensitive = true
	}

	return score
}

// EvaluateStatus 根据评分结果返回建议的审核状态。
func (s *ConfidenceScorer) EvaluateStatus(f *entity.ExtractedFact) entity.FactStatus {
	score := s.Score(f)
	if score >= s.AutoApproveThreshold {
		return entity.FactStatusApproved
	}
	if score < s.AutoRejectThreshold {
		return entity.FactStatusRejected
	}
	return entity.FactStatusPending
}
