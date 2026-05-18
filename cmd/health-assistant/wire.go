//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/medmemo/medmemo/internal/adapters/ai"
	"github.com/medmemo/medmemo/internal/adapters/repository"
	"github.com/medmemo/medmemo/internal/application/pipeline"
	"github.com/medmemo/medmemo/internal/application/usecase"
	"github.com/medmemo/medmemo/internal/infrastructure/config"
	"github.com/medmemo/medmemo/internal/infrastructure/database"
	"github.com/medmemo/medmemo/internal/infrastructure/network"
	"github.com/medmemo/medmemo/internal/infrastructure/onnx"
	"github.com/medmemo/medmemo/internal/infrastructure/secret"
)

// InitializeApp 通过 Wire 编译期依赖注入组装完整应用。
// 返回 (func()) 作为资源清理回调，由 main 函数通过 defer 调用。
func InitializeApp() (*App, func(), error) {
	wire.Build(
		NewApp,
		usecase.ApplicationSet,
		pipeline.PipelineSet,
		ai.AIAdapterSet,
		repository.RepositorySet,
		onnx.ONNXSet,
		database.DatabaseSet,
		config.ConfigSet,
		secret.SecretSet,
		network.NetworkSet,
	)
	return nil, nil, nil
}
