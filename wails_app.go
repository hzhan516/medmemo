// Package main 是 MedMemo 应用入口。
// WailsApp 暴露前端可调用的绑定方法集，供 Wire 注入后绑定到 Wails 运行时。
package main

import (
	"context"
	"sync"

	"github.com/hzhan516/medmemo/internal/adapters/auth"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/application/updater"
	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/hzhan516/medmemo/internal/infrastructure/secret"
	"github.com/hzhan516/medmemo/pkg/models"
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

	// v1.1.9: 回答准确率反馈服务
	accuracyService *usecase.AccuracyService

	// v1.1.9: 知识库管理
	knowledgeRepo     repository.KnowledgeRepository
	knowledgeImporter *usecase.KnowledgeImporter
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
	accuracyService *usecase.AccuracyService,
	knowledgeRepo repository.KnowledgeRepository,
	knowledgeImporter *usecase.KnowledgeImporter,
) *WailsApp {
	return &WailsApp{
		chatOrchestrator:  chat,
		memoryRetriever:   mem,
		config:            cfg,
		convRepo:          convRepo,
		msgRepo:           msgRepo,
		disclaimerRepo:    disclaimerRepo,
		providerStore:     providerStore,
		healthChecker:     healthChecker,
		titleGen:          titleGen,
		updaterSvc:        updaterSvc,
		secretStore:       secretStore,
		tokenRefreshSvc:   tokenRefreshSvc,
		deviceFlowSvc:     deviceFlowSvc,
		factRepo:          factRepo,
		auditLogRepo:      auditLogRepo,
		dialogueRepo:      dialogueRepo,
		embeddingSvc:      embeddingSvc,
		embeddingRepo:     embeddingRepo,
		migrator:          migrator,
		migrationState:    migrationState,
		onnxReady:         make(chan struct{}),
		callbackServers:   make(map[string]*auth.LocalCallbackServer),
		activeStreams:     make(map[string]context.CancelFunc),
		accuracyService:   accuracyService,
		knowledgeRepo:     knowledgeRepo,
		knowledgeImporter: knowledgeImporter,
	}
}

// --- Onboarding 向导相关绑定方法 ---

// --- OAuth Device Flow 绑定方法 ---

// --- Ollama 本地模型检测与引导 ---

// --- 认证方式智能检测 ---

// =============================================================================
// 记忆管理 API（TASK-060）
// =============================================================================
