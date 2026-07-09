package usecase

import (
	"context"
	"fmt"

	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/pkg/models"
)

// ContextLengthResolver 负责解析并校验模型上下文长度。
type ContextLengthResolver struct {
	providers port.ProviderStore
}

// NewContextLengthResolver 创建一个新的上下文长度解析器。
func NewContextLengthResolver(providers port.ProviderStore) *ContextLengthResolver {
	return &ContextLengthResolver{providers: providers}
}

// Resolve 根据 providerID 与 modelID 查询对应模型的最大上下文长度。
// 若未配置或低于最小阈值，则返回默认长度 models.DefaultMaxContextLen。
func (r *ContextLengthResolver) Resolve(ctx context.Context, providerID, modelID string) int {
	if providerID == "" || modelID == "" {
		return models.DefaultMaxContextLen
	}

	provider, err := r.providers.Get(ctx, providerID)
	if err != nil || provider == nil {
		return models.DefaultMaxContextLen
	}

	for _, m := range provider.Models {
		if m.ID == modelID && m.MaxContextLength >= models.MinContextLength {
			return m.MaxContextLength
		}
	}

	return models.DefaultMaxContextLen
}

// Validate 校验给定的上下文长度是否在合法范围内。
func (r *ContextLengthResolver) Validate(v int) error {
	if v < models.MinContextLength || v > models.MaxContextLengthCap {
		return fmt.Errorf("max_context_length must be within [%d, %d], got %d", models.MinContextLength, models.MaxContextLengthCap, v)
	}
	return nil
}
