package models

// ProviderType 表示 LLM 提供商类型。
type ProviderType string

const (
	ProviderKimi        ProviderType = "kimi"
	ProviderOpenAI      ProviderType = "openai"
	ProviderQwen        ProviderType = "qwen"
	ProviderSiliconFlow ProviderType = "siliconflow"
	ProviderOllama      ProviderType = "ollama"
	ProviderLocal       ProviderType = "local"
)

// ModelConfig 表示单个模型的配置信息。
type ModelConfig struct {
	Provider    ProviderType `json:"provider"`
	ModelName   string       `json:"model_name"`
	APIEndpoint string       `json:"api_endpoint"`
	MaxTokens   int          `json:"max_tokens"`
	Temperature float64      `json:"temperature"`
}
