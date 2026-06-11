// Package main 是 MedMemo 应用入口。
// WailsApp 暴露前端可调用的绑定方法集，供 Wire 注入后绑定到 Wails 运行时。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hzhan516/medmemo/internal/adapters/ai"
	"github.com/hzhan516/medmemo/internal/adapters/auth"
	"github.com/hzhan516/medmemo/internal/application"
	"github.com/hzhan516/medmemo/internal/application/feedback"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/application/stream"
	"github.com/hzhan516/medmemo/internal/application/updater"
	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/hzhan516/medmemo/internal/infrastructure/config"
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

// Startup 是 Wails 启动回调，在前端加载完成后调用。
func (a *WailsApp) Startup(ctx context.Context) {
	a.ctx = ctx

	// 校验前端嵌入资源是否完整（编译时 embed，运行时读取）
	if err := validateEmbeddedAssets(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load frontend assets: %v\n", err)
		os.Exit(1)
	}

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
		// 将配置文件中的更新通道同步到 updater 服务
		a.updaterSvc.SetSettings(&entity.UpdateSettings{
			CheckEnabled: a.config.UpdateCheckEnabled,
			Channel:      a.config.UpdateChannel,
			SkipVersion:  "",
		})
		go a.checkUpdateAsync()
	}

	// 执行数据留存自动清理（后台 goroutine，不阻塞启动）
	go a.runRetentionCleanup()

	// 启动时扫描 cli_token / oauth_device provider，安排自动刷新
	if a.tokenRefreshSvc != nil {
		go a.scheduleAutoRefreshesAsync()
	}

	// ONNX 预热：启动后异步执行一次 dummy 推理，将 warmup 成本从首次对话转移到启动阶段
	go a.warmupONNX()

	// v1.1.4: 历史 embedding 迁移（版本升级后首次启动触发）
	go a.runEmbeddingMigration()

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
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to check update: %v\n", err)
		return
	}
	if info == nil {
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
	ProviderID     string           `json:"provider_id"`
}

// SendMessageResponse 发送消息响应。
type SendMessageResponse struct {
	Reply            string                 `json:"reply"`
	ConfidenceResult map[string]interface{} `json:"confidence_result"`
	Warnings         []string               `json:"warnings"`
}

const (
	defaultChatTimeout   = 30 * time.Second
	defaultStreamTimeout = 5 * time.Minute
	minStreamTimeout     = 120 * time.Second
)

