// Package auth 实现 OAuth 2.0 Device Authorization Grant（RFC 8628）。
// 供用户在没有 API Key 的情况下，通过浏览器授权获取 access_token。
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/wire"
	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/pkg/models"
)

// DeviceAuthResponse 表示设备授权端点返回的数据。
type DeviceAuthResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"` // 秒
	Interval        int    `json:"interval"`   // 秒，建议轮询间隔
}

// DeviceTokenResponse 表示 token 端点返回的数据。
type DeviceTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// DeviceFlowStartResult 表示 StartFlow 的返回结果。
type DeviceFlowStartResult struct {
	UserCode        string
	VerificationURI string
	DeviceCode      string
	ExpiresIn       int
	Interval        int
}

// DeviceFlowStatus 表示当前轮询状态。
type DeviceFlowStatus string

const (
	// DeviceFlowStatusPending 表示等待用户授权。
	DeviceFlowStatusPending DeviceFlowStatus = "pending"
	// DeviceFlowStatusSlowDown 表示轮询间隔被厂商要求延长。
	DeviceFlowStatusSlowDown DeviceFlowStatus = "slow_down"
	// DeviceFlowStatusSuccess 表示成功获取 token。
	DeviceFlowStatusSuccess DeviceFlowStatus = "success"
	// DeviceFlowStatusError 表示发生错误。
	DeviceFlowStatusError DeviceFlowStatus = "error"
	// DeviceFlowStatusCancelled 表示用户取消。
	DeviceFlowStatusCancelled DeviceFlowStatus = "cancelled"
)

// DeviceFlowStatusResult 表示查询轮询状态的结果。
type DeviceFlowStatusResult struct {
	DeviceCode   string
	ProviderType string
	Status       DeviceFlowStatus
	Error        string
}

// deviceFlowSession 表示一个进行中的 Device Flow 会话。
type deviceFlowSession struct {
	providerType string
	deviceCode   string
	clientID     string
	tokenURL     string
	interval     int       // 当前轮询间隔（秒）
	expiresAt    time.Time // 设备码过期时间
	cancel       context.CancelFunc
	done         chan struct{} // 轮询结束信号
	result       *DeviceTokenResponse
	err          error
	status       DeviceFlowStatus
	mu           sync.Mutex
}

// 厂商 Device Flow 端点预置配置。
// client_id 留空表示需要调用方提供（如 Gemini）。
var deviceFlowConfigs = map[string]struct {
	DeviceAuthURL string
	TokenURL      string
	ClientID      string
	Scope         string
}{
	"kimi": {
		DeviceAuthURL: "https://api.moonshot.cn/v1/token/device",
		TokenURL:      "https://api.moonshot.cn/v1/token",
		ClientID:      "",
		Scope:         "api",
	},
	"gemini": {
		DeviceAuthURL: "https://oauth2.googleapis.com/device/code",
		TokenURL:      "https://oauth2.googleapis.com/token",
		ClientID:      "", // Google Device Flow 需要调用方提供 OAuth client_id
		Scope:         "https://www.googleapis.com/auth/cloud-platform",
	},
}

// OAuthDeviceFlowService 实现 OAuth 2.0 Device Flow。
type OAuthDeviceFlowService struct {
	httpClient *http.Client
	sessions   map[string]*deviceFlowSession // deviceCode -> session
	mu         sync.Mutex
	store      port.ProviderStore
	refreshSvc *TokenRefreshService

	// 事件回调（由 WailsApp 注入）
	onSuccess  func(deviceCode string, providerType string, cfg *models.ProviderConfig)
	onError    func(deviceCode string, providerType string, err error)
	onPending  func(deviceCode string, providerType string)
	onSlowDown func(deviceCode string, providerType string, newInterval int)
}

// NewOAuthDeviceFlowService 创建 Device Flow 服务。
func NewOAuthDeviceFlowService(store port.ProviderStore, refreshSvc *TokenRefreshService) *OAuthDeviceFlowService {
	return NewOAuthDeviceFlowServiceWithClient(store, refreshSvc, &http.Client{Timeout: 15 * time.Second})
}

