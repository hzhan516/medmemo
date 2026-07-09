// Package auth 实现 Token 自动刷新服务。
// 负责调用厂商 OAuth2 token endpoint 刷新 access_token，并管理后台自动调度。
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/wire"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/pkg/models"
)

// RefreshResult 表示一次刷新操作的结果。
type RefreshResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at"` // Unix 秒时间戳
	ExpiresIn    int    `json:"expires_in"` // 有效期（秒）
	TokenType    string `json:"token_type"` // 通常为 "Bearer"
}

// 厂商 OAuth2 token endpoint 预置映射。
var providerTokenEndpoints = map[string]string{
	"kimi":      "https://api.moonshot.cn/v1/token",
	"gemini":    "https://oauth2.googleapis.com/token",
	"microsoft": "https://login.microsoftonline.com/common/oauth2/v2.0/token",
	"github":    "https://github.com/login/oauth/access_token",
}

// TokenRefreshService 实现 Token 自动刷新与调度。
type TokenRefreshService struct {
	store      port.ProviderStore
	httpClient *http.Client
	timers     map[string]*time.Timer // providerID -> timer
	mu         sync.Mutex
	refreshMu  sync.Map // providerID -> *sync.Mutex，防止同一 provider 并发刷新
	onDegraded func(providerID, reason string)
}

// TokenRefreshOption 配置 TokenRefreshService 的可选参数。
type TokenRefreshOption func(*TokenRefreshService)

// WithTokenRefreshHTTPClient 使用自定义 HTTP 客户端。
func WithTokenRefreshHTTPClient(client *http.Client) TokenRefreshOption {
	return func(s *TokenRefreshService) {
		s.httpClient = client
	}
}

// WithTokenRefreshOnDegraded 设置认证降级回调。
func WithTokenRefreshOnDegraded(cb func(providerID, reason string)) TokenRefreshOption {
	return func(s *TokenRefreshService) {
		s.onDegraded = cb
	}
}

