// Package main 是 MedMemo 应用入口。
// WailsApp 暴露前端可调用的绑定方法集，供 Wire 注入后绑定到 Wails 运行时。
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

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

// --- Onboarding 向导相关绑定方法 ---

// --- OAuth Device Flow 绑定方法 ---

// --- Ollama 本地模型检测与引导 ---

// --- 认证方式智能检测 ---

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