// SendMessage 发送对话消息，编排完整对话流程（非流式）。
func (a *WailsApp) SendMessage(req SendMessageRequest) (*SendMessageResponse, error) {
	ctx, cancel := context.WithTimeout(a.ctx, defaultChatTimeout)
	defer cancel()

	chatReq := usecase.ChatRequest{
		ConversationID: models.ConversationID(req.ConversationID),
		Messages:       req.Messages,
		Model:          models.ProviderType(req.Model),
		ProviderID:     req.ProviderID,
	}

	resp, err := a.chatOrchestrator.Execute(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	confJSON := map[string]interface{}{}
	if resp.ConfidenceResult != nil {
		confJSON = confidenceResultToMap(resp.ConfidenceResult)
	}
	// 保存用户消息（非流式路径）
	if len(req.Messages) > 0 {
		lastUser := req.Messages[len(req.Messages)-1]
		if lastUser.Role == models.RoleUser {
			a.saveUserMessage(ctx, req.ConversationID, lastUser)
		}
	}
	// 保存 AI 回复
	a.saveMessages(ctx, req.ConversationID, req.Messages, resp.Reply, nil, resp.ConfidenceResult, req.ProviderID)

	return &SendMessageResponse{
		Reply:            resp.Reply,
		ConfidenceResult: confJSON,
		Warnings:         resp.Warnings,
	}, nil
}

// SendMessageStream 发送流式对话请求，通过 Wails Events 实时推送结构化 StreamChunk。
func (a *WailsApp) SendMessageStream(req SendMessageRequest) (err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[panic] SendMessageStream: %v\n", r)
			err = fmt.Errorf("stream internal error: %v", r)
		}
	}()
	// 诊断日志：检查 a.ctx 和 stream context 的 deadline
	streamTimeoutVal := a.streamTimeout(req.ProviderID)
	if d, ok := a.ctx.Deadline(); ok {
		fmt.Printf("[DIAG][Wails] a.ctx deadline=%v remaining=%v\n", d, time.Until(d))
	} else {
		fmt.Printf("[DIAG][Wails] a.ctx has NO deadline\n")
	}
	fmt.Printf("[DIAG][Wails] streamTimeout=%v providerID=%s\n", streamTimeoutVal, req.ProviderID)

	ctx, cancel := context.WithTimeout(a.ctx, streamTimeoutVal)
	if d, ok := ctx.Deadline(); ok {
		fmt.Printf("[DIAG][Wails] stream ctx deadline=%v remaining=%v\n", d, time.Until(d))
	} else {
		fmt.Printf("[DIAG][Wails] stream ctx has NO deadline\n")
	}
	if err := ctx.Err(); err != nil {
		fmt.Printf("[DIAG][Wails] WARNING: stream ctx already expired: %v\n", err)
	}

	a.streamMu.Lock()
	a.activeStreams[req.ConversationID] = cancel
	a.streamMu.Unlock()

	defer func() {
		a.streamMu.Lock()
		delete(a.activeStreams, req.ConversationID)
		a.streamMu.Unlock()
		cancel()
	}()

	chatReq := usecase.ChatRequest{
		ConversationID: models.ConversationID(req.ConversationID),
		Messages:       req.Messages,
		Model:          models.ProviderType(req.Model),
		ProviderID:     req.ProviderID,
	}

	// 统一流式处理层：将原始 callback 包装为结构化 StreamChunk 序列
	broker := stream.NewBroker(req.Model, "", func(chunk models.StreamChunk) {
		chunk.Metadata.ConversationID = req.ConversationID
		runtime.EventsEmit(a.ctx, "chat:stream_chunk", chunk)
	})
	broker.Start()

	// 立即保存用户消息，确保切换会话时可见（不阻塞流式生成）
	if len(req.Messages) > 0 {
		lastUser := req.Messages[len(req.Messages)-1]
		if lastUser.Role == models.RoleUser {
			// 使用应用生命周期 context 异步保存，避免占用 stream 超时 budget
			go a.saveUserMessage(a.ctx, req.ConversationID, lastUser)
		}
	}

	// 收集 AI 完整回复用于持久化
	var fullReply stringsBuilder

	start := time.Now()
	usage, confidenceResult, finalContent, err := a.chatOrchestrator.StreamExecute(ctx, chatReq, func(chunk string) {
		fullReply.WriteString(chunk)
		broker.Content(chunk)
	})

	if err != nil {
		if errors.Is(err, context.Canceled) {
			broker.Error("生成已中断")
			// 保存已生成的部分内容（无置信度，token 为 0）
			a.saveMessages(ctx, req.ConversationID, req.Messages, fullReply.String(), nil, nil, req.ProviderID)
			return nil
		}
		broker.Error(err.Error())
		return fmt.Errorf("stream failed: %w", err)
	}

	// 若最终内容与流式过程中展示的不同（脱敏还原或合规替换），通知前端替换
	if finalContent != fullReply.String() {
		runtime.EventsEmit(a.ctx, "chat:stream:replace", map[string]any{
			"conversation_id": req.ConversationID,
			"content":         finalContent,
		})
	}

	// 保存用户消息和 AI 回复（携带 token 与置信度）
	a.saveMessages(ctx, req.ConversationID, req.Messages, finalContent, usage, confidenceResult, req.ProviderID)

	fmt.Printf("[Stream] total execution time: %v\n", time.Since(start))

	// 流式结束后对完整内容做一次合规检测（MVP 简化策略）
	compResult, compErr := a.chatOrchestrator.CheckCompliance(ctx, finalContent)
	if compErr == nil && compResult.Level != "L4_NORMAL" {
		payload := map[string]any{
			"conversation_id": req.ConversationID,
			"level":           compResult.Level,
			"warning":         compResult.Warning,
			"notice":          compResult.Notice,
			"replacedTerms":   compResult.ReplacedTerms,
			"matchedRule":     compResult.MatchedRule,
		}
		runtime.EventsEmit(a.ctx, "chat:stream:compliance", payload)
	}

	// 推送置信度与 Token 用量事件
	if confidenceResult != nil {
		confidencePayload := map[string]any{
			"conversation_id":   req.ConversationID,
			"confidence":        confidenceResultToMap(confidenceResult),
			"prompt_tokens":     0,
			"completion_tokens": 0,
			"total_tokens":      0,
			"truncated":         false,
		}
		if usage != nil {
			confidencePayload["prompt_tokens"] = usage.PromptTokens
			confidencePayload["completion_tokens"] = usage.CompletionTokens
			confidencePayload["total_tokens"] = usage.TotalTokens
			confidencePayload["truncated"] = usage.FinishReason == "length"
		}
		runtime.EventsEmit(a.ctx, "chat:stream:confidence", confidencePayload)
	}

	broker.Done(usage)
	return nil
}

