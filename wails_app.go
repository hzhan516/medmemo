// Package main 是 MedMemo 应用入口。
// WailsApp 暴露前端可调用的绑定方法集，供 Wire 注入后绑定到 Wails 运行时。
package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/medmemo/medmemo/internal/adapters/ai"
	"github.com/medmemo/medmemo/internal/adapters/auth"
	"github.com/medmemo/medmemo/internal/application"
	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/internal/application/stream"
	"github.com/medmemo/medmemo/internal/application/updater"
	"github.com/medmemo/medmemo/internal/application/usecase"
	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/internal/infrastructure/secret"
	"github.com/medmemo/medmemo/pkg/models"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// WailsApp 暴露给前端 Wails 绑定的方法集。
type WailsApp struct {
	ctx              context.Context
	chatOrchestrator *usecase.ChatOrchestrator
	memoryRetriever  *usecase.MemoryRetriever
	config           *entity.AppConfig
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
	streamCancel     context.CancelFunc

	// 本地回调服务器管理（每个 Device Flow 会话对应一个）
	callbackServers map[string]*auth.LocalCallbackServer
	callbackMu      sync.Mutex

	// Ollama 操作互斥锁，防止并发启动/下载冲突
	ollamaMu sync.Mutex
}

// NewWailsApp 构造函数，供 Wire 调用。
func NewWailsApp(
	chat *usecase.ChatOrchestrator,
	mem *usecase.MemoryRetriever,
	cfg *entity.AppConfig,
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
		callbackServers:  make(map[string]*auth.LocalCallbackServer),
	}
}

// Startup 是 Wails 启动回调，在前端加载完成后调用。
func (a *WailsApp) Startup(ctx context.Context) {
	a.ctx = ctx

	// 初始化 token 刷新降级回调
	if a.tokenRefreshSvc != nil {
		a.tokenRefreshSvc.SetOnDegraded(func(providerID, reason string) {
			runtime.EventsEmit(a.ctx, "auth:degraded", map[string]string{
				"provider_id": providerID,
				"reason":      reason,
			})
		})
	}

	// 启动健康检测引擎
	if a.healthChecker != nil {
		a.healthChecker.SetOnChange(func(result port.HealthResult) {
			runtime.EventsEmit(a.ctx, "provider:health_changed", result)
		})
		a.healthChecker.Start(a.ctx)
	}

	// 启动时异步检测更新（不阻塞首屏）
	if a.config.UpdateCheckEnabled && a.updaterSvc != nil {
		go a.checkUpdateAsync()
	}

	// 启动时扫描 cli_token / oauth_device provider，安排自动刷新
	if a.tokenRefreshSvc != nil {
		go a.scheduleAutoRefreshesAsync()
	}

	// 初始化 Device Flow 事件回调
	if a.deviceFlowSvc != nil {
		a.deviceFlowSvc.SetRefreshService(a.tokenRefreshSvc)
		a.deviceFlowSvc.SetCallbacks(
			func(deviceCode, providerType string, cfg *models.ProviderConfig) {
				runtime.EventsEmit(a.ctx, "oauth:success", map[string]any{
					"device_code":   deviceCode,
					"provider_type": providerType,
					"provider_id":   cfg.ID,
					"provider_name": cfg.Name,
				})
			},
			func(deviceCode, providerType string, err error) {
				runtime.EventsEmit(a.ctx, "oauth:error", map[string]any{
					"device_code":   deviceCode,
					"provider_type": providerType,
					"error":         err.Error(),
				})
			},
			func(deviceCode, providerType string) {
				runtime.EventsEmit(a.ctx, "oauth:pending", map[string]any{
					"device_code":   deviceCode,
					"provider_type": providerType,
				})
			},
			func(deviceCode, providerType string, newInterval int) {
				runtime.EventsEmit(a.ctx, "oauth:slow_down", map[string]any{
					"device_code":   deviceCode,
					"provider_type": providerType,
					"new_interval":  newInterval,
				})
			},
		)
	}
}

