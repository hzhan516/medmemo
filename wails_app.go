// Package main 是 MedMemo 应用入口。
// WailsApp 暴露前端可调用的绑定方法集，供 Wire 注入后绑定到 Wails 运行时。
package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

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
	streamMu         sync.Mutex
	streamCancel     context.CancelFunc
}

// NewWailsApp 构造函数，供 Wire 调用。
func NewWailsApp(
	chat *usecase.ChatOrchestrator,
	mem *usecase.MemoryRetriever,
	cfg *entity.AppConfig,
) *WailsApp {
	return &WailsApp{
		chatOrchestrator: chat,
		memoryRetriever:  mem,
		config:           cfg,
	}
}

// Startup 是 Wails 启动回调，在前端加载完成后调用。
func (a *WailsApp) Startup(ctx context.Context) {
	a.ctx = ctx
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

	err := a.chatOrchestrator.StreamExecute(ctx, chatReq, func(chunk string) {
		runtime.EventsEmit(a.ctx, "chat:stream:token", chunk)
	})

	if err != nil {
		if errors.Is(err, context.Canceled) {
			runtime.EventsEmit(a.ctx, "chat:stream:interrupted", nil)
			return nil
		}
		runtime.EventsEmit(a.ctx, "chat:stream:error", err.Error())
		return fmt.Errorf("stream failed: %w", err)
	}

	runtime.EventsEmit(a.ctx, "chat:stream:end", nil)
	return nil
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
	// TODO(作者): 接入 ConversationRepository [Issue#031]
	return []ConversationSummary{}, nil
}

// CreateConversation 创建新会话，返回会话 ID。
func (a *WailsApp) CreateConversation() (string, error) {
	conv := entity.NewConversation(models.ProviderType(a.config.DefaultModel))
	// TODO(作者): 持久化到 ConversationRepository [Issue#027]
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
func (a *WailsApp) CheckEmergency(text string) (*EmergencyResult, error) {
	// TODO(作者): 接入紧急症状检测引擎 [Issue#028]
	// 当前返回未检测到，避免阻断前端框架验证。
	return &EmergencyResult{
		Level:   "none",
		Message: "",
		Action:  "",
	}, nil
}

// ShowEmergencyDialog 触发紧急症状弹窗（供前端调用）。
func (a *WailsApp) ShowEmergencyDialog(title, message string) {
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.WarningDialog,
		Title:   title,
		Message: message,
	})
}