func (a *WailsApp) streamTimeout(providerID string) time.Duration {
	timeout := defaultStreamTimeout
	if a.providerStore == nil || providerID == "" {
		fmt.Printf("[DIAG][streamTimeout] using default=%v (providerStore=nil=%v providerID=empty=%v)\n",
			timeout, a.providerStore == nil, providerID == "")
		return timeout
	}

	ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
	defer cancel()

	provider, err := a.providerStore.Get(ctx, providerID)
	if err != nil || provider == nil {
		fmt.Printf("[DIAG][streamTimeout] using default=%v (get provider err=%v provider=nil=%v)\n",
			timeout, err, provider == nil)
		return timeout
	}
	if provider.TimeoutMs <= 0 {
		fmt.Printf("[DIAG][streamTimeout] using default=%v (TimeoutMs=%d <= 0)\n", timeout, provider.TimeoutMs)
		return timeout
	}

	configured := time.Duration(provider.TimeoutMs) * time.Millisecond
	if configured < minStreamTimeout {
		fmt.Printf("[DIAG][streamTimeout] using minStreamTimeout=%v (configured=%v < min=%v)\n",
			minStreamTimeout, configured, minStreamTimeout)
		return minStreamTimeout
	}
	fmt.Printf("[DIAG][streamTimeout] using configured=%v (TimeoutMs=%d)\n", configured, provider.TimeoutMs)
	return configured
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

// saveUserMessage 单独保存一条用户消息并更新会话时间戳，
// 用于流式生成启动前立即持久化，确保切换会话时可见。
func (a *WailsApp) saveUserMessage(ctx context.Context, convID string, message models.Message) {
	if a.msgRepo == nil || convID == "" || message.Role != models.RoleUser {
		return
	}
	msgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	if err := a.msgRepo.Save(ctx, models.ConversationID(convID), &entity.Message{
		ID:        msgID,
		Role:      message.Role,
		Content:   message.Content,
		Timestamp: time.Now(),
	}); err != nil {
		fmt.Printf("[saveUserMessage] 保存用户消息失败: %v\n", err)
	}
	// 同步归档到 raw_dialogues，为事实提取提供源数据
	if a.dialogueRepo != nil {
		_ = a.dialogueRepo.Insert(ctx, entity.NewRawDialogue(convID, entity.RoleUser, message.Content, ""))
	}
	if a.convRepo != nil {
		if err := a.convRepo.UpdateTimestamp(ctx, models.ConversationID(convID), time.Now()); err != nil {
			fmt.Printf("[saveUserMessage] 更新会话时间失败: %v\n", err)
		}
	}
}

// saveMessages 保存 AI 回复消息到数据库，可选携带 token 用量与置信度。
// 用户消息应由调用方通过 saveUserMessage 单独保存，避免重复。
// providerID 用于异步事实提取时创建对应的 LLM client。
func (a *WailsApp) saveMessages(ctx context.Context, convID string, messages []models.Message, aiReply string, usage *models.TokenUsage, confidence *entity.ConfidenceResult, providerID string) {
	if a.msgRepo == nil || convID == "" {
		return
	}
	// 保存 AI 回复
	if aiReply != "" {
		msg := &entity.Message{
			ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
			Role:      models.RoleAssistant,
			Content:   aiReply,
			Timestamp: time.Now(),
		}
		if usage != nil {
			msg.PromptTokens = usage.PromptTokens
			msg.CompletionTokens = usage.CompletionTokens
		}
		if confidence != nil {
			msg.ConfidenceScore = confidence.OverallScore
			msg.ConfidenceLevel = string(confidence.Level)
			if confMap := confidenceResultToMap(confidence); confMap != nil {
				if jsonBytes, err := json.Marshal(confMap); err == nil {
					msg.ConfidenceJSON = string(jsonBytes)
				}
			}
			fmt.Printf("[DIAG][Confidence][saveMessages] OverallScore=%.2f Level=%s Breakdown=%v JSON=%s\n",
				confidence.OverallScore, confidence.Level, confidence.Breakdown, msg.ConfidenceJSON)
		}
		if err := a.msgRepo.Save(ctx, models.ConversationID(convID), msg); err != nil {
			fmt.Printf("[saveMessages] 保存 AI 回复失败: %v\n", err)
		}
		// 同步归档 AI 回复到 raw_dialogues
		if a.dialogueRepo != nil {
			_ = a.dialogueRepo.Insert(ctx, entity.NewRawDialogue(convID, entity.RoleAssistant, aiReply, ""))
		}
		// 异步执行事实提取（不阻塞主流程）
		if a.chatOrchestrator != nil && providerID != "" {
			var userContent string
			if len(messages) > 0 {
				lastUser := messages[len(messages)-1]
				if lastUser.Role == models.RoleUser {
					userContent = lastUser.Content
				}
			}
			go a.extractFactsAsync(userContent, aiReply, providerID)
		}
	}
	// 更新会话时间
	if a.convRepo != nil {
		if err := a.convRepo.UpdateTimestamp(ctx, models.ConversationID(convID), time.Now()); err != nil {
			fmt.Printf("[saveMessages] 更新会话时间失败: %v\n", err)
		}
	}
}

// warmupONNX 执行一次 dummy 推理，触发 ONNX Runtime 的首次模型 warmup。
// 使用应用生命周期 context，不设置短 timeout，让 warmup 自然完成。
// 失败仅记录日志，不影响应用启动。
func (a *WailsApp) warmupONNX() {
	defer a.onnxOnce.Do(func() { close(a.onnxReady) })

	// 延迟 2 秒，确保 ONNX engine 已完全初始化
	time.Sleep(2 * time.Second)

	start := time.Now()
	if a.embeddingSvc != nil {
		_, err := a.embeddingSvc.EmbedSingle(a.ctx, "warmup")
		if err != nil {
			fmt.Printf("[ONNX Warmup] embedding 预热失败: %v\n", err)
		} else {
			fmt.Printf("[ONNX Warmup] embedding 预热完成，耗时 %v\n", time.Since(start))
		}
	}
}

func (a *WailsApp) waitForONNXReady() {
	<-a.onnxReady
}

// extractFactsAsync 异步从完整对话（用户消息 + AI 回复）中提取事实并保存到 factRepo。
// 由 ChatOrchestrator 统一调度限流，避免与主对话竞争 API 配额触发 429。
func (a *WailsApp) extractFactsAsync(userContent, aiReply, providerID string) {
	ctx, cancel := context.WithTimeout(a.ctx, 60*time.Second)
	defer cancel()
	facts, err := a.chatOrchestrator.ExtractFactsFromReply(ctx, userContent, aiReply, providerID)
	if err != nil {
		// 429 限流时静默跳过，不记录错误（这是预期行为）
		if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "rate limit") || strings.Contains(err.Error(), "rate limited") {
			fmt.Printf("[extractFactsAsync] skipped due to rate limit (expected)\n")
			return
		}
		fmt.Printf("[extractFactsAsync] 事实提取失败: %v\n", err)
		return
	}
	for _, f := range facts {
		if err := a.factRepo.Save(ctx, f); err != nil {
			fmt.Printf("[extractFactsAsync] 保存事实失败 %s: %v\n", f.FactID, err)
			continue
		}
		// pending fact 不生成 embedding，embedding 只在审批通过后生成，
		// 避免未审核或后续被拒绝的事实污染向量召回空间。
	}
	fmt.Printf("[extractFactsAsync] 提取并保存 %d 条待审核事实\n", len(facts))
}

