package policy

import (
	"context"

	"github.com/medmemo/medmemo/pkg/models"
)

// SensitiveDataPolicy 定义敏感数据分级与处理策略。
type SensitiveDataPolicy interface {
	// Classify 对文本进行敏感度分级标记。
	Classify(ctx context.Context, text string) ([]models.SensitiveEntity, error)

	// Redact 根据分级执行替换或删除。
	Redact(ctx context.Context, text string, entities []models.SensitiveEntity) (string, error)
}
