// Package main 是 MedMemo 应用入口。
// WailsApp 暴露前端可调用的绑定方法集，供 Wire 注入后绑定到 Wails 运行时。
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/hzhan516/medmemo/internal/adapters/ai"
	"github.com/hzhan516/medmemo/internal/adapters/auth"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/application/updater"
	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/hzhan516/medmemo/internal/infrastructure/secret"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/hzhan516/medmemo/pkg/resourcepath"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// WailsApp 暴露给前端 Wails 绑定的方法集。
type WailsApp struct {
	ctx              context.Context
	chatOrchestrator *usecase.ChatOrchestrator
	memoryRetriever  *usecase.MemoryRetriever
	config           *models.AppConfig
	convRepo         port.ConversationRepository
	msgRepo          port.MessageRepository
	disclaimerRepo   port.DisclaimerRepository
	providerStore    port.ProviderStore
	healthChecker    port.HealthChecker
	titleGen         *usecase.TitleGenerator
	updaterSvc       *updater.Service
	secretStore      secret.Store
	tokenRefreshSvc  *auth.TokenRefreshService
	deviceFlowSvc    *auth.OAuthDeviceFlowService
	streamMu         sync.Mutex
	activeStreams    map[string]context.CancelFunc

	// 本地回调服务器管理（每个 Device Flow 会话对应一个）
	callbackServers map[string]*auth.LocalCallbackServer
	callbackMu      sync.Mutex

	// Ollama 操作互斥锁，防止并发启动/下载冲突
	ollamaMu sync.Mutex

	// 记忆管理仓库（TASK-060）
	factRepo repository.FactRepository

	// 审计日志仓库（v1.1 DoD A3）
	auditLogRepo repository.AuditLogRepository

	// 原始对话仓库（记忆归档）
	dialogueRepo repository.RawDialogueRepository

	// 语义嵌入服务与仓库（向量索引）
	embeddingSvc  port.EmbeddingService
	embeddingRepo repository.EmbeddingRepository

	// v1.1.4: embedding 版本迁移器与状态追踪
	migrator       *usecase.EmbeddingMigrator
	migrationState *usecase.MigrationState
	onnxReady      chan struct{}
	onnxOnce       sync.Once
}

// NewWailsApp 构造函数，供 Wire 调用。
func NewWailsApp(
	chat *usecase.ChatOrchestrator,
	mem *usecase.MemoryRetriever,
	cfg *models.AppConfig,
	convRepo port.ConversationRepository,
	msgRepo port.MessageRepository,
	disclaimerRepo port.DisclaimerRepository,
	providerStore port.ProviderStore,
	healthChecker port.HealthChecker,
	titleGen *usecase.TitleGenerator,
	updaterSvc *updater.Service,
	secretStore secret.Store,
	tokenRefreshSvc *auth.TokenRefreshService,
	deviceFlowSvc *auth.OAuthDeviceFlowService,
	factRepo repository.FactRepository,
	auditLogRepo repository.AuditLogRepository,
	dialogueRepo repository.RawDialogueRepository,
	embeddingSvc port.EmbeddingService,
	embeddingRepo repository.EmbeddingRepository,
	migrator *usecase.EmbeddingMigrator,
	migrationState *usecase.MigrationState,
) *WailsApp {
	return &WailsApp{
		chatOrchestrator: chat,
		memoryRetriever:  mem,
		config:           cfg,
		convRepo:         convRepo,
		msgRepo:          msgRepo,
		disclaimerRepo:   disclaimerRepo,
		providerStore:    providerStore,
		healthChecker:    healthChecker,
		titleGen:         titleGen,
		updaterSvc:       updaterSvc,
		secretStore:      secretStore,
		tokenRefreshSvc:  tokenRefreshSvc,
		deviceFlowSvc:    deviceFlowSvc,
		factRepo:         factRepo,
		auditLogRepo:     auditLogRepo,
		dialogueRepo:     dialogueRepo,
		embeddingSvc:     embeddingSvc,
		embeddingRepo:    embeddingRepo,
		migrator:         migrator,
		migrationState:   migrationState,
		onnxReady:        make(chan struct{}),
		callbackServers:  make(map[string]*auth.LocalCallbackServer),
		activeStreams:    make(map[string]context.CancelFunc),
	}
}

// ModelInfo 模型信息。
type ModelInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

// GetModels 获取可用模型列表。
func (a *WailsApp) GetModels() ([]ModelInfo, error) {
	return []ModelInfo{
		{ID: "kimi-lite", Name: "Kimi Lite", Provider: "kimi"},
		{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Provider: "openai"},
		{ID: "qwen-turbo", Name: "通义千问 Turbo", Provider: "qwen"},
		{ID: "llama3.1-8b", Name: "Llama 3.1 8B (本地)", Provider: "ollama"},
	}, nil
}

// --- Onboarding 向导相关绑定方法 ---

// SaveAPIKey 将指定提供商的 API Key 安全保存到系统密钥环。
// provider 对应 models.ProviderType 的字符串形式，apiKey 为原始密钥值。
func (a *WailsApp) SaveAPIKey(provider string, apiKey string) error {
	if provider == "" {
		return fmt.Errorf("provider cannot be empty")
	}
	if apiKey == "" {
		return fmt.Errorf("api key cannot be empty")
	}
	key := fmt.Sprintf("apikey:%s", provider)
	if err := a.secretStore.Set(key, []byte(apiKey)); err != nil {
		return fmt.Errorf("failed to save API key for provider %s: %w", provider, err)
	}
	return nil
}

// HasAPIKey 检测指定提供商是否已配置 API Key。
func (a *WailsApp) HasAPIKey(provider string) (bool, error) {
	if provider == "" {
		return false, fmt.Errorf("provider cannot be empty")
	}
	key := fmt.Sprintf("apikey:%s", provider)
	_, err := a.secretStore.Get(key)
	if err != nil {
		// 密钥不存在或读取失败，均视为未配置
		return false, nil
	}
	return true, nil
}

// TestAPIKeyResult 表示 API Key 验证结果。
type TestAPIKeyResult struct {
	Valid   bool     `json:"valid"`
	Message string   `json:"message"`
	Models  []string `json:"models,omitempty"`
}