// safeEventsEmit 安全地发射 Wails 事件，在测试环境（标准 context.Background）下静默跳过。
func (a *WailsApp) safeEventsEmit(eventName string, data ...any) {
	if a.ctx == nil || a.ctx == context.Background() || a.ctx == context.TODO() {
		return
	}
	runtime.EventsEmit(a.ctx, eventName, data...)
}

// runEmbeddingMigration 在 ONNX warmup 后执行 embedding 版本迁移。
func (a *WailsApp) runEmbeddingMigration() {
	a.waitForONNXReady()

	if a.migrator == nil || !a.embeddingSvc.IsAvailable() {
		if a.migrationState != nil {
			a.migrationState.SetComplete(true)
		}
		return
	}

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Minute)
	defer cancel()

	needs, total, err := a.migrator.NeedsMigration(ctx)
	if err != nil || !needs {
		a.migrationState.SetComplete(true)
		return
	}

	a.safeEventsEmit("embedding:migration:start", map[string]any{
		"total": total,
	})

	processed, failed, err := a.migrator.RunMigration(ctx, func(p, t int) {
		a.safeEventsEmit("embedding:migration:progress", map[string]any{
			"processed": p,
			"total":     t,
		})
	})

	a.safeEventsEmit("embedding:migration:done", map[string]any{
		"processed": processed,
		"failed":    failed,
	})

	if err != nil {
		fmt.Printf("[EmbeddingMigration] 迁移异常: %v\n", err)
	}
	if failed > 0 {
		fmt.Printf("[EmbeddingMigration] %d 条 fact 迁移失败（已记录日志）\n", failed)
	}
	fmt.Printf("[EmbeddingMigration] 迁移完成：处理 %d 条，失败 %d 条\n", processed, failed)
}

