package models

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newValidProviderConfig 返回一个可用于测试的基础 ProviderConfig。
func newValidProviderConfig() ProviderConfig {
	return ProviderConfig{
		ID:          "test-id",
		Name:        "Test Provider",
		APIHost:     "https://api.example.com",
		APIKey:      "sk-test-key",
		ModelID:     "gpt-4o",
		Temperature: 0.7,
		TimeoutMs:   30000,
		MaxRetries:  3,
		Enabled:     true,
	}
}

// TestProviderConfig_Validate_APIKey_Success 验证 api_key 方式正常通过。
func TestProviderConfig_Validate_APIKey_Success(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodAPIToken
	assert.NoError(t, p.Validate())
}

// TestProviderConfig_Validate_EmptyAuthMethod_DefaultsToAPIKey 验证空 AuthMethod 向后兼容。
func TestProviderConfig_Validate_EmptyAuthMethod_DefaultsToAPIKey(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = ""
	assert.NoError(t, p.Validate())
}

// TestProviderConfig_Validate_APIKey_MissingKey 验证 api_key 方式缺少 key 时报错。
func TestProviderConfig_Validate_APIKey_MissingKey(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodAPIToken
	p.APIKey = ""
	err := p.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "api_key is required")
}

// TestProviderConfig_Validate_CLIToken_Success 验证 cli_token 方式正常通过。
func TestProviderConfig_Validate_CLIToken_Success(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodCLIToken
	p.APIKey = "" // cli_token 允许空 api_key
	p.AuthParams = AuthParams{CLICredentialPath: "~/.kimi/credentials/kimi-code.json"}
	assert.NoError(t, p.Validate())
}

// TestProviderConfig_Validate_CLIToken_MissingCredentialPath 验证 cli_token 方式缺少路径时报错。
func TestProviderConfig_Validate_CLIToken_MissingCredentialPath(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodCLIToken
	p.APIKey = ""
	p.AuthParams = AuthParams{}
	err := p.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cli_credential_path is required")
}

// TestProviderConfig_Validate_OAuthDevice_Success 验证 oauth_device 方式正常通过。
func TestProviderConfig_Validate_OAuthDevice_Success(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodOAuthDevice
	p.APIKey = ""
	p.AuthParams = AuthParams{
		OAuthClientID: "client-123",
		OAuthTokenURL: "https://auth.example.com/token",
	}
	assert.NoError(t, p.Validate())
}

// TestProviderConfig_Validate_OAuthDevice_MissingClientID 验证 oauth_device 方式缺少 client_id 时报错。
func TestProviderConfig_Validate_OAuthDevice_MissingClientID(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodOAuthDevice
	p.APIKey = ""
	p.AuthParams = AuthParams{OAuthTokenURL: "https://auth.example.com/token"}
	err := p.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "oauth_client_id is required")
}

// TestProviderConfig_Validate_OAuthDevice_MissingTokenURL 验证 oauth_device 方式缺少 token_url 时报错。
func TestProviderConfig_Validate_OAuthDevice_MissingTokenURL(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodOAuthDevice
	p.APIKey = ""
	p.AuthParams = AuthParams{OAuthClientID: "client-123"}
	err := p.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "oauth_token_url is required")
}

// TestProviderConfig_Validate_ServiceAccount_Success 验证 service_account 方式正常通过。
func TestProviderConfig_Validate_ServiceAccount_Success(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodServiceAccount
	p.APIKey = ""
	p.AuthParams = AuthParams{
		GCPProjectID: "my-project-123",
		SAJSON:       `{"type":"service_account"}`,
	}
	assert.NoError(t, p.Validate())
}

// TestProviderConfig_Validate_ServiceAccount_MissingProjectID 验证 service_account 方式缺少 project_id 时报错。
func TestProviderConfig_Validate_ServiceAccount_MissingProjectID(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodServiceAccount
	p.APIKey = ""
	p.AuthParams = AuthParams{SAJSON: `{"type":"service_account"}`}
	err := p.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gcp_project_id is required")
}

// TestProviderConfig_Validate_ServiceAccount_MissingSAJSON 验证 service_account 方式缺少 sa_json 时报错。
func TestProviderConfig_Validate_ServiceAccount_MissingSAJSON(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodServiceAccount
	p.APIKey = ""
	p.AuthParams = AuthParams{GCPProjectID: "my-project-123"}
	err := p.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sa_json is required")
}