// NewTokenRefreshService 创建 Token 刷新服务。
// 默认 HTTP 客户端超时 15 秒；可通过 WithTokenRefreshHTTPClient 覆盖。
// 降级回调可通过 WithTokenRefreshOnDegraded 设置，供 WailsApp 在获取 runtime context 后注入。
func NewTokenRefreshService(store port.ProviderStore, opts ...TokenRefreshOption) *TokenRefreshService {
	s := &TokenRefreshService{
		store:      store,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		timers:     make(map[string]*time.Timer),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SetOnDegraded 设置降级回调（供 WailsApp 在获取 runtime context 后调用）。
func (s *TokenRefreshService) SetOnDegraded(cb func(providerID, reason string)) {
	s.onDegraded = cb
}

// Refresh 对指定 Provider 执行一次手动 token 刷新。
func (s *TokenRefreshService) Refresh(providerID string) (*RefreshResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	p, err := s.store.Get(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider %s: %w", providerID, err)
	}

	return s.RefreshProvider(p)
}

// RefreshProvider 对给定的 ProviderConfig 执行 token 刷新。
func (s *TokenRefreshService) RefreshProvider(p *models.ProviderConfig) (*RefreshResult, error) {
	if p.AuthMethod != models.AuthMethodCLIToken && p.AuthMethod != models.AuthMethodOAuthDevice {
		return nil, fmt.Errorf("provider %s auth method %s does not support refresh", p.ID, p.AuthMethod)
	}

	// 获取 provider 级别的刷新锁，防止并发刷新
	muVal, _ := s.refreshMu.LoadOrStore(p.ID, &sync.Mutex{})
	mu, ok := muVal.(*sync.Mutex)
	if !ok {
		return nil, fmt.Errorf("refreshMu stored non-*sync.Mutex value for provider %s", p.ID)
	}
	mu.Lock()
	defer mu.Unlock()

	creds, err := models.ReadCLICredentials(p.AuthParams.CLICredentialPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials for provider %s: %w", p.ID, err)
	}

	// 如果是 OAuthDevice，优先使用 AuthParams 中的 refresh_token
	if p.AuthMethod == models.AuthMethodOAuthDevice && p.AuthParams.OAuthRefreshToken != "" {
		creds.RefreshToken = p.AuthParams.OAuthRefreshToken
		creds.ClientID = p.AuthParams.OAuthClientID
		// client_secret 对于 device flow 可能不需要
	}

	if creds.RefreshToken == "" {
		return nil, fmt.Errorf("provider %s has no refresh_token available", p.ID)
	}

	providerType := inferProviderType(p)
	tokenURL := providerTokenEndpoints[providerType]
	if tokenURL == "" {
		return nil, fmt.Errorf("no token endpoint configured for provider type %s", providerType)
	}

	result, err := s.doRefresh(providerType, creds, tokenURL)
	if err != nil {
		// 4xx 错误触发降级
		if strings.Contains(err.Error(), "status 4") {
			s.handleDegradation(p, err.Error())
		}
		return nil, fmt.Errorf("failed to refresh token for provider %s: %w", p.ID, err)
	}

	// 更新 ProviderConfig 的 AuthParams
	p.AuthParams.OAuthAccessToken = result.AccessToken
	p.AuthParams.OAuthRefreshToken = result.RefreshToken
	p.AuthParams.OAuthExpiresAt = result.ExpiresAt
	p.UpdatedAt = time.Now().UnixMilli()

	// 持久化到数据库
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.store.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to persist refreshed token for provider %s: %w", p.ID, err)
	}

	// 尝试写回 CLI 凭证文件（保持 CLI 工具兼容性）
	if err := s.writeBackCredentials(p, creds, result); err != nil {
		// 文件写回失败不是致命错误，已持久化到数据库
		_ = err // 静默处理
	}

	return result, nil
}

// ScheduleAutoRefresh 为指定 Provider 安排自动刷新调度。
// 根据 AuthParams.OAuthExpiresAt 计算下次刷新时间（过期前 5 分钟）。
func (s *TokenRefreshService) ScheduleAutoRefresh(providerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	p, err := s.store.Get(ctx, providerID)
	if err != nil {
		return fmt.Errorf("failed to get provider %s: %w", providerID, err)
	}

	s.scheduleForProvider(p)
	return nil
}

// scheduleForProvider 为给定的 ProviderConfig 创建 timer。
func (s *TokenRefreshService) scheduleForProvider(p *models.ProviderConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 取消已有的调度
	if t, ok := s.timers[p.ID]; ok {
		t.Stop()
		delete(s.timers, p.ID)
	}

	now := time.Now().Unix()
	expiresAt := p.AuthParams.OAuthExpiresAt

	// 如果没有过期时间或不支持刷新，不调度
	if expiresAt == 0 || p.AuthMethod != models.AuthMethodCLIToken && p.AuthMethod != models.AuthMethodOAuthDevice {
		return
	}

	// 过期前 5 分钟触发刷新
	triggerAt := expiresAt - 300
	var delay time.Duration
	if triggerAt <= now {
		// 已过期或即将过期，立即刷新
		delay = 0
	} else {
		delay = time.Until(time.Unix(triggerAt, 0))
	}

	s.timers[p.ID] = time.AfterFunc(delay, func() {
		_, err := s.Refresh(p.ID)
		if err != nil {
			// 刷新失败，不再重新调度（等待下次外部触发或手动刷新）
			return
		}
		// 刷新成功，重新调度
		_ = s.ScheduleAutoRefresh(p.ID)
	})
}

// CancelAutoRefresh 取消指定 Provider 的自动刷新调度。
func (s *TokenRefreshService) CancelAutoRefresh(providerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t, ok := s.timers[providerID]; ok {
		t.Stop()
		delete(s.timers, providerID)
	}
}

// Shutdown 停止所有自动刷新调度器。
func (s *TokenRefreshService) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, t := range s.timers {
		t.Stop()
		delete(s.timers, id)
	}
}

