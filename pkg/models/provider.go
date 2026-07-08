package models

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// ProviderType LLM 提供商类型。
type ProviderType string

// noinspection GoUnusedConst

const (
	ProviderKimi        ProviderType = "kimi"
	ProviderOpenAI      ProviderType = "openai"
	ProviderQwen        ProviderType = "qwen"
	ProviderSiliconFlow ProviderType = "siliconflow"
	ProviderOllama      ProviderType = "ollama"
	ProviderLocal       ProviderType = "local"
	ProviderMicrosoft   ProviderType = "microsoft"
	ProviderGitHub      ProviderType = "github"
	ProviderClaude      ProviderType = "claude"
	ProviderGrok        ProviderType = "grok"
	ProviderDoubao      ProviderType = "doubao"
	ProviderGLM         ProviderType = "zhipu"
	ProviderDeepSeek    ProviderType = "deepseek"
	ProviderMiniMax     ProviderType = "minimax"
	ProviderXiaomi      ProviderType = "xiaomi"
	ProviderHunyuan     ProviderType = "hunyuan"
	ProviderVertex      ProviderType = "vertex"
)

// AuthMethod Provider 认证方式。
type AuthMethod string

// noinspection GoUnusedConst

const (
	AuthMethodAPIToken       AuthMethod = "api_key"
	AuthMethodCLIToken       AuthMethod = "cli_token"
	AuthMethodOAuthDevice    AuthMethod = "oauth_device"
	AuthMethodServiceAccount AuthMethod = "service_account"
)

// AuthParams 各认证方式特有配置参数。
type AuthParams struct {
	APIKey string `json:"api_key,omitempty"`

	CLICredentialPath string `json:"cli_credential_path,omitempty"`

	OAuthClientID string `json:"oauth_client_id,omitempty"`
	OAuthAuthURL  string `json:"oauth_auth_url,omitempty"`
	OAuthTokenURL string `json:"oauth_token_url,omitempty"`
	// OAuthRefreshToken 不序列化到数据库，运行时内存持有
	OAuthRefreshToken string `json:"-"`
	// OAuthAccessToken 不序列化到数据库，运行时内存持有
	OAuthAccessToken string `json:"-"`
	OAuthExpiresAt   int64  `json:"oauth_expires_at,omitempty"`

	GCPProjectID string `json:"gcp_project_id,omitempty"`
	GCPRegion    string `json:"gcp_region,omitempty"`
	SAJSON       string `json:"sa_json,omitempty"`
}

// ProviderModel 表示某个服务商下的单个模型配置。
type ProviderModel struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Enabled          bool   `json:"enabled"`
	MaxContextLength int    `json:"maxContextLength,omitempty"`
}

// ProviderConfig LLM Provider 完整配置。
// json tag 使用 camelCase 以匹配前端类型定义和 Wails 绑定生成。
type ProviderConfig struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Type        ProviderType    `json:"type"`
	APIHost     string          `json:"apiHost"`
	APIKey      string          `json:"apiKey,omitempty"`
	ModelID     string          `json:"modelId"`
	Models      []ProviderModel `json:"models"`
	Temperature float64         `json:"temperature"`
	TimeoutMs   int             `json:"timeoutMs"`
	MaxRetries  int             `json:"maxRetries"`
	MaxTokens   int             `json:"maxTokens"`
	GroupName   string          `json:"group"`
	Enabled     bool            `json:"enabled"`
	SortOrder   int             `json:"sortOrder"`
	CreatedAt   int64           `json:"createdAt"`
	UpdatedAt   int64           `json:"updatedAt"`

	AuthMethod AuthMethod `json:"auth_method"`
	AuthParams AuthParams `json:"auth_params"`
}

// Validate 根据 AuthMethod 校验必填字段。
func (p *ProviderConfig) Validate() error {
	if err := p.validateBaseFields(); err != nil {
		return fmt.Errorf("failed to validate base fields: %w", err)
	}
	return p.validateAuthParams()
}

// validateBaseFields 校验 Provider 通用基础字段。
func (p *ProviderConfig) validateBaseFields() error {
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
	// 向后兼容：有 Models 列表时可不填 ModelID（自动取第一个启用模型）
	if p.ModelID == "" && len(p.Models) == 0 {
		return fmt.Errorf("provider model_id is required")
	}
	if p.Temperature < 0 || p.Temperature > 2 {
		return fmt.Errorf("provider temperature must be in range [0, 2]")
	}
	return nil
}

