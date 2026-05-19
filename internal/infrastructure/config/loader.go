// Package config 封装配置加载，将外部配置映射为领域 Config 对象。
// 当前使用标准库 encoding/json + gopkg.in/yaml.v3 解析，Viper 待后续引入 [Issue#022]。
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/wire"
	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/pkg/models"
	"gopkg.in/yaml.v3"
)

const (
	defaultDataDir         = "~/.medmemo/data"
	defaultModel           = "kimi-lite"
	defaultLanguage        = "zh-CN"
	defaultEnableCloud     = true
	defaultEnableAnalytics = false
	defaultProviderType    = models.ProviderKimi
	defaultModelDir        = "resources/models/distilbert-ner"
)

// Loader 配置加载器，负责从文件/环境变量/默认值加载配置。
type Loader struct {
	configPath string
}

// NewLoader 构造函数。
func NewLoader(configPath string) *Loader {
	return &Loader{configPath: configPath}
}

// rawConfig 表示配置文件中的原始结构。
type rawConfig struct {
	DataDir         string `json:"data_dir" yaml:"data_dir"`
	DefaultModel    string `json:"default_model" yaml:"default_model"`
	Language        string `json:"language" yaml:"language"`
	EnableCloud     *bool  `json:"enable_cloud" yaml:"enable_cloud"`
	EnableAnalytics *bool  `json:"enable_analytics" yaml:"enable_analytics"`
	ProviderType    string `json:"provider_type" yaml:"provider_type"`
	APIEndpoint     string `json:"api_endpoint" yaml:"api_endpoint"`
	APIKeyFile      string `json:"api_key_file" yaml:"api_key_file"`
	ModelDir        string `json:"model_dir" yaml:"model_dir"`
}

// Load 加载并校验配置，返回领域层 AppConfig。
// 加载优先级：显式 configPath > ~/.medmemo/config.{yaml|json} > ./config.{yaml|json} > 硬编码默认值。
func (l *Loader) Load() (*entity.AppConfig, error) {
	raw := l.loadDefaults()

	// 尝试从文件加载
	if l.configPath != "" {
		if err := l.loadFromFile(l.configPath, raw); err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", l.configPath, err)
		}
	} else {
		// 按优先级搜索配置文件
		paths := l.searchPaths()
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				if err := l.loadFromFile(p, raw); err != nil {
					return nil, fmt.Errorf("failed to load config from %s: %w", p, err)
				}
				break
			}
		}
	}

	// 环境变量覆盖
	l.applyEnvOverrides(raw)

	cfg := l.toDomain(raw)
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	return cfg, nil
}

func (l *Loader) loadDefaults() *rawConfig {
	enableCloud := defaultEnableCloud
	enableAnalytics := defaultEnableAnalytics
	dataDir := expandTilde(defaultDataDir)
	if dataDir == "" {
		// 兜底：若无法解析主目录，使用当前工作目录下的 .medmemo/data
		dataDir = ".medmemo/data"
	}
	return &rawConfig{
		DataDir:         dataDir,
		DefaultModel:    defaultModel,
		Language:        defaultLanguage,
		EnableCloud:     &enableCloud,
		EnableAnalytics: &enableAnalytics,
		ProviderType:    string(defaultProviderType),
		APIEndpoint:     "",
		APIKeyFile:      "",
		ModelDir:        defaultModelDir,
	}
}

func (l *Loader) searchPaths() []string {
	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(home, ".medmemo", "config.yaml"),
		filepath.Join(home, ".medmemo", "config.json"),
		"config.yaml",
		"config.json",
	}
}

func (l *Loader) loadFromFile(path string, raw *rawConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file failed: %w", err)
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, raw); err != nil {
			return fmt.Errorf("yaml unmarshal failed: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, raw); err != nil {
			return fmt.Errorf("json unmarshal failed: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config format: %s", ext)
	}
	return nil
}

func (l *Loader) applyEnvOverrides(raw *rawConfig) {
	if v := os.Getenv("MEDMEMO_DATA_DIR"); v != "" {
		raw.DataDir = v
	}
	if v := os.Getenv("MEDMEMO_DEFAULT_MODEL"); v != "" {
		raw.DefaultModel = v
	}
	if v := os.Getenv("MEDMEMO_PROVIDER_TYPE"); v != "" {
		raw.ProviderType = v
	}
	if v := os.Getenv("MEDMEMO_API_ENDPOINT"); v != "" {
		raw.APIEndpoint = v
	}
	if v := os.Getenv("MEDMEMO_API_KEY_FILE"); v != "" {
		raw.APIKeyFile = v
	}
	if v := os.Getenv("MEDMEMO_MODEL_DIR"); v != "" {
		raw.ModelDir = v
	}
}

func (l *Loader) toDomain(raw *rawConfig) *entity.AppConfig {
	cfg := &entity.AppConfig{
		DataDir:      expandTilde(raw.DataDir),
		DefaultModel: raw.DefaultModel,
		Language:     raw.Language,
		APIEndpoint:  raw.APIEndpoint,
		ModelDir:     expandTilde(raw.ModelDir),
	}
	if raw.EnableCloud != nil {
		cfg.EnableCloud = *raw.EnableCloud
	}
	if raw.EnableAnalytics != nil {
		cfg.EnableAnalytics = *raw.EnableAnalytics
	}
	cfg.ProviderType = models.ProviderType(raw.ProviderType)
	if cfg.ProviderType == "" {
		cfg.ProviderType = defaultProviderType
	}
	return cfg
}

// LoadConfig 从 Loader 加载并返回 AppConfig，供 Wire 注入。
func LoadConfig(loader *Loader) (*entity.AppConfig, error) {
	return loader.Load()
}

// expandTilde 将路径中的 ~ 替换为用户主目录。
func expandTilde(path string) string {
	if len(path) > 0 && path[0] == '~' {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}

// ConfigSet 供 Wire 使用的 ProviderSet。
var ConfigSet = wire.NewSet(
	NewLoader,
	LoadConfig,
)
