// Package config 封装配置加载，将外部配置映射为领域 Config 对象。
// 当前使用标准库 encoding/json + gopkg.in/yaml.v3 解析，Viper 待后续引入 [Issue#022]。
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hzhan516/medmemo/pkg/models"
	"gopkg.in/yaml.v3"
)

const (
	defaultDataDir              = "~/.medmemo/data"
	defaultModel                = "kimi-lite"
	defaultLanguage             = "zh-CN"
	defaultEnableCloud          = true
	defaultEnableAnalytics      = false
	defaultProviderType         = models.ProviderKimi
	defaultModelDir             = "resources/models/distilbert-ner"
	defaultDesensitizationLevel = string(models.DesensitizationStandard)
	defaultDataRetentionDays    = 30
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
	DataDir                   string `json:"data_dir" yaml:"data_dir"`
	DefaultModel              string `json:"default_model" yaml:"default_model"`
	Language                  string `json:"language" yaml:"language"`
	EnableCloud               *bool  `json:"enable_cloud" yaml:"enable_cloud"`
	EnableAnalytics           *bool  `json:"enable_analytics" yaml:"enable_analytics"`
	ProviderType              string `json:"provider_type" yaml:"provider_type"`
	APIEndpoint               string `json:"api_endpoint" yaml:"api_endpoint"`
	APIKeyFile                string `json:"api_key_file" yaml:"api_key_file"`
	ModelDir                  string `json:"model_dir" yaml:"model_dir"`
	UpdateCheckEnabled        *bool  `json:"update_check_enabled" yaml:"update_check_enabled"`
	UpdateChannel             string `json:"update_channel" yaml:"update_channel"`
	DesensitizationLevel      string `json:"desensitization_level" yaml:"desensitization_level"`
	DataRetentionDays         *int   `json:"data_retention_days" yaml:"data_retention_days"`
	EmbeddingModelDownloadURL string `json:"embedding_model_download_url" yaml:"embedding_model_download_url"`
}

// Load 加载并校验配置，返回领域层 AppConfig。
// 加载优先级：显式 configPath > ~/.medmemo/config.{yaml|json} > ./config.{yaml|json} > 硬编码默认值。
func (l *Loader) Load() (*models.AppConfig, error) {
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
	updateChannel := string(models.ChannelBeta)
	desensitizationLevel := defaultDesensitizationLevel
	dataDir := ExpandTilde(defaultDataDir)
	if dataDir == "" {
		// 兜底：若无法解析主目录，使用当前工作目录下的 .medmemo/data
		dataDir = ".medmemo/data"
	}
	return &rawConfig{
		DataDir:                   dataDir,
		DefaultModel:              defaultModel,
		Language:                  defaultLanguage,
		EnableCloud:               new(defaultEnableCloud),
		EnableAnalytics:           new(defaultEnableAnalytics),
		ProviderType:              string(defaultProviderType),
		APIEndpoint:               "",
		APIKeyFile:                "",
		ModelDir:                  defaultModelDir,
		UpdateCheckEnabled:        new(true),
		UpdateChannel:             updateChannel,
		DesensitizationLevel:      desensitizationLevel,
		DataRetentionDays:         new(defaultDataRetentionDays),
		EmbeddingModelDownloadURL: "",
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
	if v := os.Getenv("MEDMEMO_UPDATE_CHECK"); v != "" {
		raw.UpdateCheckEnabled = new(v == "true" || v == "1")
	}
	if v := os.Getenv("MEDMEMO_UPDATE_CHANNEL"); v != "" {
		raw.UpdateChannel = v
	}
	if v := os.Getenv("MEDMEMO_DESENSITIZATION_LEVEL"); v != "" {
		raw.DesensitizationLevel = v
	}
	if v := os.Getenv("MEDMEMO_DATA_RETENTION_DAYS"); v != "" {
		if d, err := strconv.Atoi(v); err == nil {
			raw.DataRetentionDays = &d
		}
	}
	if v := os.Getenv("MEDMEMO_EMBEDDING_MODEL_DOWNLOAD_URL"); v != "" {
		raw.EmbeddingModelDownloadURL = v
	}
}

func (l *Loader) toDomain(raw *rawConfig) *models.AppConfig {
	cfg := &models.AppConfig{
		DataDir:                   ExpandTilde(raw.DataDir),
		DefaultModel:              raw.DefaultModel,
		Language:                  raw.Language,
		APIEndpoint:               raw.APIEndpoint,
		ModelDir:                  ExpandTilde(raw.ModelDir),
		UpdateChannel:             models.UpdateChannel(raw.UpdateChannel),
		DesensitizationLevel:      models.DesensitizationLevel(raw.DesensitizationLevel),
		EmbeddingModelDownloadURL: raw.EmbeddingModelDownloadURL,
	}
	if raw.EnableCloud != nil {
		cfg.EnableCloud = *raw.EnableCloud
	}
	if raw.EnableAnalytics != nil {
		cfg.EnableAnalytics = *raw.EnableAnalytics
	}
	if raw.UpdateCheckEnabled != nil {
		cfg.UpdateCheckEnabled = *raw.UpdateCheckEnabled
	}
	cfg.ProviderType = models.ProviderType(raw.ProviderType)
	if cfg.ProviderType == "" {
		cfg.ProviderType = defaultProviderType
	}
	if cfg.UpdateChannel == "" {
		cfg.UpdateChannel = models.ChannelBeta
	}
	if cfg.DesensitizationLevel == "" {
		cfg.DesensitizationLevel = models.DesensitizationStandard
	}
	if raw.DataRetentionDays != nil {
		cfg.DataRetentionDays = *raw.DataRetentionDays
	}
	return cfg
}

// ExpandTilde 将路径中的 ~ 替换为用户主目录。
func ExpandTilde(path string) string {
	if len(path) > 0 && path[0] == '~' {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}

// SaveDataRetentionDays 将数据留存期限持久化到配置文件。
// 优先写入 ~/.medmemo/config.yaml，不丢失文件中已有其他字段。
func SaveDataRetentionDays(days int) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home dir: %w", err)
	}
	path := filepath.Join(home, ".medmemo", "config.yaml")

	// 以 map 读取现有配置，避免丢失其他字段
	var m map[string]interface{}
	if data, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(data, &m)
	}
	if m == nil {
		m = make(map[string]interface{})
	}
	m["data_retention_days"] = days

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}
