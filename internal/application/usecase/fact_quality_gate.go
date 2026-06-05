package usecase

import (
	"strings"

	"github.com/hzhan516/medmemo/internal/domain/entity"
)

// FactQualityGate 对 LLM 返回的事实做确定性后过滤。
// 不依赖外部模型，仅基于规则拒绝常见噪音模式，
// 放在 JSON 解析之后、保存之前，作为第二道防线。
type FactQualityGate struct{}

// ApplyFactQualityGate 对事实列表执行质量门禁，返回通过过滤的事实。
func ApplyFactQualityGate(facts []*entity.ExtractedFact) []*entity.ExtractedFact {
	if len(facts) == 0 {
		return facts
	}
	var passed []*entity.ExtractedFact
	for _, f := range facts {
		if isQualityFact(f) {
			passed = append(passed, f)
		}
	}
	return passed
}

// isQualityFact 判断单条事实是否通过质量门禁。
func isQualityFact(f *entity.ExtractedFact) bool {
	sub := strings.ToLower(strings.TrimSpace(f.Subject))
	pred := strings.ToLower(strings.TrimSpace(f.Predicate))
	obj := strings.ToLower(strings.TrimSpace(f.Object))

	// 1. 拒绝 subject 是 AI / 助手 / 模型 / 系统
	aiSubjects := []string{"ai", "助手", "模型", "系统", "assistant", "model", "system"}
	for _, s := range aiSubjects {
		if strings.Contains(sub, s) {
			return false
		}
	}

	// 2. 拒绝 predicate 表示询问 / 咨询 / 想知道 / 了解
	questionPreds := []string{"询问", "咨询", "想知道", "了解", "问", "如何", "怎么"}
	for _, p := range questionPreds {
		if strings.Contains(pred, p) {
			return false
		}
	}

	// 3. 拒绝 predicate 表示建议 / 需要 / 应该 / 可以使用，且 object 是工具或操作
	advicePreds := []string{"建议", "需要", "应该", "可以使用", "推荐", "最好"}
	isAdvicePred := false
	for _, p := range advicePreds {
		if strings.Contains(pred, p) {
			isAdvicePred = true
			break
		}
	}
	if isAdvicePred {
		toolObjects := []string{"秤", "体重秤", "血压计", "血糖仪", "体温计", "计算器", "检查", "化验", "测量"}
		for _, t := range toolObjects {
			if strings.Contains(obj, t) {
				return false
			}
		}
	}

	// 4. 拒绝 predicate/object 表示无法告知 / 不知道 / 不能判断
	uncertaintyPatterns := []string{"无法", "不能", "不知道", "无法告知", "无法判断", "没有权限"}
	for _, p := range uncertaintyPatterns {
		if strings.Contains(pred, p) || strings.Contains(obj, p) {
			return false
		}
	}

	// 5. 个人属性必须有具体值：如果提到属性词但 object 不含数字，拒绝
	personalAttrs := []string{"体重", "身高", "年龄", "bmi", "血压", "血糖", "体温", "心率", "胆固醇"}
	for _, attr := range personalAttrs {
		if strings.Contains(pred, attr) || strings.Contains(obj, attr) {
			if !hasNumber(obj) {
				return false
			}
		}
	}

	return true
}

// hasNumber 检查字符串中是否包含阿拉伯数字。
func hasNumber(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}
