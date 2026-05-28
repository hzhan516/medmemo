package ai

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/wire"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
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

// defaultRequestTimeout ProviderFactory 默认请求超时（非流式场景）。
const defaultRequestTimeout = 30 * time.Second

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
		return NewLocalAdapter(endpoint, model, 0, defaultRequestTimeout)
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
		return NewOpenAIAdapter("", endpoint, model, 0, defaultRequestTimeout)
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

	return NewOpenAIAdapter("", baseURL, model, 0, defaultRequestTimeout)
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
	switch providerType {
	case models.ProviderOllama:
		return createOllamaClient(cfg), nil
	case models.ProviderLocal:
		return createLocalClient(cfg), nil
	default:
		return createCloudClient(cfg, providerType), nil
	}
}

// resolveTimeout 根据 ProviderConfig.TimeoutMs 解析超时，未配置则默认 120 秒。
func resolveTimeout(timeoutMs int) time.Duration {
	if timeoutMs > 0 {
		return time.Duration(timeoutMs) * time.Millisecond
	}
	return 120 * time.Second
}

// createOllamaClient 创建 Ollama 本地模型适配器。
func createOllamaClient(cfg *models.ProviderConfig) port.LLMClient {
	endpoint := resolveEndpoint(cfg.APIHost, localEndpoints[models.ProviderOllama])
	model := resolveModel(cfg.ModelID, cfg.Models, localDefaultModels[models.ProviderOllama])
	return NewLocalAdapter(endpoint, model, cfg.MaxTokens, resolveTimeout(cfg.TimeoutMs))
}

// createLocalClient 创建通用本地端点适配器。
func createLocalClient(cfg *models.ProviderConfig) port.LLMClient {
	endpoint := resolveEndpoint(cfg.APIHost, localEndpoints[models.ProviderLocal])
	model := resolveModel(cfg.ModelID, cfg.Models, localDefaultModels[models.ProviderLocal])
	return NewOpenAIAdapter("", endpoint, model, cfg.MaxTokens, resolveTimeout(cfg.TimeoutMs))
}

// createCloudClient 创建云端模型适配器。
func createCloudClient(cfg *models.ProviderConfig, providerType models.ProviderType) port.LLMClient {
	baseURL := resolveEndpoint(cfg.APIHost, defaultEndpoints[providerType])
	if baseURL == "" {
		baseURL = defaultEndpoints[models.ProviderKimi]
	}
	model := resolveModel(cfg.ModelID, cfg.Models, "kimi-lite")
	return NewOpenAIAdapter(cfg.APIKey, baseURL, model, cfg.MaxTokens, resolveTimeout(cfg.TimeoutMs))
}

// resolveEndpoint 返回有效的 API 端点，优先使用用户配置，否则使用默认值。
func resolveEndpoint(cfgValue, defaultValue string) string {
	if cfgValue != "" {
		return cfgValue
	}
	return defaultValue
}

// resolveModel 返回有效的模型 ID，优先使用用户配置，其次取 Models 列表首个，最后使用默认值。
func resolveModel(modelID string, models []models.ProviderModel, defaultModel string) string {
	if modelID != "" {
		return modelID
	}
	if len(models) > 0 {
		return models[0].ID
	}
	return defaultModel
}

// ProviderSet 供 Wire 使用的 ProviderSet。
var ProviderSet = wire.NewSet(
	ProviderFactory,
	NewLLMClientFactory,
)