// defaultAPIHosts 定义各厂商的默认 API 主机地址。
var defaultAPIHosts = map[string]string{
	"openai":    "https://api.openai.com",
	"kimi":      "https://api.moonshot.cn",
	"deepseek":  "https://api.deepseek.com",
	"claude":    "https://api.anthropic.com",
	"gemini":    "https://generativelanguage.googleapis.com/v1beta/openai",
	"microsoft": "https://models.inference.ai.azure.com",
	"github":    "https://models.inference.ai.azure.com",
	"qwen":      "https://dashscope.aliyuncs.com/compatible-mode/v1",
	"zhipu":     "https://open.bigmodel.cn/api/paas/v4",
	"doubao":    "https://ark.cn-beijing.volces.com/api/v3",
	"grok":      "https://api.x.ai",
	"minimax":   "https://api.minimax.chat/v1",
	"xiaomi":    "https://api.mi.ai",
	"hunyuan":   "https://hunyuan.tencentcloudapi.com",
	"vertex":    "https://aiplatform.googleapis.com",
}

// TestAPIKey 验证指定厂商的 API Key 是否有效。
// 通过调用厂商的 /v1/models 接口（Gemini 使用原生 API）进行连通性验证。
func (a *WailsApp) TestAPIKey(providerType string, apiKey string, apiHost string) (*TestAPIKeyResult, error) {
	if providerType == "" {
		return nil, fmt.Errorf("provider type cannot be empty")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("api key cannot be empty")
	}

	if apiHost == "" {
		if h, ok := defaultAPIHosts[providerType]; ok {
			apiHost = h
		} else {
			return nil, fmt.Errorf("unknown provider type: %s", providerType)
		}
	}

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	// Gemini 使用原生 API 验证
	if providerType == "gemini" {
		return a.testGeminiAPIKey(ctx, apiKey)
	}

	// 其他厂商使用 OpenAI 兼容格式 /v1/models
	adapter := ai.NewOpenAIAdapter(apiKey, apiHost, "", 0, 30*time.Second)
	ok, msg := adapter.CheckAvailability(ctx)
	if ok {
		return &TestAPIKeyResult{
			Valid:   true,
			Message: "API Key 验证通过，可正常使用",
		}, nil
	}
	return &TestAPIKeyResult{
		Valid:   false,
		Message: msg,
	}, nil
}

// testGeminiAPIKey 使用 Gemini 原生 API 验证 Key 有效性。
func (a *WailsApp) testGeminiAPIKey(ctx context.Context, apiKey string) (*TestAPIKeyResult, error) {
	reqURL := "https://generativelanguage.googleapis.com/v1beta/models?key=" + url.QueryEscape(apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini test request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &TestAPIKeyResult{
			Valid:   false,
			Message: fmt.Sprintf("连接失败: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return &TestAPIKeyResult{
			Valid:   true,
			Message: "API Key 验证通过，可正常使用",
		}, nil
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &TestAPIKeyResult{
			Valid:   false,
			Message: "API Key 无效或权限不足",
		}, nil
	}

	body, _ := io.ReadAll(resp.Body)
	return &TestAPIKeyResult{
		Valid:   false,
		Message: fmt.Sprintf("验证失败，HTTP %d: %s", resp.StatusCode, string(body)),
	}, nil
}

// CreateProvider 创建新的 Provider 配置。
func (a *WailsApp) CreateProvider(config models.ProviderConfig) error {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	if err := a.providerStore.Create(ctx, &config); err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	return nil
}

// UpdateProvider 更新已有 Provider 配置。
func (a *WailsApp) UpdateProvider(config models.ProviderConfig) error {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	if err := a.providerStore.Update(ctx, &config); err != nil {
		return fmt.Errorf("failed to update provider: %w", err)
	}
	return nil
}

// DeleteProvider 删除指定 Provider 配置。
func (a *WailsApp) DeleteProvider(id string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	if err := a.providerStore.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}
	return nil
}

// ListProviders 获取全部 Provider 配置列表。
func (a *WailsApp) ListProviders() ([]models.ProviderConfig, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	list, err := a.providerStore.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}

	result := make([]models.ProviderConfig, len(list))
	for i, p := range list {
		result[i] = *p
	}
	return result, nil
}

// HealthResultResponse 健康检测结果响应（供前端序列化）。
type HealthResultResponse struct {
	ProviderID string `json:"provider_id"`
	Status     string `json:"status"`
	LatencyMs  int64  `json:"latency_ms"`
	CheckedAt  string `json:"checked_at"`
	Error      string `json:"error,omitempty"`
}

// CheckProviderHealth 对指定 Provider 执行一次即时健康检测。
func (a *WailsApp) CheckProviderHealth(providerID string) (*HealthResultResponse, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.healthChecker == nil {
		return nil, fmt.Errorf("health checker not initialized")
	}

	result, err := a.healthChecker.CheckNow(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check provider health: %w", err)
	}

	return &HealthResultResponse{
		ProviderID: result.ProviderID,
		Status:     string(result.Status),
		LatencyMs:  result.LatencyMs,
		CheckedAt:  result.CheckedAt.Format(time.RFC3339),
		Error:      result.Error,
	}, nil
}

// GetProviderHealthStatus 查询指定 Provider 的缓存健康状态（无需网络请求）。
func (a *WailsApp) GetProviderHealthStatus(providerID string) (*HealthResultResponse, error) {
	if a.healthChecker == nil {
		return nil, fmt.Errorf("health checker not initialized")
	}

	result, ok := a.healthChecker.GetStatus(providerID)
	if !ok {
		return nil, fmt.Errorf("provider %s health status not available", providerID)
	}

	return &HealthResultResponse{
		ProviderID: result.ProviderID,
		Status:     string(result.Status),
		LatencyMs:  result.LatencyMs,
		CheckedAt:  result.CheckedAt.Format(time.RFC3339),
		Error:      result.Error,
	}, nil
}

// RefreshToken 对指定 Provider 手动触发 token 刷新。
func (a *WailsApp) RefreshToken(providerID string) error {
	if providerID == "" {
		return fmt.Errorf("provider_id cannot be empty")
	}
	if a.tokenRefreshSvc == nil {
		return fmt.Errorf("token refresh service not initialized")
	}

	_, err := a.tokenRefreshSvc.Refresh(providerID)
	if err != nil {
		return fmt.Errorf("failed to refresh token for provider %s: %w", providerID, err)
	}
	return nil
}

// EnableAutoRefresh 为指定 Provider 启用自动刷新调度。
func (a *WailsApp) EnableAutoRefresh(providerID string) error {
	if providerID == "" {
		return fmt.Errorf("provider_id cannot be empty")
	}
	if a.tokenRefreshSvc == nil {
		return fmt.Errorf("token refresh service not initialized")
	}

	if err := a.tokenRefreshSvc.ScheduleAutoRefresh(providerID); err != nil {
		return fmt.Errorf("failed to schedule auto refresh for provider %s: %w", providerID, err)
	}
	return nil
}