// NewOAuthDeviceFlowServiceBare 创建不带 TokenRefreshService 的 Device Flow 服务（供 Wire 注入）。
// refreshSvc 在运行时可由调用方通过 SetRefreshService 设置。
func NewOAuthDeviceFlowServiceBare(store port.ProviderStore) *OAuthDeviceFlowService {
	return NewOAuthDeviceFlowServiceWithClient(store, nil, &http.Client{Timeout: 15 * time.Second})
}

// NewOAuthDeviceFlowServiceWithClient 使用自定义 HTTP 客户端创建服务（主要用于测试）。
func NewOAuthDeviceFlowServiceWithClient(store port.ProviderStore, refreshSvc *TokenRefreshService, client *http.Client) *OAuthDeviceFlowService {
	return &OAuthDeviceFlowService{
		httpClient: client,
		sessions:   make(map[string]*deviceFlowSession),
		store:      store,
		refreshSvc: refreshSvc,
	}
}

// SetRefreshService 设置 TokenRefreshService（供 WailsApp 在 Wire 注入后调用）。
func (s *OAuthDeviceFlowService) SetRefreshService(refreshSvc *TokenRefreshService) {
	s.refreshSvc = refreshSvc
}

// SetCallbacks 设置事件回调。
func (s *OAuthDeviceFlowService) SetCallbacks(
	onSuccess func(deviceCode string, providerType string, cfg *models.ProviderConfig),
	onError func(deviceCode string, providerType string, err error),
	onPending func(deviceCode string, providerType string),
	onSlowDown func(deviceCode string, providerType string, newInterval int),
) {
	s.onSuccess = onSuccess
	s.onError = onError
	s.onPending = onPending
	s.onSlowDown = onSlowDown
}

// StartFlow 对指定厂商启动 Device Flow。
// 返回用户码和验证 URL，供前端展示；后端立即启动后台轮询。
func (s *OAuthDeviceFlowService) StartFlow(providerType string) (*DeviceFlowStartResult, error) {
	cfg, ok := deviceFlowConfigs[providerType]
	if !ok {
		return nil, fmt.Errorf("unsupported provider type for device flow: %s", providerType)
	}

	if cfg.ClientID == "" {
		return nil, fmt.Errorf("provider %s requires OAuth client_id to be configured for device flow", providerType)
	}

	// 1. 请求设备码
	deviceAuth, err := s.requestDeviceCode(cfg.DeviceAuthURL, cfg.ClientID, cfg.Scope)
	if err != nil {
		return nil, fmt.Errorf("failed to request device code for %s: %w", providerType, err)
	}

	// 2. 创建 session
	ctx, cancel := context.WithCancel(context.Background())
	interval := deviceAuth.Interval
	if interval <= 0 {
		interval = 5 // RFC 8628 默认最小间隔
	}
	expiresIn := deviceAuth.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 600 // 默认 10 分钟
	}

	session := &deviceFlowSession{
		providerType: providerType,
		deviceCode:   deviceAuth.DeviceCode,
		clientID:     cfg.ClientID,
		tokenURL:     cfg.TokenURL,
		interval:     interval,
		expiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
		cancel:       cancel,
		done:         make(chan struct{}),
		status:       DeviceFlowStatusPending,
	}

	s.mu.Lock()
	s.sessions[deviceAuth.DeviceCode] = session
	s.mu.Unlock()

	// 3. 启动后台轮询
	go s.pollLoop(ctx, session)

	return &DeviceFlowStartResult{
		UserCode:        deviceAuth.UserCode,
		VerificationURI: deviceAuth.VerificationURI,
		DeviceCode:      deviceAuth.DeviceCode,
		ExpiresIn:       expiresIn,
		Interval:        interval,
	}, nil
}

// CancelFlow 取消指定 deviceCode 的轮询。
func (s *OAuthDeviceFlowService) CancelFlow(deviceCode string) {
	s.mu.Lock()
	session, ok := s.sessions[deviceCode]
	s.mu.Unlock()

	if !ok {
		return
	}

	session.cancel()
	<-session.done

	s.mu.Lock()
	delete(s.sessions, deviceCode)
	s.mu.Unlock()
}

