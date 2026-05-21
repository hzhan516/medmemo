package ai

import (
	"fmt"
	"strings"

	"github.com/google/wire"
	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/pkg/models"
)

// defaultEndpoints 各云端提供商默认 API 端点。
var defaultEndpoints = map[models.ProviderType]string{
	models.ProviderKimi:        "https://api.moonshot.cn",
	models.ProviderOpenAI:      "https://api.openai.com",
	models.ProviderQwen:        "https://dashscope.aliyuncs.com/compatible-mode/v1",
	models.ProviderSiliconFlow: "https://api.siliconflow.cn",
	models.ProviderMicrosoft:   "https://models.inference.ai.azure.com",
	models.ProviderGitHub:      "https://models.inference.ai.azure.com",
	models.ProviderClaude:      "https://api.anthropic.com",
	models.ProviderGrok:        "https://api.x.ai",
	models.ProviderDoubao:      "https://ark.cn-beijing.volces.com/api/v3",
	models.ProviderGLM:         "https://open.bigmodel.cn/api/paas/v4",
	models.ProviderDeepSeek:    "https://api.deepseek.com",
	models.ProviderMiniMax:     "https://api.minimax.chat/v1",
	models.ProviderXiaomi:      "https://api.mi.ai",
	models.ProviderHunyuan:     "https://hunyuan.tencentcloudapi.com",
}

// localEndpoints 本地模型默认端点。
var localEndpoints = map[models.ProviderType]string{
	models.ProviderOllama: "http://localhost:11434",
	models.ProviderLocal:  "http://localhost:8080",
}

// localDefaultModels 本地模型默认模型名。
var localDefaultModels = map[models.ProviderType]string{
	models.ProviderOllama: "llama3.1-8b",
	models.ProviderLocal:  "llama3.1-8b",
}

// ProviderFactory 根据配置创建对应的 LLMClient 适配器。
//
// 路由规则：
//   - ProviderOllama  → LocalAdapter
//   - ProviderLocal   → OpenAIAdapter
//   - 云端 Provider   → OpenAIAdapter
func ProviderFactory(cfg *entity.AppConfig) port.LLMClient {
	// Ollama 本地模型路由
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

	// 通用本地端点路由
	if cfg.ProviderType == models.ProviderLocal {
		endpoint := cfg.APIEndpoint
		if endpoint == "" {
			endpoint = localEndpoints[models.ProviderLocal]
		}
		model := cfg.DefaultModel
		if model == "" {
			model = localDefaultModels[models.ProviderLocal]
		}
		return NewOpenAIAdapter("", endpoint, model)
	}

	// 云端模型路由
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

	return NewOpenAIAdapter("", baseURL, model)
}

// llmClientFactory 实现 port.LLMClientFactory，根据 ProviderConfig 动态创建 adapter。
type llmClientFactory struct{}

// NewLLMClientFactory 创建 LLMClientFactory 实例（返回接口供 Wire 注入）。
func NewLLMClientFactory() port.LLMClientFactory {
	return &llmClientFactory{}
}

// inferProviderTypeFromHost 根据 API Host 推断 ProviderType。
func inferProviderTypeFromHost(apiHost string) models.ProviderType {
	for pt, host := range defaultEndpoints {
		if strings.Contains(apiHost, host) {
			return pt
		}
	}
	for pt, host := range localEndpoints {
		if strings.Contains(apiHost, host) {
			return pt
		}
	}
	return models.ProviderKimi
}

// CreateClient 根据 ProviderConfig 创建对应的 LLMClient。
func (f *llmClientFactory) CreateClient(cfg *models.ProviderConfig) (port.LLMClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("provider config is nil")
	}

	providerType := inferProviderTypeFromHost(cfg.APIHost)

	// Ollama 本地模型路由
	if providerType == models.ProviderOllama {
		endpoint := cfg.APIHost
		if endpoint == "" {
			endpoint = localEndpoints[models.ProviderOllama]
		}
		model := cfg.ModelID
		if model == "" && len(cfg.Models) > 0 {
			model = cfg.Models[0].ID
		}
		if model == "" {
			model = localDefaultModels[models.ProviderOllama]
		}
		return NewLocalAdapter(endpoint, model), nil
	}

	// 通用本地端点路由
	if providerType == models.ProviderLocal {
		endpoint := cfg.APIHost
		if endpoint == "" {
			endpoint = localEndpoints[models.ProviderLocal]
		}
		model := cfg.ModelID
		if model == "" && len(cfg.Models) > 0 {
			model = cfg.Models[0].ID
		}
		if model == "" {
			model = localDefaultModels[models.ProviderLocal]
		}
		return NewOpenAIAdapter("", endpoint, model), nil
	}

	// 云端模型路由
	baseURL := cfg.APIHost
	if baseURL == "" {
		if u, ok := defaultEndpoints[providerType]; ok {
			baseURL = u
		} else {
			baseURL = defaultEndpoints[models.ProviderKimi]
		}
	}

	model := cfg.ModelID
	if model == "" && len(cfg.Models) > 0 {
		model = cfg.Models[0].ID
	}
	if model == "" {
		model = "kimi-lite"
	}

	return NewOpenAIAdapter(cfg.APIKey, baseURL, model), nil
}

// ProviderSet 供 Wire 使用的 ProviderSet。
var ProviderSet = wire.NewSet(
	ProviderFactory,
	NewLLMClientFactory,
)
