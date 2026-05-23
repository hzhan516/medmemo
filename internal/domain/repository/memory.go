// Package repository 定义领域层的仓库接口（Port）。
// 所有持久化操作通过接口抽象，保证 Domain 层不依赖具体存储实现。
package repository

import (
	"context"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
)

// MemoryRepository 定义健康记忆的持久化接口。
type MemoryRepository interface {
	// Save 保存记忆，新旧记忆存在矛盾时禁止静默覆盖，应返回冲突信息。
	Save(ctx context.Context, mem *entity.HealthMemory) error

	// GetByID 按 ID 检索记忆。
	GetByID(ctx context.Context, id models.MemoryID) (*entity.HealthMemory, error)

	// Search 基于关键词检索记忆，返回按时间衰减排序的结果。
	Search(ctx context.Context, query string, limit int) ([]*entity.HealthMemory, error)

	// SemanticSearch 基于向量相似度检索长期记忆（L3）。
	SemanticSearch(ctx context.Context, embedding []float32, topK int) ([]*entity.HealthMemory, error)

	// ListByTier 按层级列出记忆。
	ListByTier(ctx context.Context, tier entity.MemoryTier, limit int) ([]*entity.HealthMemory, error)

	// Delete 删除记忆，不影响历史会话内容展示。
	Delete(ctx context.Context, id models.MemoryID) error

	// ListByTimeRange 按时间范围检索记忆。
	ListByTimeRange(ctx context.Context, start, end time.Time) ([]*entity.HealthMemory, error)
}