// TestProviderConfig_Validate_CommonFields 验证通用字段校验。
func TestProviderConfig_Validate_CommonFields(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		mod  func(*ProviderConfig)
		err  string
	}{
		{"empty_id", func(p *ProviderConfig) { p.ID = "" }, "id is required"},
		{"empty_name", func(p *ProviderConfig) { p.Name = "" }, "name is required"},
		{"empty_api_host", func(p *ProviderConfig) { p.APIHost = "" }, "api_host is required"},
		{"invalid_scheme", func(p *ProviderConfig) { p.APIHost = "ftp://example.com" }, "api_host must be a valid http(s) URL"},
		{"empty_model_id", func(p *ProviderConfig) { p.ModelID = "" }, "model_id is required"},
		{"negative_temperature", func(p *ProviderConfig) { p.Temperature = -0.1 }, "temperature must be in range"},
		{"excessive_temperature", func(p *ProviderConfig) { p.Temperature = 2.1 }, "temperature must be in range"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newValidProviderConfig()
			p.AuthMethod = AuthMethodAPIToken
			tt.mod(&p)
			err := p.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.err)
		})
	}
}

// TestProviderConfig_Validate_UnknownAuthMethod 验证未知认证方式报错。
func TestProviderConfig_Validate_UnknownAuthMethod(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = "unknown_method"
	err := p.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown auth method")
}

// TestProviderConfig_ResolveAuthToken_APIKey 验证 api_key 方式返回正确 token。
func TestProviderConfig_ResolveAuthToken_APIKey(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodAPIToken
	token, err := p.ResolveAuthToken()
	assert.NoError(t, err)
	assert.Equal(t, "sk-test-key", token)
}

// TestProviderConfig_ResolveAuthToken_APIKey_Empty 验证 api_key 方式允许空 token（本地模型无需认证）。
func TestProviderConfig_ResolveAuthToken_APIKey_Empty(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodAPIToken
	p.APIKey = ""
	token, err := p.ResolveAuthToken()
	assert.NoError(t, err)
	assert.Empty(t, token)
}

// TestProviderConfig_ResolveAuthToken_CLIToken 验证 cli_token 方式正确读取凭证文件。
func TestProviderConfig_ResolveAuthToken_CLIToken(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "kimi-code.json")
	content := `{"access_token":"cli-token-abc","refresh_token":"rt_xyz"}`
	require.NoError(t, os.WriteFile(credPath, []byte(content), 0600))

	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodCLIToken
	p.APIKey = ""
	p.AuthParams = AuthParams{CLICredentialPath: credPath}

	token, err := p.ResolveAuthToken()
	assert.NoError(t, err)
	assert.Equal(t, "cli-token-abc", token)
}

// TestProviderConfig_ResolveAuthToken_CLIToken_MissingFile 验证 cli_token 凭证文件不存在时返回错误。
func TestProviderConfig_ResolveAuthToken_CLIToken_MissingFile(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodCLIToken
	p.APIKey = ""
	p.AuthParams = AuthParams{CLICredentialPath: "/nonexistent/cred.json"}

	_, err := p.ResolveAuthToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve cli token")
}

// TestProviderConfig_ResolveAuthToken_CLIToken_RefreshTokenHint 验证检测到 refresh_token 且缓存过期时返回错误。
func TestProviderConfig_ResolveAuthToken_CLIToken_RefreshTokenHint(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "adc.json")
	content := `{"refresh_token":"1//refresh","type":"authorized_user"}`
	require.NoError(t, os.WriteFile(credPath, []byte(content), 0600))

	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodCLIToken
	p.APIKey = ""
	p.AuthParams = AuthParams{CLICredentialPath: credPath}

	_, err := p.ResolveAuthToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access_token expired, refresh required")
}

