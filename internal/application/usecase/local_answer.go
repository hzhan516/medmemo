package usecase

import (
	"fmt"

	"github.com/hzhan516/medmemo/internal/domain/entity"
)

// LocalAnswerService 负责将已审批事实格式化为本地模板回答。
// 只复述已记录事实，不输出任何医疗建议。
type LocalAnswerService struct{}

// NewLocalAnswerService 创建新的本地回答服务。
func NewLocalAnswerService() *LocalAnswerService {
	return &LocalAnswerService{}
}

// Format 根据意图类型和事实内容，返回固定的本地模板回答。
// 若 intent 不在支持列表中，返回空字符串。
func (s *LocalAnswerService) Format(intent MemoryIntent, fact *entity.ExtractedFact) string {
	if fact == nil {
		return ""
	}

	switch intent {
	case IntentPersonalWeight:
		return fmt.Sprintf("记录中显示，你当前体重为 %s。", fact.Object)
	case IntentPersonalHeight:
		return fmt.Sprintf("记录中显示，你当前身高为 %s。", fact.Object)
	case IntentPersonalAge:
		return fmt.Sprintf("记录中显示，你当前年龄为 %s。", fact.Object)
	case IntentAllergyHistory:
		return fmt.Sprintf("记录中显示，你的过敏相关信息为：%s。", fact.Object)
	case IntentMedicationHistory:
		return fmt.Sprintf("记录中显示，你的用药相关信息为：%s。", fact.Object)
	default:
		return ""
	}
}