// StopGeneration 中断所有正在进行的流式生成。
func (a *WailsApp) StopGeneration() {
	a.streamMu.Lock()
	for _, cancel := range a.activeStreams {
		if cancel != nil {
			cancel()
		}
	}
	a.streamMu.Unlock()
}

// ConversationSummary 会话摘要。
type ConversationSummary struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	UpdatedAt string `json:"updated_at"`
	DeletedAt string `json:"deleted_at"` // 空字符串表示未删除，否则为软删除时间戳（毫秒）
}

// MessageResponse 单条消息响应。
type MessageResponse struct {
	ID               string                 `json:"id"`
	Role             string                 `json:"role"`
	Content          string                 `json:"content"`
	Timestamp        string                 `json:"timestamp"`
	PromptTokens     int                    `json:"prompt_tokens"`
	CompletionTokens int                    `json:"completion_tokens"`
	TotalTokens      int                    `json:"total_tokens"`
	Confidence       map[string]interface{} `json:"confidence,omitempty"`
}

// GetConversationMessages 获取指定会话的全部消息。
func (a *WailsApp) GetConversationMessages(convID string) ([]MessageResponse, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.msgRepo == nil {
		return nil, fmt.Errorf("message repository not initialized")
	}

	msgs, _, err := a.msgRepo.ListByConversation(ctx, models.ConversationID(convID), "", 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages for conversation %s: %w", convID, err)
	}

	if len(msgs) == 0 {
		fmt.Printf("[GetConversationMessages] 会话 %s 无消息\n", convID)
	}

	// ListByConversation 返回 created_at DESC（最新的在前），需要反转为正序
	result := make([]MessageResponse, len(msgs))
	for i, m := range msgs {
		mr := MessageResponse{
			ID:               m.ID,
			Role:             string(m.Role),
			Content:          m.Content,
			Timestamp:        strconv.FormatInt(m.Timestamp.UnixMilli(), 10),
			PromptTokens:     m.PromptTokens,
			CompletionTokens: m.CompletionTokens,
			TotalTokens:      m.PromptTokens + m.CompletionTokens,
		}
		if m.ConfidenceJSON != "" {
			// 先反序列化到实体结构（兼容旧数据的大驼峰和新数据的蛇形）
			var cr entity.ConfidenceResult
			if err := json.Unmarshal([]byte(m.ConfidenceJSON), &cr); err == nil {
				mr.Confidence = confidenceResultToMap(&cr)
				// 防御性修复：若 JSON 解析后 overall_score 为 0 但 ConfidenceScore > 0，用 ConfidenceScore 覆盖
				if cr.OverallScore == 0 && m.ConfidenceScore > 0 {
					fmt.Printf("[DIAG][Confidence][GetConversationMessages] OverallScore=0 but ConfidenceScore=%.2f, using ConfidenceScore as fallback. JSON=%s\n",
						m.ConfidenceScore, m.ConfidenceJSON)
					mr.Confidence["overall_score"] = m.ConfidenceScore
				}
			} else {
				// 降级：直接作为 map 透传
				var conf map[string]interface{}
				if err := json.Unmarshal([]byte(m.ConfidenceJSON), &conf); err == nil {
					mr.Confidence = conf
					fmt.Printf("[DIAG][Confidence][GetConversationMessages] fallback map parse. JSON=%s\n", m.ConfidenceJSON)
				}
			}
			fmt.Printf("[DIAG][Confidence][GetConversationMessages] msgID=%s overall_score=%v breakdown=%v\n",
				m.ID, mr.Confidence["overall_score"], mr.Confidence["breakdown"])
		}
		result[len(msgs)-1-i] = mr
	}
	return result, nil
}

