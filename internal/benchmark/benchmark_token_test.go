//go:build benchmark

package benchmark

import (
	"testing"

	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/hzhan516/medmemo/internal/domain/entity"
)

// BenchmarkTokenGrowth 验证记忆注入后系统 prompt 的 token 增长 < 20%。
func BenchmarkTokenGrowth(b *testing.B) {
	// 基线系统 prompt（模拟典型值，约 1200 个字符）
	baselinePrompt := `你是一位健康咨询助手。你的任务是帮助用户理解健康问题，提供信息参考，但不进行诊断、不开具处方、不推荐治疗方案。在回答时，请注意以下原则：1. 仅提供一般性健康信息，不输出确诊性结论；2. 鼓励用户咨询专业医生，强调医学建议需由医生面诊后制定；3. 紧急情况下建议拨打120或前往急诊；4. 使用温和、易懂的语言，避免专业术语堆砌；5. 对于用户提到的症状，仅作关联性说明，不给出确定性判断；6. 如涉及药物，仅说明常见用途，不推荐具体剂量；7. 保持中立客观，不制造焦虑；8. 尊重用户隐私，不追问敏感个人信息；9. 当前对话可能包含相关历史记忆，请结合记忆提供更连贯的回答；10. 每次回答前请自我检查，确保不违反以上任何一条原则。请基于以上原则回答用户的问题。`
	baselineRunes := len([]rune(baselinePrompt))

	// 生成 2 条记忆（控制注入 token 量）
	memories := []*entity.HealthMemory{
		{Content: "用户 患有 高血压", Confidence: 0.9},
		{Content: "用户 服用 降压药", Confidence: 0.85},
	}

	injectionText := usecase.FormatMemoriesForInjection(memories)
	injectionRunes := len([]rune(injectionText))

	// 计算增长比例
	growthRate := float64(injectionRunes) / float64(baselineRunes) * 100
	b.ReportMetric(growthRate, "%growth")

	// DoD 要求：增长 < 20%
	if growthRate > 20 {
		b.Fatalf("token growth %.1f%% > 20%%", growthRate)
	}
}
