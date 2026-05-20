package models

import "time"

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

// ProviderConfig 表示 LLM Provider 的完整配置。
// 零硬编码：所有字段均由调用方动态提供，不绑定任何厂商默认值。
type ProviderConfig struct {
	ID          string        // 唯一标识
	Name        string        // 显示名称
	APIHost     string        // API 基础地址，如 https://api.openai.com
	APIKey      string        // 认证密钥
	ModelID     string        // 模型标识
	Temperature float64       // 采样温度，范围 0-2，默认 0.7
	Timeout     time.Duration // 单次请求超时，默认 30s
	MaxRetries  int           // 最大重试次数，默认 3
	GroupName   string        // 分组名称
	Enabled     bool          // 是否启用
	SortOrder   int           // 排序权重
	CreatedAt   time.Time     // 创建时间
	UpdatedAt   time.Time     // 更新时间
}

// ModelConfig 表示单个模型的配置信息。
type ModelConfig struct {
	Provider    ProviderType `json:"provider"`
	ModelName   string       `json:"model_name"`
	APIEndpoint string       `json:"api_endpoint"`
	MaxTokens   int          `json:"max_tokens"`
	Temperature float64      `json:"temperature"`
}