// DisableAutoRefresh 取消指定 Provider 的自动刷新调度。
func (a *WailsApp) DisableAutoRefresh(providerID string) error {
	if providerID == "" {
		return fmt.Errorf("provider_id cannot be empty")
	}
	if a.tokenRefreshSvc == nil {
		return fmt.Errorf("token refresh service not initialized")
	}

	a.tokenRefreshSvc.CancelAutoRefresh(providerID)
	return nil
}

// DetectCLIToken 检测指定类型的 CLI 是否安装并登录。
// providerType 支持 "kimi" 和 "gemini"。
func (a *WailsApp) DetectCLIToken(providerType string) (*auth.CLIDetectResult, error) {
	if providerType == "" {
		return nil, fmt.Errorf("provider_type cannot be empty")
	}

	svc := auth.NewCLITokenService()
	result, err := svc.Detect(providerType)
	if err != nil {
		return nil, fmt.Errorf("failed to detect cli token: %w", err)
	}
	return result, nil
}

// BuildCLIProvider 根据检测到的 CLI 自动构建 ProviderConfig。
// 流程：Detect → ReadToken → ValidateToken → 返回 ProviderConfig。
// 验证失败时返回错误，不保存到数据库（由调用方决定是否保存）。
func (a *WailsApp) BuildCLIProvider(providerType, modelID string) (*models.ProviderConfig, error) {
	if providerType == "" {
		return nil, fmt.Errorf("provider 类型不能为空")
	}

	svc := auth.NewCLITokenService()

	// 1. 检测 CLI 是否安装
	detect, err := svc.Detect(providerType)
	if err != nil {
		return nil, fmt.Errorf("检测 %s CLI 时出错：%v", providerType, err)
	}
	if !detect.Detected {
		return nil, fmt.Errorf("未检测到 %s CLI 凭证文件，请先运行 `%s login` 登录", providerType, providerType)
	}

	// 2. 读取 token
	token, needsRefresh, err := svc.ReadToken(providerType, detect.CredentialPath)
	if err != nil {
		return nil, fmt.Errorf("读取 %s CLI 凭证失败，文件可能已损坏。建议删除凭证文件后重新登录", providerType)
	}

	// 3. 构建 ProviderConfig
	cfg, err := svc.BuildProviderConfig(providerType, modelID)
	if err != nil {
		return nil, fmt.Errorf("构建 Provider 配置失败：%v", err)
	}

	// 4. 验证 token 有效性（调用厂商 /v1/models）
	// 如果读取到的是 refresh_token，先尝试自动刷新
	if needsRefresh {
		if a.tokenRefreshSvc != nil {
			_, err := a.tokenRefreshSvc.RefreshProvider(cfg)
			if err != nil {
				return nil, fmt.Errorf("自动刷新 %s 登录凭证失败，请运行 `%s login` 重新登录后重试", providerType, providerType)
			}
			// 刷新成功，更新 cfg 中的缓存 token
			cfg.AuthParams.OAuthAccessToken = "" // 让 ResolveAuthToken 重新读取已更新的文件
		} else {
			return nil, fmt.Errorf("%s 登录凭证已过期，但自动刷新服务暂不可用。请运行 `%s login` 重新登录", providerType, providerType)
		}
	} else {
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()

		valid, err := svc.ValidateToken(ctx, cfg.APIHost, token)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return nil, fmt.Errorf("无法连接到 %s 服务，请求超时。请检查网络连接后重试", providerType)
			}
			return nil, fmt.Errorf("无法连接到 %s 服务，请检查网络连接后重试。详情：%v", providerType, err)
		}
		if !valid {
			return nil, fmt.Errorf("CLI Token 已过期或无效，请运行 `%s login` 重新登录", providerType)
		}
	}

	return cfg, nil
}

// --- OAuth Device Flow 绑定方法 ---

// DeviceFlowStartResponse 前端展示用。
type DeviceFlowStartResponse struct {
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	DeviceCode      string `json:"device_code"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	RedirectURI     string `json:"redirect_uri,omitempty"` // 本地回调服务器地址（可选）
}

// DeviceFlowStatusResponse 查询轮询状态。
type DeviceFlowStatusResponse struct {
	DeviceCode   string `json:"device_code"`
	ProviderType string `json:"provider_type"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
	ProviderID   string `json:"provider_id,omitempty"`
	ProviderName string `json:"provider_name,omitempty"`
}

// OAuthDeviceFlowProviderInfo OAuth Device Flow 支持的厂商信息。
type OAuthDeviceFlowProviderInfo struct {
	ProviderType string `json:"provider_type"`
	Name         string `json:"name"`
	Available    bool   `json:"available"`
	Configured   bool   `json:"configured"`
	Detail       string `json:"detail"`
}

// StartOAuthDeviceFlow 对指定厂商启动 OAuth Device Flow。
// 同时启动本地回调服务器作为授权完成的本地通知通道。
func (a *WailsApp) StartOAuthDeviceFlow(providerType string) (*DeviceFlowStartResponse, error) {
	if providerType == "" {
		return nil, fmt.Errorf("provider_type cannot be empty")
	}
	if a.deviceFlowSvc == nil {
		return nil, fmt.Errorf("device flow service not initialized")
	}

	result, err := a.deviceFlowSvc.StartFlow(providerType)
	if err != nil {
		return nil, fmt.Errorf("failed to start device flow for %s: %w", providerType, err)
	}

	// 启动本地回调服务器，作为 Device Flow 的本地通知通道
	callbackServer := auth.NewLocalCallbackServer()
	_, callbackErr := callbackServer.Start()
	if callbackErr == nil {
		a.callbackMu.Lock()
		a.callbackServers[result.DeviceCode] = callbackServer
		a.callbackMu.Unlock()

		// 后台等待回调，收到后触发立即轮询
		go func(deviceCode string, server *auth.LocalCallbackServer) {
			res, err := server.WaitForCallback(15 * time.Minute)
			if err == nil && res.Error == "" {
				_ = a.deviceFlowSvc.TriggerPoll(deviceCode)
			}
			_ = server.Stop()
			a.callbackMu.Lock()
			delete(a.callbackServers, deviceCode)
			a.callbackMu.Unlock()
		}(result.DeviceCode, callbackServer)
	}

	resp := &DeviceFlowStartResponse{
		UserCode:        result.UserCode,
		VerificationURI: result.VerificationURI,
		DeviceCode:      result.DeviceCode,
		ExpiresIn:       result.ExpiresIn,
		Interval:        result.Interval,
	}
	if callbackErr == nil {
		resp.RedirectURI = callbackServer.GetRedirectURI()
	}
	return resp, nil
}