// scheduleAutoRefreshesAsync 延迟 3 秒后扫描并安排自动刷新，避免与启动流程竞争。
func (a *WailsApp) scheduleAutoRefreshesAsync() {
	time.Sleep(3 * time.Second)
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	list, err := a.providerStore.List(ctx)
	if err != nil {
		return
	}

	for _, p := range list {
		if !p.Enabled {
			continue
		}
		if p.AuthMethod != models.AuthMethodCLIToken && p.AuthMethod != models.AuthMethodOAuthDevice {
			continue
		}
		creds, err := models.ReadCLICredentials(p.AuthParams.CLICredentialPath)
		if err != nil {
			continue
		}
		// 只有含 refresh_token 的 provider 才需要自动刷新
		if creds.RefreshToken != "" || p.AuthParams.OAuthRefreshToken != "" {
			_ = a.tokenRefreshSvc.ScheduleAutoRefresh(p.ID)
		}
	}
}

// checkUpdateAsync 延迟 5 秒后异步检测更新，避免与启动流程竞争资源。
func (a *WailsApp) checkUpdateAsync() {
	time.Sleep(5 * time.Second)
	if !a.updaterSvc.GetSettings().ShouldCheck(updater.CheckInterval) {
		return
	}

	info, err := a.updaterSvc.CheckUpdate(a.ctx, version)
	if err != nil || info == nil {
		return
	}

	// 通过 Wails Events 推送更新通知到前端
	payload := map[string]any{
		"version":      info.Version,
		"name":         info.Name,
		"body":         info.Body,
		"published_at": info.PublishedAt.Format(time.RFC3339),
		"mandatory":    info.Mandatory,
		"channel":      string(info.Channel),
	}
	runtime.EventsEmit(a.ctx, "update:available", payload)
}

// SendMessageRequest 前端发送消息请求。
type SendMessageRequest struct {
	ConversationID string           `json:"conversation_id"`
	Messages       []models.Message `json:"messages"`
	Model          string           `json:"model"`
}

// SendMessageResponse 发送消息响应。
type SendMessageResponse struct {
	Reply      string   `json:"reply"`
	Confidence float64  `json:"confidence"`
	Warnings   []string `json:"warnings"`
}

// SendMessage 发送对话消息，编排完整对话流程（非流式）。
func (a *WailsApp) SendMessage(req SendMessageRequest) (*SendMessageResponse, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	chatReq := usecase.ChatRequest{
		ConversationID: models.ConversationID(req.ConversationID),
		Messages:       req.Messages,
		Model:          models.ProviderType(req.Model),
	}

	resp, err := a.chatOrchestrator.Execute(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return &SendMessageResponse{
		Reply:      resp.Reply,
		Confidence: resp.Confidence,
		Warnings:   resp.Warnings,
	}, nil
}

// SendMessageStream 发送流式对话请求，通过 Wails Events 实时推送结构化 StreamChunk。
func (a *WailsApp) SendMessageStream(req SendMessageRequest) error {
	ctx, cancel := context.WithTimeout(a.ctx, 120*time.Second)

	a.streamMu.Lock()
	a.streamCancel = cancel
	a.streamMu.Unlock()

	defer func() {
		a.streamMu.Lock()
		a.streamCancel = nil
		a.streamMu.Unlock()
		cancel()
	}()

	chatReq := usecase.ChatRequest{
		ConversationID: models.ConversationID(req.ConversationID),
		Messages:       req.Messages,
		Model:          models.ProviderType(req.Model),
	}

	// 统一流式处理层：将原始 callback 包装为结构化 StreamChunk 序列
	broker := stream.NewBroker(req.Model, "", func(chunk models.StreamChunk) {
		runtime.EventsEmit(a.ctx, "chat:stream_chunk", chunk)
	})
	broker.Start()

	// 收集 AI 完整回复用于持久化
	var fullReply stringsBuilder

	err := a.chatOrchestrator.StreamExecute(ctx, chatReq, func(chunk string) {
		fullReply.WriteString(chunk)
		broker.Content(chunk)
	})

	if err != nil {
		if errors.Is(err, context.Canceled) {
			broker.Error("生成已中断")
			// 保存已生成的部分内容
			a.saveMessages(ctx, req.ConversationID, req.Messages, fullReply.String())
			return nil
		}
		broker.Error(err.Error())
		return fmt.Errorf("stream failed: %w", err)
	}

	// 保存用户消息和 AI 回复
	a.saveMessages(ctx, req.ConversationID, req.Messages, fullReply.String())

	// 流式结束后对完整内容做一次合规检测（MVP 简化策略）
	compResult, compErr := a.chatOrchestrator.CheckCompliance(ctx, fullReply.String())
	if compErr == nil && compResult.Level != "L4_NORMAL" {
		payload := map[string]any{
			"level":         compResult.Level,
			"warning":       compResult.Warning,
			"notice":        compResult.Notice,
			"replacedTerms": compResult.ReplacedTerms,
			"matchedRule":   compResult.MatchedRule,
		}
		runtime.EventsEmit(a.ctx, "chat:stream:compliance", payload)
	}

	broker.Done()
	return nil
}

// stringsBuilder 是 strings.Builder 的别名，用于收集流式内容。
type stringsBuilder struct {
	b []byte
}

func (s *stringsBuilder) WriteString(str string) {
	s.b = append(s.b, str...)
}

func (s *stringsBuilder) String() string {
	return string(s.b)
}

// saveMessages 将对话消息持久化到数据库。
func (a *WailsApp) saveMessages(ctx context.Context, convID string, messages []models.Message, aiReply string) {
	if a.msgRepo == nil || convID == "" {
		return
	}
	// 保存最后一条用户消息
	if len(messages) > 0 {
		lastUser := messages[len(messages)-1]
		if lastUser.Role == models.RoleUser {
			_ = a.msgRepo.Save(ctx, models.ConversationID(convID), &entity.Message{
				ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
				Role:      lastUser.Role,
				Content:   lastUser.Content,
				Timestamp: time.Now(),
			})
		}
	}
	// 保存 AI 回复
	if aiReply != "" {
		_ = a.msgRepo.Save(ctx, models.ConversationID(convID), &entity.Message{
			ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
			Role:      models.RoleAssistant,
			Content:   aiReply,
			Timestamp: time.Now(),
		})
	}
	// 更新会话时间
	if a.convRepo != nil {
		_ = a.convRepo.Save(ctx, &entity.Conversation{
			ID:        models.ConversationID(convID),
			UpdatedAt: time.Now(),
		})
	}
}

// StopGeneration 中断当前正在进行的流式生成。
func (a *WailsApp) StopGeneration() {
	a.streamMu.Lock()
	if a.streamCancel != nil {
		a.streamCancel()
	}
	a.streamMu.Unlock()
}

// ConversationSummary 会话摘要。
type ConversationSummary struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	UpdatedAt string `json:"updated_at"`
}