// GetStatus 查询指定 deviceCode 的当前状态。
func (s *OAuthDeviceFlowService) GetStatus(deviceCode string) *DeviceFlowStatusResult {
	s.mu.Lock()
	session, ok := s.sessions[deviceCode]
	s.mu.Unlock()

	if !ok {
		return nil
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	result := &DeviceFlowStatusResult{
		DeviceCode:   session.deviceCode,
		ProviderType: session.providerType,
		Status:       session.status,
	}
	if session.err != nil {
		result.Error = session.err.Error()
	}
	return result
}

// Shutdown 取消所有进行中的轮询会话。
func (s *OAuthDeviceFlowService) Shutdown() {
	s.mu.Lock()
	codes := make([]string, 0, len(s.sessions))
	for code := range s.sessions {
		codes = append(codes, code)
	}
	s.mu.Unlock()

	for _, code := range codes {
		s.CancelFlow(code)
	}
}

// requestDeviceCode 向设备授权端点请求设备码。
func (s *OAuthDeviceFlowService) requestDeviceCode(deviceAuthURL, clientID, scope string) (*DeviceAuthResponse, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	if scope != "" {
		data.Set("scope", scope)
	}

	req, err := http.NewRequest(http.MethodPost, deviceAuthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create device auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device auth endpoint returned status %d", resp.StatusCode)
	}

	var result DeviceAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode device auth response: %w", err)
	}

	if result.DeviceCode == "" {
		return nil, fmt.Errorf("device auth response missing device_code")
	}

	return &result, nil
}

// pollLoop 后台轮询 token 端点。
func (s *OAuthDeviceFlowService) pollLoop(ctx context.Context, session *deviceFlowSession) {
	defer close(session.done)

	ticker := time.NewTicker(time.Duration(session.interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			session.mu.Lock()
			session.status = DeviceFlowStatusCancelled
			session.mu.Unlock()
			return

		case <-ticker.C:
			// 检查设备码是否已过期
			if time.Now().After(session.expiresAt) {
				session.mu.Lock()
				session.status = DeviceFlowStatusError
				session.err = fmt.Errorf("device code expired")
				session.mu.Unlock()
				s.emitError(session)
				return
			}

			tokenResp, err := s.pollTokenEndpoint(session)
			if err != nil {
				// 根据错误类型处理
				switch {
				case strings.Contains(err.Error(), "authorization_pending"):
					s.emitPending(session)
					continue

				case strings.Contains(err.Error(), "slow_down"):
					// 延长轮询间隔（按服务器要求或默认翻倍）
					session.mu.Lock()
					session.interval += 5
					newInterval := session.interval
					session.status = DeviceFlowStatusSlowDown
					session.mu.Unlock()
					ticker.Reset(time.Duration(newInterval) * time.Second)
					s.emitSlowDown(session, newInterval)
					continue

				case strings.Contains(err.Error(), "access_denied"):
					session.mu.Lock()
					session.status = DeviceFlowStatusError
					session.err = fmt.Errorf("user denied authorization")
					session.mu.Unlock()
					s.emitError(session)
					return

				case strings.Contains(err.Error(), "expired_token"):
					session.mu.Lock()
					session.status = DeviceFlowStatusError
					session.err = fmt.Errorf("device code expired")
					session.mu.Unlock()
					s.emitError(session)
					return

				default:
					// 其他网络或服务器错误，继续轮询（但总超时由 expiresAt 控制）
					s.emitPending(session)
					continue
				}
			}

			// 成功获取 token
			session.mu.Lock()
			session.result = tokenResp
			session.status = DeviceFlowStatusSuccess
			session.mu.Unlock()

			// 保存 ProviderConfig 并注册自动刷新
			cfg, saveErr := s.saveProviderConfig(session, tokenResp)
			if saveErr != nil {
				session.mu.Lock()
				session.err = saveErr
				session.status = DeviceFlowStatusError
				session.mu.Unlock()
				s.emitError(session)
				return
			}

			s.emitSuccess(session, cfg)
			return
		}
	}
}