// CancelOAuthDeviceFlow 取消指定 deviceCode 的轮询。
func (a *WailsApp) CancelOAuthDeviceFlow(deviceCode string) error {
	if deviceCode == "" {
		return fmt.Errorf("device_code cannot be empty")
	}
	if a.deviceFlowSvc == nil {
		return fmt.Errorf("device flow service not initialized")
	}

	a.deviceFlowSvc.CancelFlow(deviceCode)

	// 同时停止对应的本地回调服务器
	a.callbackMu.Lock()
	server, ok := a.callbackServers[deviceCode]
	if ok {
		delete(a.callbackServers, deviceCode)
	}
	a.callbackMu.Unlock()
	if ok {
		_ = server.Stop()
	}

	return nil
}

// GetOAuthDeviceFlowStatus 查询指定 deviceCode 的当前轮询状态。
func (a *WailsApp) GetOAuthDeviceFlowStatus(deviceCode string) (*DeviceFlowStatusResponse, error) {
	if deviceCode == "" {
		return nil, fmt.Errorf("device_code cannot be empty")
	}
	if a.deviceFlowSvc == nil {
		return nil, fmt.Errorf("device flow service not initialized")
	}

	status := a.deviceFlowSvc.GetStatus(deviceCode)
	if status == nil {
		return nil, nil
	}

	return &DeviceFlowStatusResponse{
		DeviceCode:   status.DeviceCode,
		ProviderType: status.ProviderType,
		Status:       string(status.Status),
		Error:        status.Error,
		ProviderID:   status.ProviderID,
		ProviderName: status.ProviderName,
	}, nil
}

// GetOAuthDeviceFlowProviders 返回支持 OAuth Device Flow 的厂商列表及其可用状态。
// 基于环境变量是否配置判断各厂商的可用性。
func (a *WailsApp) GetOAuthDeviceFlowProviders() ([]OAuthDeviceFlowProviderInfo, error) {
	providers := []OAuthDeviceFlowProviderInfo{
		{ProviderType: "kimi", Name: "Kimi (Moonshot)"},
		{ProviderType: "gemini", Name: "Gemini (Google)"},
		{ProviderType: "microsoft", Name: "Microsoft Copilot"},
		{ProviderType: "github", Name: "GitHub Copilot"},
	}

	envVars := map[string]string{
		"kimi":      "MEDMEMO_KIMI_CLIENT_ID",
		"gemini":    "MEDMEMO_GEMINI_CLIENT_ID",
		"microsoft": "MEDMEMO_MICROSOFT_CLIENT_ID",
		"github":    "MEDMEMO_GITHUB_CLIENT_ID",
	}

	for i := range providers {
		envVar := envVars[providers[i].ProviderType]
		if envVar == "" {
			providers[i].Available = false
			providers[i].Configured = false
			providers[i].Detail = "未知厂商"
			continue
		}

		if os.Getenv(envVar) != "" {
			providers[i].Available = true
			providers[i].Configured = true
			providers[i].Detail = "已配置 OAuth client_id"
		} else {
			providers[i].Available = false
			providers[i].Configured = false
			providers[i].Detail = fmt.Sprintf("需配置环境变量 %s", envVar)
		}
	}

	return providers, nil
}

// ParseServiceAccountJSON 解析 Google Service Account JSON 密钥内容。
// 返回提取后的 project_id、client_email、private_key。
// 前端应在获取返回值后立即丢弃原始 JSON 字符串。
func (a *WailsApp) ParseServiceAccountJSON(jsonStr string) (map[string]string, error) {
	projectID, clientEmail, privateKey, err := auth.ParseServiceAccountJSON(jsonStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service account JSON: %w", err)
	}
	return map[string]string{
		"project_id":   projectID,
		"client_email": clientEmail,
		"private_key":  privateKey,
	}, nil
}

// shutdownCallbackServers 停止所有运行中的本地回调服务器。
// 在应用退出时由 NewApp.cleanup 调用，确保端口释放。
func (a *WailsApp) shutdownCallbackServers() {
	a.callbackMu.Lock()
	servers := make([]*auth.LocalCallbackServer, 0, len(a.callbackServers))
	for _, s := range a.callbackServers {
		servers = append(servers, s)
	}
	a.callbackServers = make(map[string]*auth.LocalCallbackServer)
	a.callbackMu.Unlock()

	for _, s := range servers {
		_ = s.Stop()
	}
}

// --- Ollama 本地模型检测与引导 ---

// OllamaDetectResult Ollama 环境检测结果，供前端展示。
type OllamaDetectResult struct {
	Installed      bool   `json:"installed"`                 // ollama 命令是否存在于 PATH
	Running        bool   `json:"running"`                   // 11434 端口是否响应
	HasSmolLM2     bool   `json:"has_smollm2"`               // smollm2:135m 模型是否已下载
	InstallGuide   string `json:"install_guide,omitempty"`   // 未安装时返回的安装引导
	ServerStarting bool   `json:"server_starting,omitempty"` // 正在后台启动服务
	PullProgress   string `json:"pull_progress,omitempty"`   // 模型下载进度文本
}

// DetectOllama 检测本地 Ollama 环境状态，返回即时结果（不触发后台操作）。
func (a *WailsApp) DetectOllama() (*OllamaDetectResult, error) {
	detector := ai.NewOllamaDetector()
	d := detector.Detect()

	return &OllamaDetectResult{
		Installed:    d.Installed,
		Running:      d.Running,
		HasSmolLM2:   d.HasSmolLM2,
		InstallGuide: d.InstallGuide,
	}, nil
}

// StartOllamaServer 在后台启动 ollama serve，通过 Wails Events 推送状态变更。
// 事件名称：ollama:server_starting, ollama:server_ready, ollama:server_error
func (a *WailsApp) StartOllamaServer() error {
	a.ollamaMu.Lock()
	defer a.ollamaMu.Unlock()

	detector := ai.NewOllamaDetector()

	if detector.IsRunning() {
		runtime.EventsEmit(a.ctx, "ollama:server_ready", map[string]any{
			"already_running": true,
		})
		return nil
	}

	if !detector.IsInstalled() {
		return fmt.Errorf("ollama is not installed")
	}

	go func() {
		detector := ai.NewOllamaDetector()

		_, err := detector.StartServer()
		if err != nil {
			runtime.EventsEmit(a.ctx, "ollama:server_error", map[string]string{
				"error": fmt.Errorf("failed to start ollama server: %w", err).Error(),
			})
			return
		}

		runtime.EventsEmit(a.ctx, "ollama:server_starting", map[string]string{})

		if err := detector.WaitForServer(30 * time.Second); err != nil {
			runtime.EventsEmit(a.ctx, "ollama:server_error", map[string]string{
				"error": fmt.Errorf("ollama server failed to become ready: %w", err).Error(),
			})
			return
		}

		runtime.EventsEmit(a.ctx, "ollama:server_ready", map[string]any{
			"already_running": false,
		})
	}()

	return nil
}

