// Package service 定义领域服务接口。
// 领域服务封装跨实体的业务逻辑，无状态、无副作用。
package service

import (
	"context"

	"github.com/medmemo/medmemo/internal/domain/entity"
)

// MemoryConsolidator 负责记忆冲突检测与合并决策。
// 禁止静默覆盖——新旧记忆矛盾时必须高亮冲突并请求用户确认。
type MemoryConsolidator interface {
	// DetectConflict 检测新记忆与现有记忆之间是否存在矛盾。
	DetectConflict(ctx context.Context, newMem, existingMem *entity.HealthMemory) (bool, string, error)

	// ProposeMerge 当检测到冲突时，提出合并建议。
	ProposeMerge(ctx context.Context, memories []*entity.HealthMemory) (*entity.HealthMemory, error)
}

// FamilyGraphAnalyzer 负责家族关系图谱的分析服务。
type FamilyGraphAnalyzer interface {
	// DetectCycle 检测家族关系图中是否存在环（无效血缘链）。
	DetectCycle(ctx context.Context, rootID string) (bool, []string, error)

	// AnalyzeDiseaseCluster 分析某疾病在家族中的聚集模式。
	AnalyzeDiseaseCluster(ctx context.Context, diseaseName string) ([]string, error)
}