// GetConversations 获取会话列表。
func (a *WailsApp) GetConversations() (result []ConversationSummary, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[panic] GetConversations: %v\n", r)
			err = fmt.Errorf("internal error: %v", r)
		}
	}()
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.convRepo == nil {
		return nil, fmt.Errorf("conversation repository not initialized")
	}

	convs, err := a.convRepo.ListRecent(ctx, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	result = make([]ConversationSummary, len(convs))
	for i, conv := range convs {
		summary := ConversationSummary{
			ID:        string(conv.ID),
			Title:     conv.Title,
			UpdatedAt: strconv.FormatInt(conv.UpdatedAt.UnixMilli(), 10),
		}
		if conv.DeletedAt != nil {
			summary.DeletedAt = strconv.FormatInt(conv.DeletedAt.UnixMilli(), 10)
		}
		result[i] = summary
	}
	return result, nil
}

// GetDeletedConversations 获取已软删除的会话列表（回收站）。
func (a *WailsApp) GetDeletedConversations() (result []ConversationSummary, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[panic] GetDeletedConversations: %v\n", r)
			err = fmt.Errorf("internal error: %v", r)
		}
	}()
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.convRepo == nil {
		return nil, fmt.Errorf("conversation repository not initialized")
	}

	convs, err := a.convRepo.ListDeleted(ctx, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to list deleted conversations: %w", err)
	}

	result = make([]ConversationSummary, len(convs))
	for i, conv := range convs {
		summary := ConversationSummary{
			ID:        string(conv.ID),
			Title:     conv.Title,
			UpdatedAt: strconv.FormatInt(conv.UpdatedAt.UnixMilli(), 10),
		}
		if conv.DeletedAt != nil {
			summary.DeletedAt = strconv.FormatInt(conv.DeletedAt.UnixMilli(), 10)
		}
		result[i] = summary
	}
	return result, nil
}

// SetDataRetentionDays 设置数据留存期限并持久化到配置文件。
func (a *WailsApp) SetDataRetentionDays(days int) error {
	if days < 0 {
		return fmt.Errorf("data retention days must be non-negative")
	}
	a.config.DataRetentionDays = days
	if err := config.SaveDataRetentionDays(days); err != nil {
		return fmt.Errorf("failed to save data retention days: %w", err)
	}
	return nil
}

// DeleteConversation 软删除指定会话（移入回收站）。
func (a *WailsApp) DeleteConversation(convID string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.convRepo == nil {
		return fmt.Errorf("conversation repository not initialized")
	}
	if err := a.convRepo.Delete(ctx, models.ConversationID(convID)); err != nil {
		return fmt.Errorf("failed to delete conversation %s: %w", convID, err)
	}
	return nil
}

// RestoreConversation 恢复软删除的会话。
func (a *WailsApp) RestoreConversation(convID string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.convRepo == nil {
		return fmt.Errorf("conversation repository not initialized")
	}
	if err := a.convRepo.Restore(ctx, models.ConversationID(convID)); err != nil {
		return fmt.Errorf("failed to restore conversation %s: %w", convID, err)
	}
	return nil
}

// HardDeleteConversation 永久删除指定会话及其消息。
func (a *WailsApp) HardDeleteConversation(convID string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.convRepo == nil {
		return fmt.Errorf("conversation repository not initialized")
	}
	if err := a.convRepo.HardDelete(ctx, models.ConversationID(convID)); err != nil {
		return fmt.Errorf("failed to hard delete conversation %s: %w", convID, err)
	}
	return nil
}