// PullOllamaModel 在后台执行 ollama pull 下载指定模型，通过 Wails Events 推送进度。
// 事件名称：ollama:pull_progress（每行进度）, ollama:pull_done, ollama:pull_error
func (a *WailsApp) PullOllamaModel(modelName string) error {
	if modelName == "" {
		modelName = ai.DefaultModelName
	}

	a.ollamaMu.Lock()
	defer a.ollamaMu.Unlock()

	detector := ai.NewOllamaDetector()
	if !detector.IsInstalled() {
		return fmt.Errorf("ollama is not installed")
	}

	go func(name string) {
		detector := ai.NewOllamaDetector()
		ctx, cancel := context.WithTimeout(a.ctx, 30*time.Minute)
		defer cancel()

		err := detector.PullModel(ctx, name, func(progress string) {
			runtime.EventsEmit(a.ctx, "ollama:pull_progress", map[string]string{
				"model":    name,
				"progress": progress,
			})
		})

		if err != nil {
			runtime.EventsEmit(a.ctx, "ollama:pull_error", map[string]string{
				"model": name,
				"error": fmt.Errorf("failed to pull model %s: %w", name, err).Error(),
			})
			return
		}

		runtime.EventsEmit(a.ctx, "ollama:pull_done", map[string]string{
			"model": name,
		})
	}(modelName)

	return nil
}

// EnsureOllamaAndSmolLM2 一键检测并确保 Ollama + SmolLM2 就绪。
// 返回当前检测状态；若需要后台操作（启动服务/下载模型），通过 Events 推送进度。
func (a *WailsApp) EnsureOllamaAndSmolLM2() (*OllamaDetectResult, error) {
	detector := ai.NewOllamaDetector()
	d := detector.Detect()

	result := &OllamaDetectResult{
		Installed:    d.Installed,
		Running:      d.Running,
		HasSmolLM2:   d.HasSmolLM2,
		InstallGuide: d.InstallGuide,
	}

	// 未安装：仅返回引导，不触发后台
	if !result.Installed {
		return result, nil
	}

	// 已安装未运行：后台启动
	if !result.Running {
		result.ServerStarting = true
		_ = a.StartOllamaServer()
		return result, nil
	}

	// 已运行但无模型：后台下载
	if !result.HasSmolLM2 {
		_ = a.PullOllamaModel(ai.DefaultModelName)
	}

	return result, nil
}

// CreateOllamaProvider 创建并保存 Ollama Provider 配置到数据库。
func (a *WailsApp) CreateOllamaProvider() (*models.ProviderConfig, error) {
	detector := ai.NewOllamaDetector()
	cfg := detector.BuildProviderConfig()

	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	if err := a.providerStore.Create(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to create ollama provider: %w", err)
	}

	return cfg, nil
}

// --- 认证方式智能检测 ---

// AuthMethodDetectStatus 表示单种认证方式的检测结果。
type AuthMethodDetectStatus struct {
	Method       string `json:"method"`                  // "cli_token" | "oauth_device" | "api_key" | "local"
	Available    bool   `json:"available"`               // 该方式是否可用
	Connected    bool   `json:"connected"`               // 是否已连接/认证成功
	Tier         int    `json:"tier"`                    // 1-4
	ProviderType string `json:"provider_type,omitempty"` // 检测到的厂商类型
	Detail       string `json:"detail,omitempty"`        // 状态描述文本
	Error        string `json:"error,omitempty"`         // 不可用原因
}

// AuthDetectResult 认证方式统一检测结果。
type AuthDetectResult struct {
	Results        []AuthMethodDetectStatus `json:"results"`
	Recommended    string                   `json:"recommended"`     // 推荐的方法
	AllUnavailable bool                     `json:"all_unavailable"` // 是否全部不可用
}

// DetectAuthMethods 并行检测四种认证方式，2 秒内返回统一结果。
// 按 Tier 优先级推荐最佳方式（Tier 1 CLI Token 优先）。
func (a *WailsApp) DetectAuthMethods() (*AuthDetectResult, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	results := make([]AuthMethodDetectStatus, 0, 5)
	var mu sync.Mutex

	detectors := []func() AuthMethodDetectStatus{
		a.detectCLIToken,
		func() AuthMethodDetectStatus { return a.detectOAuthDevice(ctx) },
		a.detectAPIKey,
		func() AuthMethodDetectStatus { return a.detectServiceAccount(ctx) },
		a.detectLocalModel,
	}

	for _, d := range detectors {
		wg.Add(1)
		go func(fn func() AuthMethodDetectStatus) {
			defer wg.Done()
			status := fn()
			mu.Lock()
			results = append(results, status)
			mu.Unlock()
		}(d)
	}

	// 等待所有检测完成或超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}

	recommended, allUnavailable := computeRecommendation(results)
	return &AuthDetectResult{
		Results:        results,
		Recommended:    recommended,
		AllUnavailable: allUnavailable,
	}, nil
}

// detectCLIToken 检测 Kimi / Gemini CLI Token 认证方式。
func (a *WailsApp) detectCLIToken() AuthMethodDetectStatus {
	status := AuthMethodDetectStatus{Method: "cli_token", Tier: 1}
	svc := auth.NewCLITokenService()

	// 优先检测 Kimi CLI
	kimiResult, err := svc.Detect("kimi")
	if err == nil && kimiResult.Detected {
		status.Available = true
		status.ProviderType = "kimi"
		if kimiResult.LoggedIn {
			status.Connected = true
			status.Detail = fmt.Sprintf("已检测到 %s CLI，已登录", kimiResult.ProviderType)
		} else {
			status.Detail = fmt.Sprintf("已检测到 %s CLI，未登录", kimiResult.ProviderType)
			status.Error = "请执行 kimi login 登录"
		}
		return status
	}

	// 降级检测 Gemini CLI
	geminiResult, err := svc.Detect("gemini")
	if err == nil && geminiResult.Detected {
		status.Available = true
		status.ProviderType = "gemini"
		if geminiResult.LoggedIn {
			status.Connected = true
			status.Detail = fmt.Sprintf("已检测到 %s CLI，已登录", geminiResult.ProviderType)
		} else {
			status.Detail = fmt.Sprintf("已检测到 %s CLI，未登录", geminiResult.ProviderType)
			status.Error = "请执行 gcloud auth login 登录"
		}
		return status
	}

	status.Detail = "未检测到 Kimi 或 Gemini CLI 工具"
	status.Error = "未安装 CLI 工具"
	return status
}