// GetConversations 获取会话列表。
func (a *WailsApp) GetConversations() ([]ConversationSummary, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	convs, err := a.convRepo.ListRecent(ctx, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	result := make([]ConversationSummary, len(convs))
	for i, conv := range convs {
		result[i] = ConversationSummary{
			ID:        string(conv.ID),
			Title:     conv.Title,
			UpdatedAt: strconv.FormatInt(conv.UpdatedAt.UnixMilli(), 10),
		}
	}
	return result, nil
}

// CreateConversation 创建新会话，返回会话 ID。
func (a *WailsApp) CreateConversation() (string, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	conv := entity.NewConversation(models.ProviderType(a.config.DefaultModel))
	if err := a.convRepo.Save(ctx, conv); err != nil {
		return "", fmt.Errorf("failed to create conversation: %w", err)
	}
	return string(conv.ID), nil
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

// EmergencyResult 紧急症状检测结果。
type EmergencyResult struct {
	Level   string `json:"level"`   // A, B, none
	Message string `json:"message"` // 提示信息
	Action  string `json:"action"`  // 建议操作
}

// CheckEmergency 检查文本是否包含紧急症状（AGENTS.md 7.3）。
// 委托 application 层的 EvaluateEmergency 执行本地关键词匹配，延迟 <5ms，独立于 AI 回复流程。
func (a *WailsApp) CheckEmergency(text string) (*EmergencyResult, error) {
	result := application.EvaluateEmergency(text)
	return &EmergencyResult{
		Level:   string(result.Level),
		Message: result.Message,
		Action:  result.Action,
	}, nil
}

// GenerateTitle 异步生成会话标题，通过 Wails Events 推送结果。
// 前端应在首条用户消息发送后调用此方法。
func (a *WailsApp) GenerateTitle(convID string, userMessage string) {
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
		defer cancel()

		title, err := a.titleGen.Generate(ctx, userMessage)
		if err != nil {
			// AI 生成失败或超时，降级到本地规则
			title = usecase.FallbackTitle(userMessage)
		}

		// 持久化到数据库
		if a.convRepo != nil {
			_ = a.convRepo.Save(ctx, &entity.Conversation{
				ID:        models.ConversationID(convID),
				Title:     title,
				UpdatedAt: time.Now(),
			})
		}

		// 推送前端更新
		runtime.EventsEmit(a.ctx, "chat:title:generated", map[string]string{
			"conv_id": convID,
			"title":   title,
		})
	}()
}

// DisclaimerStatus 返回当前免责声明状态，供前端在启动时检测是否需要展示。
type DisclaimerStatus struct {
	Required bool   `json:"required"`
	Text     string `json:"text"`
	Version  string `json:"version"`
}

// GetDisclaimerStatus 查询用户是否需要同意当前版本的免责声明。
func (a *WailsApp) GetDisclaimerStatus() (*DisclaimerStatus, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	rec, err := a.disclaimerRepo.GetAcceptance(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get disclaimer status: %w", err)
	}

	// 若从未同意，或已同意版本低于当前版本，均需重新展示
	if rec == nil || rec.Version != entity.CurrentDisclaimerVersion {
		return &DisclaimerStatus{
			Required: true,
			Text:     entity.DisclaimerText,
			Version:  entity.CurrentDisclaimerVersion,
		}, nil
	}

	return &DisclaimerStatus{
		Required: false,
		Text:     entity.DisclaimerText,
		Version:  entity.CurrentDisclaimerVersion,
	}, nil
}

// AcceptDisclaimer 记录用户同意当前版本的免责声明。
func (a *WailsApp) AcceptDisclaimer(version string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	if version != entity.CurrentDisclaimerVersion {
		return fmt.Errorf("disclaimer version mismatch: expected %s, got %s", entity.CurrentDisclaimerVersion, version)
	}

	rec := &entity.DisclaimerAcceptance{
		Version:    version,
		AcceptedAt: time.Now(),
		TextHash:   "", // 当前阶段无需哈希校验，预留字段
	}
	if err := a.disclaimerRepo.SaveAcceptance(ctx, rec); err != nil {
		return fmt.Errorf("failed to save disclaimer acceptance: %w", err)
	}
	return nil
}

// DeclineDisclaimer 用户不同意免责声明，退出应用。
func (a *WailsApp) DeclineDisclaimer() {
	runtime.Quit(a.ctx)
}

// ShowEmergencyDialog 触发紧急症状弹窗（供前端调用）。
func (a *WailsApp) ShowEmergencyDialog(title, message string) {
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.WarningDialog,
		Title:   title,
		Message: message,
	})
}

