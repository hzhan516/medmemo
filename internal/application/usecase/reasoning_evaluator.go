package usecase

import (
	"strings"

	"github.com/hzhan516/medmemo/internal/domain/entity"
)

// ReasoningChainEvaluator 推理链评估器，分析 AI 回答的逻辑完整性（TASK-063）。
type ReasoningChainEvaluator struct{}

// NewReasoningChainEvaluator 创建新的推理链评估器。
func NewReasoningChainEvaluator() *ReasoningChainEvaluator {
	return &ReasoningChainEvaluator{}
}

// Evaluate 评估推理链完整性，返回 0-100 分。
func (re *ReasoningChainEvaluator) Evaluate(chain entity.ReasoningChain) float64 {
	return chain.Evaluate()
}

// DetectMissingInfo 检测回答中缺失的关键信息列表。
// 基于关键词匹配判断用户是否提供了某些关键信息。
func (re *ReasoningChainEvaluator) DetectMissingInfo(answer string) []string {
	var missing []string
	lower := strings.ToLower(answer)

	// 关键信息检测清单
	checks := []struct {
		keyword string
		label   string
	}{
		{"持续", "疼痛持续时间"},
		{"时间", "疼痛持续时间"},
		{"病史", "既往病史"},
		{"以前", "既往病史"},
		{"过敏", "过敏史"},
		{"药物", "用药情况"},
		{"服用", "用药情况"},
		{"年龄", "年龄"},
		{"岁", "年龄"},
		{"性别", "性别"},
		{"男", "性别"},
		{"女", "性别"},
		{"体重", "体重"},
		{"发热", "是否发热"},
		{"烧", "是否发热"},
		{"体温", "是否发热"},
	}

	// 收集已提及的信息类别（去重）
	found := make(map[string]bool)
	for _, c := range checks {
		if strings.Contains(lower, c.keyword) {
			found[c.label] = true
		}
	}

	// 定义应检测的关键信息集合
	required := []string{"疼痛持续时间", "既往病史", "过敏史", "用药情况", "是否发热"}
	for _, r := range required {
		if !found[r] {
			missing = append(missing, r)
		}
	}

	return missing
}

// ExtractReasoningChain 从回答文本中提取推理链结构。
// 基于关键词启发式分析，判断回答是否包含五个关键要素。
func (re *ReasoningChainEvaluator) ExtractReasoningChain(answer string) entity.ReasoningChain {
	lower := strings.ToLower(answer)
	chain := entity.ReasoningChain{}

	// 症状分析：包含症状关键词
	symptomKeywords := []string{"症状", "表现", "疼痛", "不适", "恶心", "头晕", "头痛", "发热"}
	for _, kw := range symptomKeywords {
		if strings.Contains(lower, kw) {
			chain.HasSymptomAnalysis = true
			break
		}
	}

	// 鉴别诊断：包含"可能"+多个疾病名称，或"或"、"鉴别"
	if strings.Contains(lower, "可能") && (strings.Contains(lower, "或") || strings.Contains(lower, "鉴别")) {
		chain.HasDifferentialDx = true
	}

	// 建议：包含"建议"、"推荐"、"可以"、"考虑"
	recommendKeywords := []string{"建议", "推荐", "可以", "考虑", "就诊", "就医"}
	for _, kw := range recommendKeywords {
		if strings.Contains(lower, kw) {
			chain.HasRecommendation = true
			break
		}
	}

	// 不确定性承认：包含"可能"、"不确定"、"仅供参考"、"建议确认"
	uncertaintyKeywords := []string{"可能", "不确定", "仅供参考", "建议确认", "请遵医嘱", "医生"}
	for _, kw := range uncertaintyKeywords {
		if strings.Contains(lower, kw) {
			chain.HasUncertaintyAck = true
			break
		}
	}

	// 紧急检查：包含"立即"、"尽快"、"严重"、"危险"
	emergencyKeywords := []string{"立即", "尽快", "严重", "危险", "紧急", "马上"}
	for _, kw := range emergencyKeywords {
		if strings.Contains(lower, kw) {
			chain.HasEmergencyCheck = true
			break
		}
	}

	chain.MissingInfoList = re.DetectMissingInfo(answer)

	return chain
}
