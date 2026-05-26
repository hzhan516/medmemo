//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/hzhan516/medmemo/internal/adapters/ai"
	"github.com/hzhan516/medmemo/internal/adapters/auth"
	"github.com/hzhan516/medmemo/internal/adapters/detector"
	"github.com/hzhan516/medmemo/internal/adapters/repository"
	adapterUpdater "github.com/hzhan516/medmemo/internal/adapters/updater"
	"github.com/hzhan516/medmemo/internal/application/healthcheck"
	"github.com/hzhan516/medmemo/internal/application/pipeline"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/application/updater"
	"github.com/hzhan516/medmemo/internal/application/usecase"
	domainRepo "github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/hzhan516/medmemo/internal/infrastructure/config"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/hzhan516/medmemo/internal/infrastructure/onnx"
	"github.com/hzhan516/medmemo/internal/infrastructure/secret"
	infraUpdater "github.com/hzhan516/medmemo/internal/infrastructure/updater"
)

// InitializeApp 通过 Wire 编译期依赖注入组装完整应用。
// 返回 (func()) 作为资源清理回调，由 main 函数通过 defer 调用。
func InitializeApp() (*App, func(), error) {
	wire.Build(
		NewApp,
		NewWailsApp,
		usecase.ApplicationSet,
		usecase.NewMemoryRetriever,
		usecase.NewDecayScorer,
		wire.Bind(new(port.MemoryRepository), new(*repository.MemoryRepoSQLite)),
		wire.Bind(new(domainRepo.FactRepository), new(*repository.FactRepoSQLite)),
		wire.Bind(new(domainRepo.EmbeddingRepository), new(*repository.EmbeddingRepoSQLite)),
		wire.Bind(new(domainRepo.AuditLogRepository), new(*repository.AuditLogRepoSQLite)),
		wire.Bind(new(port.SensitiveDetector), new(*detector.RuleDetector)),
		wire.Bind(new(usecase.ComplianceChecker), new(*usecase.RuleComplianceChecker)),
		wire.Bind(new(port.ConversationRepository), new(*repository.ConversationRepoSQLite)),
		wire.Bind(new(port.MessageRepository), new(*repository.MessageRepoSQLite)),
		wire.Bind(new(port.DisclaimerRepository), new(*repository.DisclaimerRepoSQLite)),
		wire.Bind(new(port.ProviderStore), new(*repository.ProviderRepoSQLite)),
		wire.Bind(new(port.HealthChecker), new(*healthcheck.HealthEngine)),
		healthcheck.NewHealthEngine,
		ai.ProviderSet,
		ai.EmbeddingServiceSet,
		auth.TokenRefreshProviderSet,
		auth.OAuthDeviceFlowProviderSet,
		repository.RepositorySet,
		repository.FactRepoSet,
		repository.EmbeddingRepoSet,
		repository.AuditLogRepoSet,
		detector.ProviderSet,
		detector.ONNXNERSet,
		pipeline.PipelineSet,
		config.ConfigSet,
		database.DatabaseSet,
		onnx.ONNXSet,
		secret.SecretSet,
		infraUpdater.InstallerSet,
		adapterUpdater.ProviderSet,
		updater.ProviderSet,
		NewEngineConfig,
		wire.Bind(new(port.EmbeddingService), new(*ai.EmbeddingServiceAdapter)),
		wire.Bind(new(ai.EmbeddingEngine), new(*onnx.Engine)),
		wire.Bind(new(port.NERDetector), new(*detector.ONNXNERDetector)),
		wire.Bind(new(secret.Store), new(*secret.KeyringStore)),
		wire.Bind(new(database.DBConnector), new(*database.SQLCipherConnector)),
		wire.Bind(new(usecase.Deidentifier), new(*pipeline.DeidentifyPipeline)),
		wire.Bind(new(usecase.MemoryQuerier), new(*usecase.MemoryRetriever)),
		wire.Value("all-MiniLM-L6-v2"),
	)
	return nil, nil, nil
}