// ReportComplianceFeedback 接收前端提交的合规误判反馈。
func (a *WailsApp) ReportComplianceFeedback(ruleID string, originalText string) error {
	logger := application.NewComplianceLogger("data")
	if err := logger.LogFeedback(a.ctx, ruleID, originalText, "false_positive"); err != nil {
		return fmt.Errorf("failed to log compliance feedback: %w", err)
	}
	return nil
}

// UpdateInfoResponse 前端更新信息响应。
type UpdateInfoResponse struct {
	Version     string `json:"version"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	Mandatory   bool   `json:"mandatory"`
	Channel     string `json:"channel"`
}

// CheckUpdate 检测是否存在可用更新，供前端主动调用。
func (a *WailsApp) CheckUpdate() (*UpdateInfoResponse, error) {
	if a.updaterSvc == nil {
		return nil, fmt.Errorf("updater service not initialized")
	}

	info, err := a.updaterSvc.CheckUpdate(a.ctx, version)
	if err != nil {
		return nil, fmt.Errorf("failed to check update: %w", err)
	}
	if info == nil {
		return nil, nil
	}

	return &UpdateInfoResponse{
		Version:     info.Version,
		Name:        info.Name,
		Body:        info.Body,
		PublishedAt: info.PublishedAt.Format(time.RFC3339),
		Mandatory:   info.Mandatory,
		Channel:     string(info.Channel),
	}, nil
}

// DownloadUpdateRequest 下载更新请求。
type DownloadUpdateRequest struct {
	Version string `json:"version"`
}

// DownloadUpdate 下载指定版本的更新包。
// 下载进度通过 Wails Events "update:progress" 推送。
func (a *WailsApp) DownloadUpdate(req DownloadUpdateRequest) (string, error) {
	if a.updaterSvc == nil {
		return "", fmt.Errorf("updater service not initialized")
	}

	// 先重新获取 UpdateInfo（包含正确的下载 URL）
	info, err := a.updaterSvc.CheckUpdate(a.ctx, version)
	if err != nil || info == nil {
		return "", fmt.Errorf("failed to get update info for version %s: %w", req.Version, err)
	}

	progressCb := func(downloaded, total int64) {
		runtime.EventsEmit(a.ctx, "update:progress", map[string]int64{
			"downloaded": downloaded,
			"total":      total,
		})
	}

	path, err := a.updaterSvc.DownloadUpdate(a.ctx, info, progressCb)
	if err != nil {
		return "", fmt.Errorf("failed to download update: %w", err)
	}

	return path, nil
}

// ApplyUpdate 应用已下载的更新。
func (a *WailsApp) ApplyUpdate(assetPath string) error {
	if a.updaterSvc == nil {
		return fmt.Errorf("updater service not initialized")
	}

	if err := a.updaterSvc.ApplyUpdate(assetPath); err != nil {
		return fmt.Errorf("failed to apply update: %w", err)
	}
	return nil
}

// UpdateSettingsResponse 更新设置响应。
type UpdateSettingsResponse struct {
	CheckEnabled bool   `json:"check_enabled"`
	Channel      string `json:"channel"`
	SkipVersion  string `json:"skip_version"`
}

// GetUpdateSettings 获取当前更新设置。
func (a *WailsApp) GetUpdateSettings() (*UpdateSettingsResponse, error) {
	if a.updaterSvc == nil {
		return nil, fmt.Errorf("updater service not initialized")
	}

	s := a.updaterSvc.GetSettings()
	return &UpdateSettingsResponse{
		CheckEnabled: s.CheckEnabled,
		Channel:      string(s.Channel),
		SkipVersion:  s.SkipVersion,
	}, nil
}

// SetUpdateSettings 保存更新设置。
func (a *WailsApp) SetUpdateSettings(req UpdateSettingsResponse) error {
	if a.updaterSvc == nil {
		return fmt.Errorf("updater service not initialized")
	}

	s := &entity.UpdateSettings{
		CheckEnabled: req.CheckEnabled,
		Channel:      entity.UpdateChannel(req.Channel),
		SkipVersion:  req.SkipVersion,
	}
	a.updaterSvc.SetSettings(s)
	return nil
}

// SkipUpdateVersion 标记跳过指定版本。
func (a *WailsApp) SkipUpdateVersion(v string) error {
	if a.updaterSvc == nil {
		return fmt.Errorf("updater service not initialized")
	}
	a.updaterSvc.SkipVersion(v)
	return nil
}

// OpenDownloadURL 通过系统浏览器打开指定 URL。
func (a *WailsApp) OpenDownloadURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
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

// GetVersion 返回当前应用版本号（构建时通过 -ldflags 注入）。
func (a *WailsApp) GetVersion() string {
	return version
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
		return nil, fmt.Errorf("provider_type cannot be empty")
	}

	svc := auth.NewCLITokenService()

	// 1. 检测 CLI 是否安装
	detect, err := svc.Detect(providerType)
	if err != nil {
		return nil, fmt.Errorf("failed to detect cli: %w", err)
	}
	if !detect.Detected {
		return nil, fmt.Errorf("cli %s not detected (credential file not found)", providerType)
	}

	// 2. 读取 token
	token, needsRefresh, err := svc.ReadToken(providerType, detect.CredentialPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cli token: %w", err)
	}

	// 3. 构建 ProviderConfig
	cfg, err := svc.BuildProviderConfig(providerType, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to build provider config: %w", err)
	}

	// 4. 验证 token 有效性（调用厂商 /v1/models）
	// 如果读取到的是 refresh_token，先尝试自动刷新
	if needsRefresh {
		if a.tokenRefreshSvc != nil {
			_, err := a.tokenRefreshSvc.RefreshProvider(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to refresh cli token for %s: %w", providerType, err)
			}
			// 刷新成功，更新 cfg 中的缓存 token
			cfg.AuthParams.OAuthAccessToken = "" // 让 ResolveAuthToken 重新读取已更新的文件
		} else {
			return nil, fmt.Errorf("cli token for %s is a refresh_token and token refresh service is not available", providerType)
		}
	} else {
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()

		valid, err := svc.ValidateToken(ctx, cfg.APIHost, token)
		if err != nil {
			return nil, fmt.Errorf("token validation failed: %w", err)
		}
		if !valid {
			return nil, fmt.Errorf("cli token for %s is invalid or expired", providerType)
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
