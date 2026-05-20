package models

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

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

// AuthMethod 表示 Provider 的认证方式。
type AuthMethod string

const (
	// AuthMethodAPIToken 表示通过 API Key（Bearer Token）认证。
	AuthMethodAPIToken AuthMethod = "api_key"
	// AuthMethodCLIToken 表示通过本地 CLI 缓存的 OAuth Token 认证。
	AuthMethodCLIToken AuthMethod = "cli_token"
	// AuthMethodOAuthDevice 表示通过 OAuth 2.0 Device Flow 获取的 Token 认证。
	AuthMethodOAuthDevice AuthMethod = "oauth_device"
	// AuthMethodServiceAccount 表示通过 GCP Service Account JSON 认证。
	AuthMethodServiceAccount AuthMethod = "service_account"
)

// AuthParams 存储各认证方式特有的配置参数。
// 根据 AuthMethod 的不同，仅填充对应方式的字段，其余字段为空。
type AuthParams struct {
	// API Key 方式（api_key）
	APIKey string `json:"api_key,omitempty"`

	// CLI Token 方式（cli_token）
	CLICredentialPath string `json:"cli_credential_path,omitempty"`

	// OAuth Device Flow 方式（oauth_device）
	OAuthClientID     string `json:"oauth_client_id,omitempty"`
	OAuthAuthURL      string `json:"oauth_auth_url,omitempty"`
	OAuthTokenURL     string `json:"oauth_token_url,omitempty"`
	OAuthRefreshToken string `json:"oauth_refresh_token,omitempty"`
	OAuthAccessToken  string `json:"oauth_access_token,omitempty"`
	OAuthExpiresAt    int64  `json:"oauth_expires_at,omitempty"`

	// Service Account 方式（service_account）
	GCPProjectID string `json:"gcp_project_id,omitempty"`
	GCPRegion    string `json:"gcp_region,omitempty"`
	SAJSON       string `json:"sa_json,omitempty"`
}

// ProviderConfig 表示 LLM Provider 的完整配置。
// 零硬编码：所有字段均由调用方动态提供，不绑定任何厂商默认值。
type ProviderConfig struct {
	ID          string        // 唯一标识
	Name        string        // 显示名称
	APIHost     string        // API 基础地址，如 https://api.openai.com
	APIKey      string        // 认证密钥（api_key 方式下使用）
	ModelID     string        // 模型标识
	Temperature float64       // 采样温度，范围 0-2，默认 0.7
	Timeout     time.Duration // 单次请求超时，默认 30s
	MaxRetries  int           // 最大重试次数，默认 3
	GroupName   string        // 分组名称
	Enabled     bool          // 是否启用
	SortOrder   int           // 排序权重
	CreatedAt   time.Time     // 创建时间
	UpdatedAt   time.Time     // 更新时间

	// AuthMethod 指定认证方式，默认 api_key。
	// 空字符串向后兼容视为 api_key。
	AuthMethod AuthMethod `json:"auth_method"`

	// AuthParams 存储各认证方式的特有参数。
	AuthParams AuthParams `json:"auth_params"`
}

// Validate 根据 AuthMethod 校验 ProviderConfig 的必填字段。
// AuthMethod 为空字符串时向后兼容视为 api_key。
func (p *ProviderConfig) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("provider id is required")
	}
	if p.Name == "" {
		return fmt.Errorf("provider name is required")
	}
	if p.APIHost == "" {
		return fmt.Errorf("provider api_host is required")
	}
	u, err := url.Parse(p.APIHost)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("provider api_host must be a valid http(s) URL")
	}
	if p.ModelID == "" {
		return fmt.Errorf("provider model_id is required")
	}
	if p.Temperature < 0 || p.Temperature > 2 {
		return fmt.Errorf("provider temperature must be in range [0, 2]")
	}

	method := p.AuthMethod
	if method == "" {
		method = AuthMethodAPIToken
	}

	switch method {
	case AuthMethodAPIToken:
		if p.APIKey == "" {
			return fmt.Errorf("provider api_key is required for auth method api_key")
		}
	case AuthMethodCLIToken:
		// apiKey 可为空，运行时从 CLI 缓存读取
		if p.AuthParams.CLICredentialPath == "" {
			return fmt.Errorf("provider cli_credential_path is required for auth method cli_token")
		}
	case AuthMethodOAuthDevice:
		if p.AuthParams.OAuthClientID == "" {
			return fmt.Errorf("provider oauth_client_id is required for auth method oauth_device")
		}
		if p.AuthParams.OAuthTokenURL == "" {
			return fmt.Errorf("provider oauth_token_url is required for auth method oauth_device")
		}
	case AuthMethodServiceAccount:
		if p.AuthParams.GCPProjectID == "" {
			return fmt.Errorf("provider gcp_project_id is required for auth method service_account")
		}
		if p.AuthParams.SAJSON == "" {
			return fmt.Errorf("provider sa_json is required for auth method service_account")
		}
	default:
		return fmt.Errorf("unknown auth method: %s", method)
	}

	return nil
}

// ResolveAuthToken 根据 AuthMethod 获取用于 HTTP 请求的认证 Token。
// 对于尚未实现的认证方式，返回明确的错误信息。
// AuthMethod 为空字符串时向后兼容视为 api_key。
func (p *ProviderConfig) ResolveAuthToken() (string, error) {
	method := p.AuthMethod
	if method == "" {
		method = AuthMethodAPIToken
	}

	switch method {
	case AuthMethodAPIToken:
		// 允许空 API Key（如本地 Ollama 无需认证），由调用方决定是否发送 Authorization
		return p.APIKey, nil
	case AuthMethodCLIToken:
		token, hint, err := ReadCLITokenFromFile(p.AuthParams.CLICredentialPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve cli token: %w", err)
		}
		if hint == "" {
			// 文件直接包含可用的 access_token
			return token, nil
		}
		// hint == "refresh_token"，检查内存缓存的 access_token
		if p.AuthParams.OAuthAccessToken != "" && p.AuthParams.OAuthExpiresAt > time.Now().Unix() {
			return p.AuthParams.OAuthAccessToken, nil
		}
		return "", fmt.Errorf("access_token expired, refresh required")
	case AuthMethodOAuthDevice:
		return "", fmt.Errorf("oauth_device auth not yet implemented (TASK-046)")
	case AuthMethodServiceAccount:
		return "", fmt.Errorf("service_account auth not yet implemented")
	default:
		return "", fmt.Errorf("unknown auth method: %s", method)
	}
}

// MarshalAuthParams 将 AuthParams 序列化为 JSON 字符串，供数据库存储。
func (p *ProviderConfig) MarshalAuthParams() (string, error) {
	b, err := json.Marshal(p.AuthParams)
	if err != nil {
		return "", fmt.Errorf("failed to marshal auth params: %w", err)
	}
	return string(b), nil
}

// UnmarshalAuthParams 从 JSON 字符串反序列化到 AuthParams。
func (p *ProviderConfig) UnmarshalAuthParams(data string) error {
	if data == "" || data == "{}" {
		p.AuthParams = AuthParams{}
		return nil
	}
	if err := json.Unmarshal([]byte(data), &p.AuthParams); err != nil {
		return fmt.Errorf("failed to unmarshal auth params: %w", err)
	}
	return nil
}

// ModelConfig 表示单个模型的配置信息。
type ModelConfig struct {
	Provider    ProviderType `json:"provider"`
	ModelName   string       `json:"model_name"`
	APIEndpoint string       `json:"api_endpoint"`
	MaxTokens   int          `json:"max_tokens"`
	Temperature float64      `json:"temperature"`
}
