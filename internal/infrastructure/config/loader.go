// Package config 封装 Viper 配置加载，将外部配置映射为领域 Config 对象。
package config

import (
	"fmt"

	"github.com/google/wire"
	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/pkg/models"
)

// Loader 配置加载器，负责从文件/环境变量/默认值加载配置。
type Loader struct {
	configPath string
}

// NewLoader 构造函数。
func NewLoader(configPath string) *Loader {
	return &Loader{configPath: configPath}
}

// Load 加载并校验配置，返回领域层 AppConfig。
func (l *Loader) Load() (*entity.AppConfig, error) {
	// TODO(作者): 接入 Viper 加载 YAML/JSON 配置 [Issue#022]
	cfg := &entity.AppConfig{
		DataDir:         "~/.medmemo/data",
		DefaultModel:    "kimi-lite",
		Language:        "zh-CN",
		EnableCloud:     true,
		EnableAnalytics: false,
		ProviderType:    models.ProviderKimi,
		APIEndpoint:     "",
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	return cfg, nil
}

// LoadConfig 从 Loader 加载并返回 AppConfig，供 Wire 注入。
func LoadConfig(loader *Loader) (*entity.AppConfig, error) {
	return loader.Load()
}

// ConfigSet 供 Wire 使用的 ProviderSet。
var ConfigSet = wire.NewSet(
	NewLoader,
	LoadConfig,
)
