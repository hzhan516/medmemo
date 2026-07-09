package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/hzhan516/medmemo/internal/adapters/ai"
	"github.com/hzhan516/medmemo/internal/adapters/auth"
	"github.com/hzhan516/medmemo/pkg/models"
)

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
