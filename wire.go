//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/medmemo/medmemo/internal/adapters/ai"
	"github.com/medmemo/medmemo/internal/adapters/detector"
	"github.com/medmemo/medmemo/internal/adapters/repository"
	adapterUpdater "github.com/medmemo/medmemo/internal/adapters/updater"
	"github.com/medmemo/medmemo/internal/application/pipeline"
	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/internal/application/updater"
	"github.com/medmemo/medmemo/internal/application/usecase"
	"github.com/medmemo/medmemo/internal/infrastructure/config"
	"github.com/medmemo/medmemo/internal/infrastructure/database"
	"github.com/medmemo/medmemo/internal/infrastructure/onnx"
	"github.com/medmemo/medmemo/internal/infrastructure/secret"
	infraUpdater "github.com/medmemo/medmemo/internal/infrastructure/updater"
)

// InitializeApp 通过 Wire 编译期依赖注入组装完整应用。
// 返回 (func()) 作为资源清理回调，由 main 函数通过 defer 调用。
func InitializeApp() (*App, func(), error) {
	wire.Build(
		NewApp,
		NewWailsApp,
		usecase.ApplicationSet,
		usecase.NewMemoryRetriever,
		wire.Bind(new(port.MemoryRepository), new(*repository.MemoryRepoSQLite)),
		wire.Bind(new(port.SensitiveDetector), new(*detector.RuleDetector)),
		wire.Bind(new(usecase.ComplianceChecker), new(*usecase.RuleComplianceChecker)),
		wire.Bind(new(port.ConversationRepository), new(*repository.ConversationRepoSQLite)),
		wire.Bind(new(port.MessageRepository), new(*repository.MessageRepoSQLite)),
		wire.Bind(new(port.DisclaimerRepository), new(*repository.DisclaimerRepoSQLite)),
		wire.Bind(new(port.ProviderStore), new(*repository.ProviderRepoSQLite)),
		ai.ProviderSet,
		repository.RepositorySet,
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
		wire.Bind(new(port.NERDetector), new(*detector.ONNXNERDetector)),
		wire.Bind(new(secret.Store), new(*secret.KeyringStore)),
		wire.Bind(new(database.DBConnector), new(*database.SQLCipherConnector)),
		wire.Bind(new(usecase.Deidentifier), new(*pipeline.DeidentifyPipeline)),
		wire.Bind(new(usecase.MemoryQuerier), new(*usecase.MemoryRetriever)),
		wire.Value(""),
	)
	return nil, nil, nil
}
