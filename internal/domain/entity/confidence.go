package entity

import (
	"math"
	"time"
)

// =============================================================================
// 知识源类型与结构体（TASK-062）
// =============================================================================

// SourceType 表示知识来源的类型。
type SourceType string

const (
	// SourceMedicalGuideline 医学指南，最高可信度。
	SourceMedicalGuideline SourceType = "medical_guideline"
	// SourceEvidenceDB 循证数据库。
	SourceEvidenceDB SourceType = "evidence_db"
	// SourceKnowledgeGraph 医学知识图谱。
	SourceKnowledgeGraph SourceType = "knowledge_graph"
	// SourceLLMInternal LLM 内部推理，最低可信度。
	SourceLLMInternal SourceType = "llm_internal"
)

// baseConfidenceMap 各来源类型的基准可信度。
var baseConfidenceMap = map[SourceType]float64{
	SourceMedicalGuideline: 0.95,
	SourceEvidenceDB:       0.85,
	SourceKnowledgeGraph:   0.75,
	SourceLLMInternal:      0.60,
}

// BaseConfidence 返回该来源类型的基准可信度，未知类型降级为 llm_internal。
func (st SourceType) BaseConfidence() float64 {
	if v, ok := baseConfidenceMap[st]; ok {
		return v
	}
	return baseConfidenceMap[SourceLLMInternal]
}

// KnowledgeSource 表示一条知识片段的来源信息。
type KnowledgeSource struct {
	Type       SourceType // 来源类型
	Confidence float64    // 该来源本身的可信度
	Citation   string     // 来源引用（如"中华医学会消化指南2023"）
}

// =============================================================================
// 推理链评估（TASK-063）
// =============================================================================

// ReasoningChain 表示 AI 回答的推理链完整性。
type ReasoningChain struct {
	HasSymptomAnalysis bool     // 是否分析了症状
	HasDifferentialDx  bool     // 是否有鉴别诊断
	HasRecommendation  bool     // 是否有建议
	HasUncertaintyAck  bool     // 是否承认了不确定性
	HasEmergencyCheck  bool     // 是否检查了紧急症状
	MissingInfoList    []string // 缺失的关键信息列表
}

// Evaluate 评估推理链完整性，返回 0-100 分。
// 五个要素各 15-25 分，每个缺失信息扣 10 分，最低 0 分。
func (rc *ReasoningChain) Evaluate() float64 {
	score := 0.0
	if rc.HasSymptomAnalysis {
		score += 20
	}
	if rc.HasDifferentialDx {
		score += 25
	}
	if rc.HasRecommendation {
		score += 25
	}
	if rc.HasUncertaintyAck {
		score += 15
	}
	if rc.HasEmergencyCheck {
		score += 15
	}
	// 缺失信息扣分
	score -= float64(len(rc.MissingInfoList)) * 10
	return math.Max(0, math.Min(100, score))
}

// =============================================================================
// 置信度等级（TASK-064）
// =============================================================================

// ConfidenceLevel 表示置信度等级 A/B/C/D/E。
type ConfidenceLevel string

const (
	ConfidenceLevelA ConfidenceLevel = "A"
	ConfidenceLevelB ConfidenceLevel = "B"
	ConfidenceLevelC ConfidenceLevel = "C"
	ConfidenceLevelD ConfidenceLevel = "D"
	ConfidenceLevelE ConfidenceLevel = "E"
)

// ScoreToLevel 将 0-100 分转换为置信度等级。
func ScoreToLevel(score float64) ConfidenceLevel {
	switch {
	case score >= 90:
		return ConfidenceLevelA
	case score >= 70:
		return ConfidenceLevelB
	case score >= 50:
		return ConfidenceLevelC
	case score >= 30:
		return ConfidenceLevelD
	default:
		return ConfidenceLevelE
	}
}

// Color 返回该等级对应的颜色代码。
func (cl ConfidenceLevel) Color() string {
	switch cl {
	case ConfidenceLevelA:
		return "#27ae60"
	case ConfidenceLevelB:
		return "#3498db"
	case ConfidenceLevelC:
		return "#f39c12"
	case ConfidenceLevelD:
		return "#e67e22"
	case ConfidenceLevelE:
		return "#e74c3c"
	default:
		return "#e74c3c"
	}
}

// Label 返回该等级的标签文本。
func (cl ConfidenceLevel) Label() string {
	switch cl {
	case ConfidenceLevelA:
		return "高度确信"
	case ConfidenceLevelB:
		return "较为确信"
	case ConfidenceLevelC:
		return "中等确信"
	case ConfidenceLevelD:
		return "低确信"
	case ConfidenceLevelE:
		return "不确定"
	default:
		return "不确定"
	}
}

// Icon 返回该等级的图标。
func (cl ConfidenceLevel) Icon() string {
	switch cl {
	case ConfidenceLevelA:
		return "✅"
	case ConfidenceLevelB:
		return "👍"
	case ConfidenceLevelC:
		return "⚠️"
	case ConfidenceLevelD:
		return "❗"
	case ConfidenceLevelE:
		return "🚫"
	default:
		return "🚫"
	}
}

// Suggestion 返回该等级对应的用户建议。
func (cl ConfidenceLevel) Suggestion() string {
	switch cl {
	case ConfidenceLevelA:
		return "可作为参考"
	case ConfidenceLevelB:
		return "建议与医生讨论"
	case ConfidenceLevelC:
		return "仅供参考，强烈建议咨询医生"
	case ConfidenceLevelD:
		return "建议尽快就医"
	case ConfidenceLevelE:
		return "必须就医"
	default:
		return "必须就医"
	}
}

// String 返回等级字符串表示。
func (cl ConfidenceLevel) String() string {
	return string(cl)
}

// ConfidenceResult 表示置信度聚合计算结果。
type ConfidenceResult struct {
	OverallScore float64             // 综合分数 0-100
	Level        ConfidenceLevel     // 等级 A/B/C/D/E
	Breakdown    map[string]float64  // 五维度细分分数
	Explanation  string              // 解释文本
	Suggestion   string              // 用户建议
	MissingInfo  []string            // 缺失信息列表
	Citations    []KnowledgeCitation // 知识库引用条目（v1.1.9 临时存放在 confidence_json）
}

// =============================================================================
// 回答类型与历史准确率（TASK-068）
// =============================================================================

// AnswerType 表示回答的类型，用于历史准确率分类统计。
type AnswerType string

const (
	AnswerTypeSymptomAnalysis AnswerType = "symptom_analysis"
	AnswerTypeRecommendation  AnswerType = "recommendation"
	AnswerTypeHealthInfo      AnswerType = "health_info"
	AnswerTypeEmergency       AnswerType = "emergency"
)

// AccuracyStats 表示某类型回答的历史准确率统计。
type AccuracyStats struct {
	AnswerType   AnswerType
	TotalCount   int
	CorrectCount int
	WindowStart  time.Time
	UpdatedAt    time.Time
}

// Accuracy 计算准确率，无数据时返回冷启动默认值 0.75。
func (as *AccuracyStats) Accuracy() float64 {
	if as.TotalCount == 0 {
		return 0.75 // 冷启动默认基准值
	}
	return float64(as.CorrectCount) / float64(as.TotalCount)
}