// detectOAuthDevice 检测 OAuth Device Flow 认证方式。
func (a *WailsApp) detectOAuthDevice(ctx context.Context) AuthMethodDetectStatus {
	status := AuthMethodDetectStatus{Method: "oauth_device", Tier: 2}

	if a.deviceFlowSvc == nil {
		status.Detail = "OAuth Device Flow 服务未初始化"
		status.Error = "服务不可用"
		return status
	}

	status.Available = true
	status.Detail = "支持 OAuth Device Flow 授权"

	providers, err := a.providerStore.List(ctx)
	if err == nil {
		for _, p := range providers {
			if p.AuthMethod == models.AuthMethodOAuthDevice {
				status.Connected = true
				status.ProviderType = p.ModelID
				status.Detail = fmt.Sprintf("已配置 OAuth Device Flow（%s）", p.Name)
				break
			}
		}
	}
	return status
}

// detectAPIKey 检测 API Key 认证方式。
func (a *WailsApp) detectAPIKey() AuthMethodDetectStatus {
	status := AuthMethodDetectStatus{Method: "api_key", Tier: 3, Available: true, Detail: "可手动输入 API Key"}

	providers := []string{"kimi", "openai", "deepseek", "claude", "qwen"}
	for _, provider := range providers {
		key := fmt.Sprintf("apikey:%s", provider)
		_, err := a.secretStore.Get(key)
		if err == nil {
			status.Connected = true
			status.ProviderType = provider
			status.Detail = fmt.Sprintf("已配置 %s 的 API Key", provider)
			break
		}
	}
	return status
}

// detectServiceAccount 检测 Vertex AI Service Account 认证方式。
func (a *WailsApp) detectServiceAccount(ctx context.Context) AuthMethodDetectStatus {
	status := AuthMethodDetectStatus{Method: "service_account", Tier: 3, Available: true, Detail: "可手动配置 Vertex AI Service Account"}

	providers, err := a.providerStore.List(ctx)
	if err == nil {
		for _, p := range providers {
			if p.AuthMethod == models.AuthMethodServiceAccount {
				status.Connected = true
				status.ProviderType = "vertex"
				status.Detail = fmt.Sprintf("已配置 Vertex AI Service Account（%s）", p.Name)
				break
			}
		}
	}
	return status
}

// detectLocalModel 检测 Ollama 本地模型认证方式。
func (a *WailsApp) detectLocalModel() AuthMethodDetectStatus {
	status := AuthMethodDetectStatus{Method: "local", Tier: 4}

	detector := ai.NewOllamaDetector()
	d := detector.Detect()

	if !d.Installed {
		status.Detail = "未检测到 Ollama"
		status.Error = "Ollama 未安装"
		return status
	}

	status.Available = true
	switch {
	case d.Running && d.HasSmolLM2:
		status.Connected = true
		status.Detail = "Ollama 运行中，SmolLM2 已就绪"
	case d.Running:
		status.Detail = "Ollama 运行中，SmolLM2 未下载"
		status.Error = "模型未下载"
	default:
		status.Detail = "Ollama 已安装，服务未运行"
		status.Error = "服务未运行"
	}
	return status
}

// computeRecommendation 从检测结果中计算推荐认证方式。
func computeRecommendation(results []AuthMethodDetectStatus) (string, bool) {
	allUnavailable := true
	for _, r := range results {
		if r.Available {
			allUnavailable = false
			break
		}
	}

	// 按 Tier 顺序找第一个 available && connected 的
	for tier := 1; tier <= 4; tier++ {
		for _, r := range results {
			if r.Tier == tier && r.Available && r.Connected {
				return r.Method, allUnavailable
			}
		}
	}

	// 没有已连接的，推荐第一个 available 的
	for tier := 1; tier <= 4; tier++ {
		for _, r := range results {
			if r.Tier == tier && r.Available {
				return r.Method, allUnavailable
			}
		}
	}

	// 全部不可用时兜底推荐 local
	return "local", allUnavailable
}

// =============================================================================
// 记忆管理 API（TASK-060）
// =============================================================================

// MemoryItem 记忆列表项 DTO，供前端管理界面展示。
type MemoryItem struct {
	FactID      string  `json:"fact_id"`
	Subject     string  `json:"subject"`
	Predicate   string  `json:"predicate"`
	Object      string  `json:"object"`
	Confidence  float64 `json:"confidence"`
	Status      string  `json:"status"`
	IsSensitive bool    `json:"is_sensitive"`
	CreatedAt   int64   `json:"created_at"`
}

// MemoryStats 记忆统计 DTO。
type MemoryStats struct {
	Total    int64 `json:"total"`
	Approved int64 `json:"approved"`
	Rejected int64 `json:"rejected"`
	Pending  int64 `json:"pending"`
}

// EmbeddingStatusResponse Embedding 模型状态响应。
type EmbeddingStatusResponse struct {
	Available         bool   `json:"available"`           // 语义搜索是否真实可用
	ModelPresent      bool   `json:"model_present"`       // 模型文件是否存在
	EngineAvailable   bool   `json:"engine_available"`    // ONNX embedding 引擎是否可用
	RuntimeLibPresent bool   `json:"runtime_lib_present"` // ONNX Runtime 动态库是否存在
	RuntimeLibPath    string `json:"runtime_lib_path"`    // ONNX Runtime 动态库路径
	FailureReason     string `json:"failure_reason"`      // 初始化失败原因
	ModelPath         string `json:"model_path"`          // 模型存放路径
	ModelName         string `json:"model_name"`          // 模型名称
	DownloadURL       string `json:"download_url"`        // 模型下载页面 URL
}

type embeddingAvailabilityReporter interface {
	IsAvailable() bool
}

type embeddingFailureReasonReporter interface {
	FailureReason() string
}

type embeddingRuntimeLibPathReporter interface {
	RuntimeLibPath() string
}

func factToMemoryItem(f *entity.ExtractedFact) MemoryItem {
	return MemoryItem{
		FactID:      f.FactID,
		Subject:     f.Subject,
		Predicate:   f.Predicate,
		Object:      f.Object,
		Confidence:  f.Confidence,
		Status:      string(f.Status),
		IsSensitive: f.IsSensitive,
		CreatedAt:   f.CreatedAt.UnixMilli(),
	}
}

// requireAuth 检查应用是否已通过首次启动的免责声明同意流程。
// 未授权时返回 ErrUnauthorized，阻止记忆数据的访问。
func (a *WailsApp) requireAuth() error {
	if a.disclaimerRepo == nil {
		return fmt.Errorf("disclaimer repository not initialized: %w", entity.ErrUnauthorized)
	}
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()
	rec, err := a.disclaimerRepo.GetAcceptance(ctx)
	if err != nil {
		return fmt.Errorf("failed to check authorization: %w", err)
	}
	if rec == nil {
		return entity.ErrUnauthorized
	}
	return nil
}

