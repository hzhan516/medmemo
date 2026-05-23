package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLITokenService_Detect_Kimi_Installed 验证检测到已安装的 Kimi CLI。
func TestCLITokenService_Detect_Kimi_Installed(t *testing.T) {
	// 模拟 home 目录环境
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	kimiDir := filepath.Join(tmpHome, ".kimi", "credentials")
	require.NoError(t, os.MkdirAll(kimiDir, 0700))
	credPath := filepath.Join(kimiDir, "kimi-code.json")
	require.NoError(t, os.WriteFile(credPath, []byte(`{"access_token":"tk_kimi","expires_at":9999999999}`), 0600))

	svc := NewCLITokenService()
	result, err := svc.Detect("kimi")
	require.NoError(t, err)
	assert.True(t, result.Detected)
	assert.True(t, result.LoggedIn)
	assert.Equal(t, "kimi", result.ProviderType)
	assert.Empty(t, result.Error)
}

// TestCLITokenService_Detect_Kimi_NotInstalled 验证未安装 Kimi CLI 时 Detected=false。
func TestCLITokenService_Detect_Kimi_NotInstalled(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	svc := NewCLITokenService()
	result, err := svc.Detect("kimi")
	require.NoError(t, err)
	assert.False(t, result.Detected)
	assert.False(t, result.LoggedIn)
	assert.Empty(t, result.Error)
}

// TestCLITokenService_Detect_Gemini_Installed 验证检测到已安装的 Gemini CLI（ADC 文件）。
func TestCLITokenService_Detect_Gemini_Installed(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	gcloudDir := filepath.Join(tmpHome, ".config", "gcloud")
	require.NoError(t, os.MkdirAll(gcloudDir, 0700))
	credPath := filepath.Join(gcloudDir, "application_default_credentials.json")
	require.NoError(t, os.WriteFile(credPath, []byte(`{"refresh_token":"1//refresh","type":"authorized_user"}`), 0600))

	svc := NewCLITokenService()
	result, err := svc.Detect("gemini")
	require.NoError(t, err)
	assert.True(t, result.Detected)
	assert.True(t, result.LoggedIn) // refresh_token 也算 logged_in
	assert.Equal(t, "gemini", result.ProviderType)
	assert.Empty(t, result.Error)
}

// TestCLITokenService_Detect_UnsupportedProvider 验证不支持的 providerType 返回错误。
func TestCLITokenService_Detect_UnsupportedProvider(t *testing.T) {
	svc := NewCLITokenService()
	_, err := svc.Detect("unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

// TestCLITokenService_ReadToken_Kimi 验证读取 Kimi CLI token。
func TestCLITokenService_ReadToken_Kimi(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "kimi.json")
	require.NoError(t, os.WriteFile(credPath, []byte(`{"access_token":"tk_123"}`), 0600))

	svc := NewCLITokenService()
	token, needsRefresh, err := svc.ReadToken("kimi", credPath)
	assert.NoError(t, err)
	assert.Equal(t, "tk_123", token)
	assert.False(t, needsRefresh)
}

// TestCLITokenService_ReadToken_DefaultPath 验证空路径时使用默认路径。
func TestCLITokenService_ReadToken_DefaultPath(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	kimiDir := filepath.Join(tmpHome, ".kimi", "credentials")
	require.NoError(t, os.MkdirAll(kimiDir, 0700))
	credPath := filepath.Join(kimiDir, "kimi-code.json")
	require.NoError(t, os.WriteFile(credPath, []byte(`{"access_token":"tk_default"}`), 0600))

	svc := NewCLITokenService()
	token, needsRefresh, err := svc.ReadToken("kimi", "")
	assert.NoError(t, err)
	assert.Equal(t, "tk_default", token)
	assert.False(t, needsRefresh)
}

// TestCLITokenService_ReadToken_Gemini_RefreshHint 验证 Gemini refresh_token 返回 needsRefresh=true。
func TestCLITokenService_ReadToken_Gemini_RefreshHint(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "adc.json")
	require.NoError(t, os.WriteFile(credPath, []byte(`{"refresh_token":"1//xyz","type":"authorized_user"}`), 0600))

	svc := NewCLITokenService()
	token, needsRefresh, err := svc.ReadToken("gemini", credPath)
	assert.NoError(t, err)
	assert.True(t, needsRefresh)
	assert.Equal(t, "1//xyz", token)
}

// TestCLITokenService_ValidateToken_Valid 验证有效 token 返回 true。
func TestCLITokenService_ValidateToken_Valid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/models", r.URL.Path)
		assert.Equal(t, "Bearer valid_token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	svc := NewCLITokenServiceWithClient(server.Client())
	valid, err := svc.ValidateToken(context.Background(), server.URL, "valid_token")
	assert.NoError(t, err)
	assert.True(t, valid)
}

// TestCLITokenService_ValidateToken_Invalid 验证无效 token 返回 false。
func TestCLITokenService_ValidateToken_Invalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid token"}`))
	}))
	defer server.Close()

	svc := NewCLITokenServiceWithClient(server.Client())
	valid, err := svc.ValidateToken(context.Background(), server.URL, "invalid_token")
	assert.NoError(t, err)
	assert.False(t, valid)
}

