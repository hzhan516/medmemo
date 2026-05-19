package ai

import (
	"os"

	"github.com/google/wire"
	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/pkg/models"
)

// defaultEndpoints 定义各提供商的默认 API 端点。
var defaultEndpoints = map[models.ProviderType]string{
	models.ProviderKimi:        "https://api.moonshot.cn",
	models.ProviderOpenAI:      "https://api.openai.com",
	models.ProviderQwen:        "https://dashscope.aliyuncs.com/compatible-mode/v1",
	models.ProviderSiliconFlow: "https://api.siliconflow.cn",
}

// ProviderFactory 根据配置创建对应的 LLMClient 适配器。
func ProviderFactory(cfg *entity.AppConfig) port.LLMClient {
	// 从环境变量读取 API Key（实际生产环境应通过 secret.Store 获取，此处保留环境变量作为开发调试入口）
	apiKey := os.Getenv("MEDMEMO_API_KEY")

	// 本地模型路由
	if cfg.ProviderType == models.ProviderOllama || cfg.ProviderType == models.ProviderLocal {
		endpoint := cfg.APIEndpoint
		if endpoint == "" {
			endpoint = "http://localhost:11434"
		}
		model := cfg.DefaultModel
		if model == "" {
			model = "llama3.1-8b"
		}
		return NewLocalAdapter(endpoint, model)
	}

	// 云端模型路由：使用 OpenAIAdapter
	baseURL := cfg.APIEndpoint
	if baseURL == "" {
		if u, ok := defaultEndpoints[cfg.ProviderType]; ok {
			baseURL = u
		} else {
			baseURL = defaultEndpoints[models.ProviderKimi]
		}
	}

	model := cfg.DefaultModel
	if model == "" {
		model = "kimi-lite"
	}

	return NewOpenAIAdapter(apiKey, baseURL, model)
}

// ProviderSet 供 Wire 使用的 ProviderSet。
var ProviderSet = wire.NewSet(
	ProviderFactory,
)