// TestProviderConfig_ResolveAuthToken_CLIToken_CachedAccessToken 验证 refresh_token 场景下缓存的 access_token 可用。
func TestProviderConfig_ResolveAuthToken_CLIToken_CachedAccessToken(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "adc.json")
	content := `{"refresh_token":"1//refresh","type":"authorized_user"}`
	require.NoError(t, os.WriteFile(credPath, []byte(content), 0600))

	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodCLIToken
	p.APIKey = ""
	p.AuthParams = AuthParams{
		CLICredentialPath: credPath,
		OAuthAccessToken:  "cached_acc_token",
		OAuthExpiresAt:    time.Now().Add(1 * time.Hour).Unix(),
	}

	token, err := p.ResolveAuthToken()
	assert.NoError(t, err)
	assert.Equal(t, "cached_acc_token", token)
}

// TestProviderConfig_ResolveAuthToken_OAuthDevice 验证 oauth_device 方式缓存过期时返回错误。
func TestProviderConfig_ResolveAuthToken_OAuthDevice(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodOAuthDevice
	_, err := p.ResolveAuthToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access_token expired, refresh required")
}

// TestProviderConfig_ResolveAuthToken_OAuthDevice_CachedAccessToken 验证 oauth_device 方式缓存命中时正确返回 token。
func TestProviderConfig_ResolveAuthToken_OAuthDevice_CachedAccessToken(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodOAuthDevice
	p.AuthParams = AuthParams{
		OAuthClientID:    "client-123",
		OAuthTokenURL:    "https://auth.example.com/token",
		OAuthAccessToken: "oauth_cached_token",
		OAuthExpiresAt:   time.Now().Add(1 * time.Hour).Unix(),
	}

	token, err := p.ResolveAuthToken()
	assert.NoError(t, err)
	assert.Equal(t, "oauth_cached_token", token)
}

// TestProviderConfig_ResolveAuthToken_ServiceAccount 验证 service_account 方式返回未实现错误。
func TestProviderConfig_ResolveAuthToken_ServiceAccount(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = AuthMethodServiceAccount
	_, err := p.ResolveAuthToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

// TestProviderConfig_ResolveAuthToken_EmptyMethod_DefaultsToAPIKey 验证空 AuthMethod 向后兼容。
func TestProviderConfig_ResolveAuthToken_EmptyMethod_DefaultsToAPIKey(t *testing.T) {
	t.Parallel()
	p := newValidProviderConfig()
	p.AuthMethod = ""
	token, err := p.ResolveAuthToken()
	assert.NoError(t, err)
	assert.Equal(t, "sk-test-key", token)
}

// TestProviderConfig_MarshalUnmarshalAuthParams 验证 AuthParams JSON 序列化/反序列化。
func TestProviderConfig_MarshalUnmarshalAuthParams(t *testing.T) {
	t.Parallel()
	p := ProviderConfig{
		AuthParams: AuthParams{
			OAuthClientID:     "client-123",
			OAuthTokenURL:     "https://auth.example.com/token",
			OAuthRefreshToken: "refresh-abc",
			OAuthAccessToken:  "access-xyz",
			OAuthExpiresAt:    1234567890,
		},
	}

	jsonStr, err := p.MarshalAuthParams()
	assert.NoError(t, err)
	assert.Contains(t, jsonStr, "client-123")

	p2 := ProviderConfig{}
	assert.NoError(t, p2.UnmarshalAuthParams(jsonStr))
	assert.Equal(t, "client-123", p2.AuthParams.OAuthClientID)
	assert.Equal(t, "https://auth.example.com/token", p2.AuthParams.OAuthTokenURL)
	// OAuthRefreshToken / OAuthAccessToken 标记为 json:"-"，不参与序列化
	assert.Equal(t, "", p2.AuthParams.OAuthRefreshToken)
	assert.Equal(t, "", p2.AuthParams.OAuthAccessToken)
	assert.Equal(t, int64(1234567890), p2.AuthParams.OAuthExpiresAt)
}

// TestProviderConfig_UnmarshalAuthParams_Empty 验证空字符串反序列化为零值。
func TestProviderConfig_UnmarshalAuthParams_Empty(t *testing.T) {
	t.Parallel()
	p := ProviderConfig{}
	assert.NoError(t, p.UnmarshalAuthParams(""))
	assert.Equal(t, AuthParams{}, p.AuthParams)

	p2 := ProviderConfig{}
	assert.NoError(t, p2.UnmarshalAuthParams("{}"))
	assert.Equal(t, AuthParams{}, p2.AuthParams)
}
