// Package auth 实现 CLI Token 自动发现与读取服务。
// 将外部 CLI 工具（Kimi CLI、Gemini CLI/gcloud）的凭证格式适配到系统内部。
package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hzhan516/medmemo/pkg/models"
)

// CLIDetectResult CLI 检测结果，供前端展示。
type CLIDetectResult struct {
	ProviderType   string `json:"provider_type"`   // "kimi" | "gemini"
	Detected       bool   `json:"detected"`        // CLI 凭证文件是否存在
	CredentialPath string `json:"credential_path"` // 检测到的凭证文件路径
	LoggedIn       bool   `json:"logged_in"`       // 文件内容是否包含有效 token 信息
	Error          string `json:"error,omitempty"` // 检测过程中的错误提示
}

// 各 CLI 工具的默认凭证路径。
var defaultCredentialPaths = map[string]string{
	"kimi":   "~/.kimi/credentials/kimi-code.json",
	"gemini": "~/.config/gcloud/application_default_credentials.json",
}

// 预置 CLI Provider 配置模板。
var cliProviderTemplates = map[string]struct {
	Name    string
	APIHost string
	ModelID string
}{
	"kimi":   {Name: "Kimi (CLI)", APIHost: "https://api.moonshot.cn", ModelID: "moonshot-v1-8k"},
	"gemini": {Name: "Gemini (CLI)", APIHost: "https://generativelanguage.googleapis.com/v1beta/openai/", ModelID: "gemini-1.5-flash"},
}

// CLITokenService 实现 CLI Token 自动发现、读取、验证与 ProviderConfig 构建。
type CLITokenService struct {
	httpClient *http.Client
}

// NewCLITokenService 创建默认的 CLI Token 服务。
func NewCLITokenService() *CLITokenService {
	return &CLITokenService{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// NewCLITokenServiceWithClient 使用自定义 HTTP 客户端创建服务（主要用于测试注入）。
func NewCLITokenServiceWithClient(client *http.Client) *CLITokenService {
	return &CLITokenService{httpClient: client}
}

// Detect 检测指定类型的 CLI 是否安装并登录。
//
// 检测逻辑：
//  1. 查找默认凭证路径是否存在
//  2. 文件存在则尝试读取并解析 token
//  3. 返回解析结果（Detected / LoggedIn / Error）
func (s *CLITokenService) Detect(providerType string) (*CLIDetectResult, error) {
	credPath, ok := defaultCredentialPaths[providerType]
	if !ok {
		return nil, fmt.Errorf("unsupported cli provider type: %s", providerType)
	}

	result := &CLIDetectResult{
		ProviderType:   providerType,
		CredentialPath: credPath,
	}

	expanded := expandHome(credPath)
	info, err := os.Stat(expanded)
	if err != nil {
		if os.IsNotExist(err) {
			result.Detected = false
			return result, nil
		}
		result.Error = fmt.Sprintf("failed to stat credential file: %v", err)
		return result, nil
	}

	result.Detected = true

	if info.IsDir() {
		result.Error = "credential path is a directory"
		return result, nil
	}
	if info.Size() == 0 {
		result.Error = "credential file is empty"
		return result, nil
	}

	token, hint, err := models.ReadCLITokenFromFile(expanded)
	if err != nil {
		result.Error = fmt.Sprintf("failed to parse credential: %v", err)
		return result, nil
	}

	// refresh_token 也算"已登录"（有凭证信息），只是需要后续 TASK-045 刷新
	result.LoggedIn = token != "" || hint == "refresh_token"

	return result, nil
}

// ReadToken 从指定路径读取 CLI token。
//
// 若 credentialPath 为空，使用 providerType 对应的默认路径。
// 返回值 needsRefresh 为 true 时表示读取到的是 refresh_token，需要调用 TokenRefreshService 刷新。
func (s *CLITokenService) ReadToken(providerType, credentialPath string) (token string, needsRefresh bool, err error) {
	if credentialPath == "" {
		if path, ok := defaultCredentialPaths[providerType]; ok {
			credentialPath = path
		} else {
			return "", false, fmt.Errorf("unsupported cli provider type: %s", providerType)
		}
	}

	token, hint, err := models.ReadCLITokenFromFile(credentialPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to read %s cli token: %w", providerType, err)
	}

	if hint == "refresh_token" {
		return token, true, nil
	}

	return token, false, nil
}

// ValidateToken 调用厂商 /v1/models 端点验证 token 有效性。
//
// HTTP 200 视为有效；401 视为无效（token 过期或错误）；
// 其他状态码返回错误供上层判断。
func (s *CLITokenService) ValidateToken(ctx context.Context, apiHost, token string) (bool, error) {
	if apiHost == "" {
		return false, fmt.Errorf("api_host is required")
	}
	if token == "" {
		return false, fmt.Errorf("token is required")
	}

	url := strings.TrimSuffix(apiHost, "/") + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return false, nil
	}
	return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// BuildProviderConfig 根据检测到的 CLI 自动构建完整 ProviderConfig。
//
// AuthMethod 固定为 cli_token，AuthParams.CLICredentialPath 使用默认路径。
// modelID 为空时使用预置默认模型。
func (s *CLITokenService) BuildProviderConfig(providerType, modelID string) (*models.ProviderConfig, error) {
	tmpl, ok := cliProviderTemplates[providerType]
	if !ok {
		return nil, fmt.Errorf("unsupported cli provider type: %s", providerType)
	}

	credPath, ok := defaultCredentialPaths[providerType]
	if !ok {
		return nil, fmt.Errorf("no default credential path for %s", providerType)
	}

	mID := modelID
	if mID == "" {
		mID = tmpl.ModelID
	}

	now := time.Now()
	return &models.ProviderConfig{
		ID:          fmt.Sprintf("cli-%s-%d", providerType, now.Unix()),
		Name:        tmpl.Name,
		APIHost:     tmpl.APIHost,
		APIKey:      "",
		ModelID:     mID,
		Temperature: 0.7,
		TimeoutMs:   30000,
		MaxRetries:  3,
		GroupName:   "CLI",
		Enabled:     true,
		AuthMethod:  models.AuthMethodCLIToken,
		AuthParams: models.AuthParams{
			CLICredentialPath: credPath,
		},
		CreatedAt: now.UnixMilli(),
		UpdatedAt: now.UnixMilli(),
	}, nil
}

// expandHome 将路径中的 ~ 展开为用户主目录。
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
