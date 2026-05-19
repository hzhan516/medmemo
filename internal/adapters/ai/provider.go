package ai

import (
	"os"

	"github.com/google/wire"
	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/pkg/models"
)

// defaultEndpoints 定义各云端提供商的默认 API 端点。
var defaultEndpoints = map[models.ProviderType]string{
	models.ProviderKimi:        "https://api.moonshot.cn",
	models.ProviderOpenAI:      "https://api.openai.com",
	models.ProviderQwen:        "https://dashscope.aliyuncs.com/compatible-mode/v1",
	models.ProviderSiliconFlow: "https://api.siliconflow.cn",
}

// localEndpoints 定义本地模型的默认端点。
var localEndpoints = map[models.ProviderType]string{
	models.ProviderOllama: "http://localhost:11434",
	models.ProviderLocal:  "http://localhost:8080",
}

// localDefaultModels 定义本地模型的默认模型名。
var localDefaultModels = map[models.ProviderType]string{
	models.ProviderOllama: "llama3.1-8b",
	models.ProviderLocal:  "llama3.1-8b",
}

// ProviderFactory 根据配置创建对应的 LLMClient 适配器。
//
// 路由规则：
//   - ProviderOllama  → LocalAdapter（Ollama 原生 /api/chat API）
//   - ProviderLocal   → OpenAIAdapter（OpenAI 兼容本地端点，如 llama.cpp / text-generation-webui）
//   - 云端 Provider   → OpenAIAdapter（OpenAI-compatible API）
func ProviderFactory(cfg *entity.AppConfig) port.LLMClient {
	// 从环境变量读取 API Key（实际生产环境应通过 secret.Store 获取，此处保留环境变量作为开发调试入口）
	apiKey := os.Getenv("MEDMEMO_API_KEY")

	// Ollama 本地模型路由（原生 Ollama API）
	if cfg.ProviderType == models.ProviderOllama {
		endpoint := cfg.APIEndpoint
		if endpoint == "" {
			endpoint = localEndpoints[models.ProviderOllama]
		}
		model := cfg.DefaultModel
		if model == "" {
			model = localDefaultModels[models.ProviderOllama]
		}
		return NewLocalAdapter(endpoint, model)
	}

	// 通用本地端点路由（OpenAI 兼容 API，如 llama.cpp / text-generation-webui）
	if cfg.ProviderType == models.ProviderLocal {
		endpoint := cfg.APIEndpoint
		if endpoint == "" {
			endpoint = localEndpoints[models.ProviderLocal]
		}
		model := cfg.DefaultModel
		if model == "" {
			model = localDefaultModels[models.ProviderLocal]
		}
		// llama.cpp 等本地端点使用 OpenAI 兼容协议，apiKey 留空
		return NewOpenAIAdapter("", endpoint, model)
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