// pollTokenEndpoint 执行一次 token 端点轮询。
func (s *OAuthDeviceFlowService) pollTokenEndpoint(session *deviceFlowSession) (*DeviceTokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", session.clientID)
	data.Set("device_code", session.deviceCode)
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	req, err := http.NewRequest(http.MethodPost, session.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var result DeviceTokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode token response: %w", err)
		}
		if result.AccessToken == "" {
			return nil, fmt.Errorf("token response missing access_token")
		}
		return &result, nil
	}

	// 解析错误响应
	var errBody struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&errBody)

	if errBody.Error != "" {
		return nil, fmt.Errorf("token endpoint error: %s", errBody.Error)
	}

	return nil, fmt.Errorf("token endpoint returned status %d", resp.StatusCode)
}

// saveProviderConfig 将获取到的 token 保存为 ProviderConfig。
func (s *OAuthDeviceFlowService) saveProviderConfig(session *deviceFlowSession, tokenResp *DeviceTokenResponse) (*models.ProviderConfig, error) {
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Unix()
	if tokenResp.ExpiresIn == 0 {
		expiresAt = time.Now().Add(1 * time.Hour).Unix()
	}

	// 根据 providerType 推断 API Host 和默认模型
	apiHost, modelID, name := s.inferProviderInfo(session.providerType)

	now := time.Now()
	cfg := &models.ProviderConfig{
		ID:          fmt.Sprintf("oauth-%s-%d", session.providerType, now.Unix()),
		Name:        name,
		APIHost:     apiHost,
		APIKey:      "",
		ModelID:     modelID,
		Temperature: 0.7,
		Timeout:     30 * time.Second,
		MaxRetries:  3,
		GroupName:   "OAuth",
		Enabled:     true,
		AuthMethod:  models.AuthMethodOAuthDevice,
		AuthParams: models.AuthParams{
			OAuthClientID:     session.clientID,
			OAuthTokenURL:     session.tokenURL,
			OAuthAccessToken:  tokenResp.AccessToken,
			OAuthRefreshToken: tokenResp.RefreshToken,
			OAuthExpiresAt:    expiresAt,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.store.Create(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to save provider config: %w", err)
	}

	// 注册自动刷新
	if s.refreshSvc != nil && tokenResp.RefreshToken != "" {
		_ = s.refreshSvc.ScheduleAutoRefresh(cfg.ID)
	}

	return cfg, nil
}

// inferProviderInfo 根据 providerType 推断 API Host、默认模型和显示名称。
func (s *OAuthDeviceFlowService) inferProviderInfo(providerType string) (apiHost, modelID, name string) {
	switch providerType {
	case "kimi":
		return "https://api.moonshot.cn", "moonshot-v1-8k", "Kimi (OAuth)"
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta/openai/", "gemini-1.5-flash", "Gemini (OAuth)"
	default:
		return "", "", fmt.Sprintf("OAuth %s", providerType)
	}
}

// emitSuccess 触发成功回调。
func (s *OAuthDeviceFlowService) emitSuccess(session *deviceFlowSession, cfg *models.ProviderConfig) {
	if s.onSuccess != nil {
		s.onSuccess(session.deviceCode, session.providerType, cfg)
	}
}

// emitError 触发错误回调。
func (s *OAuthDeviceFlowService) emitError(session *deviceFlowSession) {
	if s.onError != nil {
		s.onError(session.deviceCode, session.providerType, session.err)
	}
}

// emitPending 触发等待中回调。
func (s *OAuthDeviceFlowService) emitPending(session *deviceFlowSession) {
	if s.onPending != nil {
		s.onPending(session.deviceCode, session.providerType)
	}
}

// emitSlowDown 触发 slow_down 回调。
func (s *OAuthDeviceFlowService) emitSlowDown(session *deviceFlowSession, newInterval int) {
	if s.onSlowDown != nil {
		s.onSlowDown(session.deviceCode, session.providerType, newInterval)
	}
}

// OAuthDeviceFlowProviderSet 是 OAuthDeviceFlowService 的 Wire ProviderSet。
var OAuthDeviceFlowProviderSet = wire.NewSet(NewOAuthDeviceFlowServiceBare)