// validateAuthParams 根据 AuthMethod 校验认证参数。
func (p *ProviderConfig) validateAuthParams() error {
	method := p.AuthMethod
	if method == "" {
		method = AuthMethodAPIToken
	}

	switch method {
	case AuthMethodAPIToken:
		return validateAPIToken(p.APIKey)
	case AuthMethodCLIToken:
		return validateCLIToken(p.AuthParams.CLICredentialPath)
	case AuthMethodOAuthDevice:
		return validateOAuthDevice(p.AuthParams.OAuthClientID, p.AuthParams.OAuthTokenURL)
	case AuthMethodServiceAccount:
		return validateServiceAccount(p.AuthParams.GCPProjectID, p.AuthParams.SAJSON)
	default:
		return fmt.Errorf("unknown auth method: %s", method)
	}
}

func validateAPIToken(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("provider api_key is required for auth method api_key")
	}
	return nil
}

func validateCLIToken(path string) error {
	if path == "" {
		return fmt.Errorf("provider cli_credential_path is required for auth method cli_token")
	}
	return nil
}

func validateOAuthDevice(clientID, tokenURL string) error {
	if clientID == "" {
		return fmt.Errorf("provider oauth_client_id is required for auth method oauth_device")
	}
	if tokenURL == "" {
		return fmt.Errorf("provider oauth_token_url is required for auth method oauth_device")
	}
	return nil
}

func validateServiceAccount(projectID, saJSON string) error {
	if projectID == "" {
		return fmt.Errorf("provider gcp_project_id is required for auth method service_account")
	}
	if saJSON == "" {
		return fmt.Errorf("provider sa_json is required for auth method service_account")
	}
	return nil
}

// ResolveAuthToken 根据 AuthMethod 获取 HTTP 认证 Token。
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
		// 优先检查内存缓存的 access_token
		if p.AuthParams.OAuthAccessToken != "" && p.AuthParams.OAuthExpiresAt > time.Now().Unix() {
			return p.AuthParams.OAuthAccessToken, nil
		}
		return "", fmt.Errorf("access_token expired, refresh required")
	case AuthMethodServiceAccount:
		return "", fmt.Errorf("service_account auth not yet implemented")
	default:
		return "", fmt.Errorf("unknown auth method: %s", method)
	}
}

// MarshalAuthParams 序列化为 JSON 字符串供数据库存储。
func (p *ProviderConfig) MarshalAuthParams() (string, error) {
	b, err := json.Marshal(p.AuthParams)
	if err != nil {
		return "", fmt.Errorf("failed to marshal auth params: %w", err)
	}
	return string(b), nil
}

// UnmarshalAuthParams 从 JSON 字符串反序列化。
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

// InferProviderType 根据 API Host 推断 ProviderType。
// 用于后端在仅有 api_host 时还原 provider 类型，辅助本地/云端判据。
// 无法识别时返回空字符串。
func InferProviderType(apiHost string) ProviderType {
	u, err := url.Parse(apiHost)
	if err != nil {
		return ""
	}

	host := u.Hostname()
	port := u.Port()

	// 本地回环端点：按常用端口推断 Ollama / 通用本地，其余统一视为 local。
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		switch port {
		case "11434":
			return ProviderOllama
		case "8080":
			return ProviderLocal
		default:
			return ProviderLocal
		}
	}

	// 已知云端厂商默认域名。
	knownCloudHosts := map[string]ProviderType{
		"api.moonshot.cn":                   ProviderKimi,
		"api.openai.com":                    ProviderOpenAI,
		"dashscope.aliyuncs.com":            ProviderQwen,
		"api.siliconflow.cn":                ProviderSiliconFlow,
		"models.inference.ai.azure.com":     ProviderMicrosoft,
		"api.anthropic.com":                 ProviderClaude,
		"api.x.ai":                          ProviderGrok,
		"ark.cn-beijing.volces.com":         ProviderDoubao,
		"open.bigmodel.cn":                  ProviderGLM,
		"api.deepseek.com":                  ProviderDeepSeek,
		"api.minimax.chat":                  ProviderMiniMax,
		"api.mi.ai":                         ProviderXiaomi,
		"hunyuan.tencentcloudapi.com":       ProviderHunyuan,
		"generativelanguage.googleapis.com": ProviderKimi, // 未定义 gemini，保守归为 kimi 云端
	}

	for h, pt := range knownCloudHosts {
		if strings.Contains(host, h) {
			return pt
		}
	}

	return ""
}

// ModelConfig 单个模型配置信息。
type ModelConfig struct {
	Provider    ProviderType `json:"provider"`
	ModelName   string       `json:"model_name"`
	APIEndpoint string       `json:"api_endpoint"`
	MaxTokens   int          `json:"max_tokens"`
	Temperature float64      `json:"temperature"`
}