// doRefresh 执行实际的 HTTP POST 刷新请求。
func (s *TokenRefreshService) doRefresh(providerType string, creds *models.CLICredentials, tokenURL string) (*RefreshResult, error) {
	if creds.ClientID == "" && providerType == "gemini" {
		return nil, fmt.Errorf("missing client_id for %s refresh", providerType)
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", creds.RefreshToken)
	if creds.ClientID != "" {
		data.Set("client_id", creds.ClientID)
	}
	if creds.ClientSecret != "" {
		data.Set("client_secret", creds.ClientSecret)
	}

	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return nil, fmt.Errorf("refresh endpoint returned status %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh endpoint returned unexpected status %d", resp.StatusCode)
	}

	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to decode refresh response: %w", err)
	}

	if body.AccessToken == "" {
		return nil, fmt.Errorf("refresh response missing access_token")
	}

	expiresAt := time.Now().Add(time.Duration(body.ExpiresIn) * time.Second).Unix()
	if body.ExpiresIn == 0 {
		// 默认 1 小时有效期
		expiresAt = time.Now().Add(1 * time.Hour).Unix()
	}

	return &RefreshResult{
		AccessToken:  body.AccessToken,
		RefreshToken: body.RefreshToken,
		ExpiresAt:    expiresAt,
		ExpiresIn:    body.ExpiresIn,
		TokenType:    body.TokenType,
	}, nil
}

// handleDegradation 处理刷新失败降级。
func (s *TokenRefreshService) handleDegradation(p *models.ProviderConfig, reason string) {
	p.AuthParams.OAuthAccessToken = ""
	p.Enabled = false
	p.UpdatedAt = time.Now().UnixMilli()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.store.Update(ctx, p) // 降级时静默处理持久化错误

	if s.onDegraded != nil {
		s.onDegraded(p.ID, reason)
	}
}

// writeBackCredentials 将刷新后的 token 写回 CLI 凭证文件。
func (s *TokenRefreshService) writeBackCredentials(p *models.ProviderConfig, creds *models.CLICredentials, result *RefreshResult) error {
	path := models.ExpandPath(p.AuthParams.CLICredentialPath)
	if path == "" {
		return fmt.Errorf("credential path is empty")
	}

	providerType := inferProviderType(p)

	switch providerType {
	case "kimi":
		return s.writeKimiCredentials(path, creds, result)
	case "gemini":
		return s.writeGeminiCredentials(path, creds, result)
	default:
		return fmt.Errorf("unsupported provider type for write-back: %s", providerType)
	}
}

// writeKimiCredentials 更新 Kimi 凭证文件。
func (s *TokenRefreshService) writeKimiCredentials(path string, _ *models.CLICredentials, result *RefreshResult) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read kimi credential file: %w", err)
	}

	var file map[string]any
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to parse kimi credential file: %w", err)
	}

	file["access_token"] = result.AccessToken
	if result.RefreshToken != "" {
		file["refresh_token"] = result.RefreshToken
	}
	file["expires_at"] = result.ExpiresAt

	updated, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated kimi credentials: %w", err)
	}

	if err := os.WriteFile(path, updated, 0600); err != nil {
		return fmt.Errorf("failed to write kimi credential file: %w", err)
	}
	return nil
}

// writeGeminiCredentials 为 Gemini 写 companion 缓存文件。
func (s *TokenRefreshService) writeGeminiCredentials(adcPath string, creds *models.CLICredentials, result *RefreshResult) error {
	// Gemini 的 gcloud ADC 文件不原生支持 access_token 字段，
	// 将 access_token 缓存到同级目录的 companion 文件中。
	dir := filepath.Dir(adcPath)
	cachePath := filepath.Join(dir, "medmemo_adc_cache.json")

	cache := map[string]any{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_at":    result.ExpiresAt,
		"client_id":     creds.ClientID,
		"client_secret": creds.ClientSecret,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal gemini cache: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write gemini cache file: %w", err)
	}
	return nil
}

// inferProviderType 根据 ProviderConfig 推断 provider 类型。
func inferProviderType(p *models.ProviderConfig) string {
	host := strings.ToLower(p.APIHost)
	switch {
	case strings.Contains(host, "moonshot"):
		return "kimi"
	case strings.Contains(host, "google") || strings.Contains(host, "gemini"):
		return "gemini"
	case strings.Contains(host, "microsoft") || strings.Contains(host, "azure") || strings.Contains(host, "windows"):
		return "microsoft"
	case strings.Contains(host, "github"):
		return "github"
	default:
		return ""
	}
}

// TokenRefreshProviderSet 是 TokenRefreshService 的 Wire ProviderSet。
// 使用 NewTokenRefreshService 无回调版本注入，WailsApp 在 Startup 中通过 SetOnDegraded 注入回调。
var TokenRefreshProviderSet = wire.NewSet(
	NewTokenRefreshService,
	wire.Value([]TokenRefreshOption{}),
)