// GetMemories 分页获取已审批的记忆列表。
func (a *WailsApp) GetMemories(limit int, offset int) ([]MemoryItem, error) {
	if err := a.requireAuth(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return nil, fmt.Errorf("fact repository not initialized")
	}

	facts, err := a.factRepo.ListByStatus(ctx, entity.FactStatusApproved, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list memories: %w", err)
	}

	items := make([]MemoryItem, 0, len(facts))
	for _, f := range facts {
		items = append(items, factToMemoryItem(f))
	}
	return items, nil
}

// GetMemoryByID 按 ID 获取单条记忆详情。
func (a *WailsApp) GetMemoryByID(factID string) (MemoryItem, error) {
	if err := a.requireAuth(); err != nil {
		return MemoryItem{}, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return MemoryItem{}, fmt.Errorf("fact repository not initialized")
	}

	f, err := a.factRepo.GetByID(ctx, factID)
	if err != nil {
		return MemoryItem{}, fmt.Errorf("failed to get memory: %w", err)
	}
	return factToMemoryItem(f), nil
}

// DeleteMemory 删除指定记忆（级联删除关联嵌入）。
func (a *WailsApp) DeleteMemory(factID string) error {
	if err := a.requireAuth(); err != nil {
		return fmt.Errorf("requireAuth failed: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return fmt.Errorf("fact repository not initialized")
	}

	if err := a.factRepo.Delete(ctx, factID); err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	// 记录审计日志（失败不影响主流程）
	if a.auditLogRepo != nil {
		entry := entity.NewAuditLogEntry(entity.AuditActionDelete, "fact", factID, "user")
		_ = a.auditLogRepo.Save(ctx, entry)
	}
	return nil
}

// SearchMemories 关键词搜索已审批的记忆。
// 使用数据库层 LIKE 过滤，避免一次性加载全部 approved 事实到内存。
func (a *WailsApp) SearchMemories(query string) ([]MemoryItem, error) {
	if err := a.requireAuth(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return nil, fmt.Errorf("fact repository not initialized")
	}

	facts, err := a.factRepo.SearchApproved(ctx, strings.TrimSpace(query), 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}

	items := make([]MemoryItem, 0, len(facts))
	for _, f := range facts {
		items = append(items, factToMemoryItem(f))
	}
	return items, nil
}

// GetPendingReviews 获取待审核事实列表。
func (a *WailsApp) GetPendingReviews(limit int, offset int) ([]MemoryItem, error) {
	if err := a.requireAuth(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return nil, fmt.Errorf("fact repository not initialized")
	}

	facts, err := a.factRepo.ListPending(ctx, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending reviews: %w", err)
	}

	items := make([]MemoryItem, 0, len(facts))
	for _, f := range facts {
		items = append(items, factToMemoryItem(f))
	}
	return items, nil
}

// ApproveFact 审核通过指定事实。
func (a *WailsApp) ApproveFact(factID string) error {
	if err := a.requireAuth(); err != nil {
		return fmt.Errorf("requireAuth failed: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return fmt.Errorf("fact repository not initialized")
	}

	if err := a.factRepo.UpdateStatus(ctx, factID, entity.FactStatusApproved); err != nil {
		return fmt.Errorf("failed to approve fact: %w", err)
	}

	// 审批通过后生成语义嵌入（向量索引），供记忆召回使用
	if a.embeddingSvc != nil && a.embeddingRepo != nil {
		fact, err := a.factRepo.GetByID(ctx, factID)
		if err == nil && fact != nil {
			content := usecase.BuildFactRetrievalText(fact)
			vector, embErr := a.embeddingSvc.EmbedSingle(ctx, content)
			if embErr == nil {
				embedding := entity.NewSemanticEmbedding(factID, vector, models.CurrentEmbeddingVersion)
				if saveErr := a.embeddingRepo.Save(ctx, embedding); saveErr != nil {
					fmt.Printf("[ApproveFact] 保存嵌入向量失败 %s: %v\n", factID, saveErr)
				}
			} else {
				fmt.Printf("[ApproveFact] 生成嵌入向量失败 %s: %v\n", factID, embErr)
			}
		}
	}

	// 记录审计日志（失败不影响主流程）
	if a.auditLogRepo != nil {
		entry := entity.NewAuditLogEntry(entity.AuditActionApprove, "fact", factID, "user")
		_ = a.auditLogRepo.Save(ctx, entry)
	}
	return nil
}

// RejectFact 审核拒绝指定事实。
func (a *WailsApp) RejectFact(factID string) error {
	if err := a.requireAuth(); err != nil {
		return fmt.Errorf("requireAuth failed: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return fmt.Errorf("fact repository not initialized")
	}

	if err := a.factRepo.UpdateStatus(ctx, factID, entity.FactStatusRejected); err != nil {
		return fmt.Errorf("failed to reject fact: %w", err)
	}

	// 拒绝时清理可能已存在的 embedding，避免历史版本或异常写入留下的 stale 向量
	if a.embeddingRepo != nil {
		if delErr := a.embeddingRepo.DeleteByFactID(ctx, factID); delErr != nil {
			fmt.Printf("[RejectFact] 清理 embedding 失败 %s: %v\n", factID, delErr)
		}
	}

	// 记录审计日志（失败不影响主流程）
	if a.auditLogRepo != nil {
		entry := entity.NewAuditLogEntry(entity.AuditActionReject, "fact", factID, "user")
		_ = a.auditLogRepo.Save(ctx, entry)
	}
	return nil
}

// embeddingModelDir 返回 Embedding 模型目录的绝对路径。
// 始终使用用户数据目录 ~/.medmemo/data/models/all-MiniLM-L6-v2，
// 确保在 AppImage（只读 FS）、macOS .app bundle 及 Windows 安装目录中均可正常读写。
func (a *WailsApp) embeddingModelDir() string {
	return filepath.Join(a.config.DataDir, "models", models.EmbeddingModelName)
}

// GetEmbeddingStatus 获取本地 Embedding 模型状态。
func (a *WailsApp) GetEmbeddingStatus() (*EmbeddingStatusResponse, error) {
	if a.config == nil {
		return nil, fmt.Errorf("app config not initialized")
	}
	modelPath := a.embeddingModelDir()
	modelFile := filepath.Join(modelPath, "model.onnx")
	tokenizerFile := filepath.Join(modelPath, "tokenizer.json")

	modelPresent := fileExists(modelFile)
	tokenizerPresent := fileExists(tokenizerFile)

	runtimeLibPath := ""
	runtimeLibPresent := false
	if reporter, ok := a.embeddingSvc.(embeddingRuntimeLibPathReporter); ok {
		runtimeLibPath = reporter.RuntimeLibPath()
	}
	if runtimeLibPath == "" {
		runtimeLibPath = defaultONNXRuntimeLibPath(resourcepath.Dir())
	}
	if runtimeLibPath != "" {
		runtimeLibPresent = fileExists(runtimeLibPath)
	}

	engineAvailable := false
	if reporter, ok := a.embeddingSvc.(embeddingAvailabilityReporter); ok {
		engineAvailable = reporter.IsAvailable()
	} else if a.embeddingSvc != nil {
		baseCtx := a.ctx
		if baseCtx == nil {
			baseCtx = context.Background()
		}
		ctx, cancel := context.WithTimeout(baseCtx, 2*time.Second)
		defer cancel()
		if _, err := a.embeddingSvc.EmbedSingle(ctx, "status check"); err == nil {
			engineAvailable = true
		}
	}

	failureReason := ""
	if reporter, ok := a.embeddingSvc.(embeddingFailureReasonReporter); ok {
		failureReason = reporter.FailureReason()
	}
	switch {
	case !modelPresent:
		failureReason = "embedding model file is missing"
	case !tokenizerPresent && !engineAvailable:
		failureReason = fmt.Sprintf("embedding tokenizer file is missing: %s", tokenizerFile)
	case !runtimeLibPresent:
		if runtimeLibPath == "" {
			failureReason = "ONNX Runtime library path could not be resolved"
		} else {
			failureReason = fmt.Sprintf("ONNX Runtime library not found: %s", runtimeLibPath)
		}
	case !engineAvailable && failureReason == "":
		failureReason = "embedding engine not available"
	}

	downloadURL := a.config.EmbeddingModelDownloadURL
	if downloadURL == "" {
		// 默认指向 GitHub Release 下载页面，用户可在 config.yaml 中自定义
		downloadURL = "https://github.com/hzhan516/medmemo/releases/tag/embedding-model-v1"
	}

	// 向前端返回绝对路径，便于展示
	absPath, _ := filepath.Abs(modelPath)
	if absPath != "" {
		modelPath = absPath
	}

	return &EmbeddingStatusResponse{
		Available:         modelPresent && engineAvailable,
		ModelPresent:      modelPresent,
		EngineAvailable:   engineAvailable,
		RuntimeLibPresent: runtimeLibPresent,
		RuntimeLibPath:    runtimeLibPath,
		FailureReason:     failureReason,
		ModelPath:         modelPath,
		ModelName:         models.EmbeddingModelName,
		DownloadURL:       downloadURL,
	}, nil
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func defaultONNXRuntimeLibPath(resourceDir string) string {
	switch goruntime.GOOS {
	case "linux":
		primary := filepath.Join(resourceDir, "lib", "linux", "libonnxruntime.so")
		if fileExists(primary) {
			return primary
		}
		return filepath.Join(resourceDir, "lib", "linux", "libonnxruntime.so.1")
	case "darwin":
		return filepath.Join(resourceDir, "lib", "darwin", "libonnxruntime.dylib")
	case "windows":
		return filepath.Join(resourceDir, "lib", "windows", "onnxruntime.dll")
	default:
		return ""
	}
}

// GetEmbeddingModelDirPath 返回 Embedding 模型目录的绝对路径。
func (a *WailsApp) GetEmbeddingModelDirPath() (string, error) {
	modelPath := a.embeddingModelDir()
	absPath, err := filepath.Abs(modelPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve model dir path: %w", err)
	}
	// 确保目录存在（使用用户可写路径，不会在只读 FS 上失败）
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create model dir: %w", err)
	}
	return absPath, nil
}

// OpenEmbeddingModelDir 打开 Embedding 模型所在目录。
// 使用平台特定命令打开文件管理器，比 BrowserOpenURL 更可靠。
func (a *WailsApp) OpenEmbeddingModelDir() error {
	absPath, err := a.GetEmbeddingModelDirPath()
	if err != nil {
		return fmt.Errorf("failed to get embedding model directory: %w", err)
	}

	var cmd string
	var args []string
	switch goruntime.GOOS {
	case "windows":
		cmd = "explorer.exe"
		args = []string{absPath}
	case "darwin":
		cmd = "open"
		args = []string{absPath}
	default: // linux and others
		cmd = "xdg-open"
		args = []string{absPath}
	}

	c := exec.Command(cmd, args...)
	if err := c.Start(); err != nil {
		// 命令启动失败时，弹窗提示用户手动前往
		_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.InfoDialog,
			Title:   "打开模型目录",
			Message: fmt.Sprintf("无法自动打开文件管理器，请手动前往以下目录：\n\n%s", absPath),
		})
		return fmt.Errorf("failed to open model dir with %s: %w", cmd, err)
	}
	return nil
}

// GetMemoryStats 获取记忆审核统计。
func (a *WailsApp) GetMemoryStats() (MemoryStats, error) {
	if err := a.requireAuth(); err != nil {
		return MemoryStats{}, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return MemoryStats{}, fmt.Errorf("fact repository not initialized")
	}

	total, approved, rejected, pending, err := a.factRepo.GetStats(ctx)
	if err != nil {
		return MemoryStats{}, fmt.Errorf("failed to get memory stats: %w", err)
	}
	return MemoryStats{
		Total:    total,
		Approved: approved,
		Rejected: rejected,
		Pending:  pending,
	}, nil
}

// GetMemoriesBySession 按会话 ID 获取关联的已审批记忆。
func (a *WailsApp) GetMemoriesBySession(sessionID string) ([]MemoryItem, error) {
	if err := a.requireAuth(); err != nil {
		return nil, err
	}
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return nil, fmt.Errorf("fact repository not initialized")
	}

	facts, err := a.factRepo.FindBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get memories by session: %w", err)
	}

	items := make([]MemoryItem, 0, len(facts))
	for _, f := range facts {
		items = append(items, factToMemoryItem(f))
	}
	return items, nil
}

// SetMemoryInjectionEnabled 设置记忆注入全局开关。
func (a *WailsApp) SetMemoryInjectionEnabled(enabled bool) error {
	if err := a.requireAuth(); err != nil {
		return fmt.Errorf("requireAuth failed: %w", err)
	}
	if a.memoryRetriever == nil {
		return fmt.Errorf("memory retriever not initialized")
	}
	a.memoryRetriever.SetEnabled(enabled)
	return nil
}

// SetSessionMemoryInjection 设置指定会话的记忆注入开关。
func (a *WailsApp) SetSessionMemoryInjection(sessionID string, enabled bool) error {
	if err := a.requireAuth(); err != nil {
		return fmt.Errorf("requireAuth failed: %w", err)
	}
	if a.memoryRetriever == nil {
		return fmt.Errorf("memory retriever not initialized")
	}
	a.memoryRetriever.SetSessionEnabled(sessionID, enabled)
	return nil
}
