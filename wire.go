//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/medmemo/medmemo/internal/adapters/ai"
	"github.com/medmemo/medmemo/internal/adapters/detector"
	"github.com/medmemo/medmemo/internal/adapters/repository"
	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/internal/application/usecase"
	"github.com/medmemo/medmemo/internal/infrastructure/config"
	"github.com/medmemo/medmemo/internal/infrastructure/database"
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
		wire.Bind(new(usecase.ComplianceChecker), new(*usecase.DefaultComplianceChecker)),
		wire.Bind(new(port.ConversationRepository), new(*repository.ConversationRepoSQLite)),
		wire.Bind(new(port.MessageRepository), new(*repository.MessageRepoSQLite)),
		ai.ProviderSet,
		repository.RepositorySet,
		detector.ProviderSet,
		config.ConfigSet,
		database.DatabaseSet,
		wire.Value(""),
	)
	return nil, nil, nil
}