// TestCLITokenService_ValidateToken_EmptyToken 验证空 token 返回错误。
func TestCLITokenService_ValidateToken_EmptyToken(t *testing.T) {
	svc := NewCLITokenService()
	_, err := svc.ValidateToken(context.Background(), "http://example.com", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is required")
}

// TestCLITokenService_ValidateToken_EmptyHost 验证空 apiHost 返回错误。
func TestCLITokenService_ValidateToken_EmptyHost(t *testing.T) {
	svc := NewCLITokenService()
	_, err := svc.ValidateToken(context.Background(), "", "token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "api_host is required")
}

// TestCLITokenService_BuildProviderConfig_Kimi 验证 Kimi CLI ProviderConfig 构建。
func TestCLITokenService_BuildProviderConfig_Kimi(t *testing.T) {
	svc := NewCLITokenService()
	cfg, err := svc.BuildProviderConfig("kimi", "")
	require.NoError(t, err)

	assert.Contains(t, cfg.ID, "cli-kimi-")
	assert.Equal(t, "Kimi (CLI)", cfg.Name)
	assert.Equal(t, "https://api.moonshot.cn", cfg.APIHost)
	assert.Equal(t, "moonshot-v1-8k", cfg.ModelID)
	assert.Equal(t, models.AuthMethodCLIToken, cfg.AuthMethod)
	assert.Equal(t, "~/.kimi/credentials/kimi-code.json", cfg.AuthParams.CLICredentialPath)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "CLI", cfg.GroupName)
}

// TestCLITokenService_BuildProviderConfig_Gemini 验证 Gemini CLI ProviderConfig 构建。
func TestCLITokenService_BuildProviderConfig_Gemini(t *testing.T) {
	svc := NewCLITokenService()
	cfg, err := svc.BuildProviderConfig("gemini", "gemini-pro")
	require.NoError(t, err)

	assert.Contains(t, cfg.ID, "cli-gemini-")
	assert.Equal(t, "Gemini (CLI)", cfg.Name)
	assert.Equal(t, "https://generativelanguage.googleapis.com/v1beta/openai/", cfg.APIHost)
	assert.Equal(t, "gemini-pro", cfg.ModelID)
	assert.Equal(t, models.AuthMethodCLIToken, cfg.AuthMethod)
	assert.Equal(t, "~/.config/gcloud/application_default_credentials.json", cfg.AuthParams.CLICredentialPath)
}

// TestCLITokenService_BuildProviderConfig_Unsupported 验证不支持类型返回错误。
func TestCLITokenService_BuildProviderConfig_Unsupported(t *testing.T) {
	svc := NewCLITokenService()
	_, err := svc.BuildProviderConfig("unknown", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

// TestCLITokenService_Detect_Kimi_EmptyFile 验证空凭证文件返回错误信息。
func TestCLITokenService_Detect_Kimi_EmptyFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	kimiDir := filepath.Join(tmpHome, ".kimi", "credentials")
	require.NoError(t, os.MkdirAll(kimiDir, 0700))
	credPath := filepath.Join(kimiDir, "kimi-code.json")
	require.NoError(t, os.WriteFile(credPath, []byte(""), 0600))

	svc := NewCLITokenService()
	result, err := svc.Detect("kimi")
	require.NoError(t, err)
	assert.True(t, result.Detected) // 文件存在
	assert.False(t, result.LoggedIn)
	assert.Contains(t, result.Error, "empty")
}

// TestCLITokenService_Integration 端到端集成测试：Detect → ReadToken → ValidateToken。
func TestCLITokenService_Integration(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	kimiDir := filepath.Join(tmpHome, ".kimi", "credentials")
	require.NoError(t, os.MkdirAll(kimiDir, 0700))
	credPath := filepath.Join(kimiDir, "kimi-code.json")
	require.NoError(t, os.WriteFile(credPath, []byte(`{"access_token":"tk_integration"}`), 0600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer tk_integration" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	svc := NewCLITokenServiceWithClient(server.Client())

	// 1. Detect
	result, err := svc.Detect("kimi")
	require.NoError(t, err)
	assert.True(t, result.Detected)
	assert.True(t, result.LoggedIn)

	// 2. ReadToken
	token, needsRefresh, err := svc.ReadToken("kimi", "")
	require.NoError(t, err)
	assert.Equal(t, "tk_integration", token)
	assert.False(t, needsRefresh)

	// 3. ValidateToken（使用模拟端点）
	valid, err := svc.ValidateToken(context.Background(), server.URL, token)
	require.NoError(t, err)
	assert.True(t, valid)

	// 4. BuildProviderConfig
	cfg, err := svc.BuildProviderConfig("kimi", "moonshot-v1-32k")
	require.NoError(t, err)
	assert.Equal(t, "moonshot-v1-32k", cfg.ModelID)
	assert.Equal(t, models.AuthMethodCLIToken, cfg.AuthMethod)

	// 验证 ValidateToken 对错误 token 返回 false
	valid, err = svc.ValidateToken(context.Background(), server.URL, "wrong_token")
	require.NoError(t, err)
	assert.False(t, valid)
}

// TestCLITokenService_ValidateToken_ContextTimeout 验证 context 超时处理。
func TestCLITokenService_ValidateToken_ContextTimeout(t *testing.T) {
	// 创建一个缓慢响应的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	svc := NewCLITokenServiceWithClient(server.Client())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := svc.ValidateToken(ctx, server.URL, "token")
	assert.Error(t, err)
	// 可能是 context deadline exceeded 或 request failed
	assert.Contains(t, err.Error(), "request failed")
}