// runRetentionCleanup 执行数据留存自动归档与清理。
func (a *WailsApp) runRetentionCleanup() {
	retentionDays := a.config.DataRetentionDays
	if retentionDays <= 0 {
		return // 永久保留，不执行清理
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	if a.convRepo != nil {
		// 自动归档：主列表中超过期限的会话 → 移入回收站
		if err := a.convRepo.ArchiveOlderThan(ctx, cutoff); err != nil {
			fmt.Printf("[retention] 自动归档失败: %v\n", err)
		}
		// 自动清理：回收站中超过期限的会话 → 物理删除
		if err := a.convRepo.PermanentlyDeleteOlderThan(ctx, cutoff); err != nil {
			fmt.Printf("[retention] 自动清理失败: %v\n", err)
		}
	}
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
			if err := a.convRepo.UpdateTitle(ctx, models.ConversationID(convID), title); err != nil {
				fmt.Printf("[GenerateTitle] 更新标题失败: %v\n", err)
			}
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
	_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
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

// GetVersion 返回当前应用版本号（构建时通过 -ldflags 注入）。
func (a *WailsApp) GetVersion() string {
	return version
}

// CollectSystemInfo 收集当前运行环境信息，供前端展示。
func (a *WailsApp) CollectSystemInfo() (*feedback.SystemInfo, error) {
	reporter := feedback.NewReporter(version, buildTime)
	return reporter.Collect(), nil
}

// OpenGitHubIssue 打开系统浏览器到 GitHub Issue 创建页面，预填日志内容。
// 前端调用后，用户只需在浏览器中点击 Submit 即可创建 Issue。
func (a *WailsApp) OpenGitHubIssue(userDescription string, errorLog string) error {
	reporter := feedback.NewReporter(version, buildTime)
	info := reporter.Collect()

	logContent, err := feedback.ReadAppLogFile("")
	if err != nil {
		// 日志读取失败不影响主流程，仅记录
		logContent = ""
	}

	// 合并显式传入的错误日志与本地日志文件
	combinedLog := errorLog
	if logContent != "" {
		if combinedLog != "" {
			combinedLog += "\n\n--- 本地日志 ---\n" + logContent
		} else {
			combinedLog = logContent
		}
	}

	issueURL := reporter.BuildIssueURL(info, userDescription, combinedLog)
	runtime.BrowserOpenURL(a.ctx, issueURL)
	return nil
}

// GetVersionNotes 返回全部版本提示数据，按版本降序排列（最新在前）。
func (a *WailsApp) GetVersionNotes() []entity.VersionNote {
	notes := make([]entity.VersionNote, len(entity.AllVersionNotes))
	copy(notes, entity.AllVersionNotes)
	// 倒序：最新版本在前
	for i, j := 0, len(notes)-1; i < j; i, j = i+1, j-1 {
		notes[i], notes[j] = notes[j], notes[i]
	}
	return notes
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
		return err
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
func (a *WailsApp) SearchMemories(query string) ([]MemoryItem, error) {
	if err := a.requireAuth(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return nil, fmt.Errorf("fact repository not initialized")
	}

	facts, err := a.factRepo.ListByStatus(ctx, entity.FactStatusApproved, 0, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}

	query = strings.ToLower(strings.TrimSpace(query))
	var items []MemoryItem
	for _, f := range facts {
		if query == "" ||
			strings.Contains(strings.ToLower(f.Subject), query) ||
			strings.Contains(strings.ToLower(f.Predicate), query) ||
			strings.Contains(strings.ToLower(f.Object), query) {
			items = append(items, factToMemoryItem(f))
		}
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
		return err
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
		return err
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
		return err
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
		return err
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
		return err
	}
	if a.memoryRetriever == nil {
		return fmt.Errorf("memory retriever not initialized")
	}
	a.memoryRetriever.SetSessionEnabled(sessionID, enabled)
	return nil
}

// confidenceResultToMap 将 entity.ConfidenceResult 转为 map[string]interface{}，
// 供 Wails JSON 绑定序列化。
func confidenceResultToMap(r *entity.ConfidenceResult) map[string]interface{} {
	if r == nil {
		return nil
	}
	m := map[string]interface{}{
		"overall_score": r.OverallScore,
		"level":         string(r.Level),
		"explanation":   r.Explanation,
		"suggestion":    r.Suggestion,
		"missing_info":  r.MissingInfo,
	}
	if r.Breakdown != nil {
		m["breakdown"] = r.Breakdown
	}
	return m
}

// validateEmbeddedAssets 校验嵌入的前端资源是否完整可用。
// 放在 Startup 中执行而非 main() 开头，避免 Wails binding 生成阶段触发误报。
func validateEmbeddedAssets() error {
	data, err := assets.ReadFile("web/dist/index.html")
	if err != nil {
		return fmt.Errorf("embedded web/dist/index.html missing: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return fmt.Errorf("embedded web/dist/index.html is empty")
	}
	return nil
}
