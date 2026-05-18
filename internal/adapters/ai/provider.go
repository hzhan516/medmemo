package ai

import (
	"github.com/google/wire"
	"github.com/medmemo/medmemo/internal/application/port"
)

// ProviderFactory 根据配置创建对应的 LLMClient 适配器。
func ProviderFactory() port.LLMClient {
	// TODO(作者): 基于配置路由到具体适配器 [Issue#013]
	return nil
}

// AIAdapterSet 供 Wire 使用的 ProviderSet。
var AIAdapterSet = wire.NewSet(
	NewOpenAIAdapter,
	NewLocalAdapter,
	ProviderFactory,
)
