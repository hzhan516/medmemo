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

	"github.com/medmemo/medmemo/internal/application"
	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/internal/application/updater"
	"github.com/medmemo/medmemo/internal/application/usecase"
	"github.com/medmemo/medmemo/internal/domain/entity"
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
	titleGen         *usecase.TitleGenerator
	updaterSvc       *updater.Service
	streamMu         sync.Mutex
	streamCancel     context.CancelFunc
}

// NewWailsApp 构造函数，供 Wire 调用。
func NewWailsApp(
	chat *usecase.ChatOrchestrator,
	mem *usecase.MemoryRetriever,
	cfg *entity.AppConfig,
	convRepo port.ConversationRepository,
	msgRepo port.MessageRepository,
	disclaimerRepo port.DisclaimerRepository,
	titleGen *usecase.TitleGenerator,
	updaterSvc *updater.Service,
) *WailsApp {
	return &WailsApp{
		chatOrchestrator: chat,
		memoryRetriever:  mem,
		config:           cfg,
		convRepo:         convRepo,
		msgRepo:          msgRepo,
		disclaimerRepo:   disclaimerRepo,
		titleGen:         titleGen,
		updaterSvc:       updaterSvc,
	}
}

// Startup 是 Wails 启动回调，在前端加载完成后调用。
func (a *WailsApp) Startup(ctx context.Context) {
	a.ctx = ctx

	// 启动时异步检测更新（不阻塞首屏）
	if a.config.UpdateCheckEnabled && a.updaterSvc != nil {
		go a.checkUpdateAsync()
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

// SendMessageStream 发送流式对话请求，通过 Wails Events 实时推送 token。
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

	// 收集 AI 完整回复用于持久化
	var fullReply stringsBuilder

	err := a.chatOrchestrator.StreamExecute(ctx, chatReq, func(chunk string) {
		fullReply.WriteString(chunk)
		runtime.EventsEmit(a.ctx, "chat:stream:token", chunk)
	})

	if err != nil {
		if errors.Is(err, context.Canceled) {
			runtime.EventsEmit(a.ctx, "chat:stream:interrupted", nil)
			// 保存已生成的部分内容
			a.saveMessages(ctx, req.ConversationID, req.Messages, fullReply.String())
			return nil
		}
		runtime.EventsEmit(a.ctx, "chat:stream:error", err.Error())
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

	runtime.EventsEmit(a.ctx, "chat:stream:end", nil)
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
