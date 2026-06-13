package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_Load_Defaults(t *testing.T) {
	loader := NewLoader("")
	cfg, err := loader.Load()
	require.NoError(t, err)
	assert.NotEmpty(t, cfg.DataDir)
	assert.Equal(t, "kimi-lite", cfg.DefaultModel)
	assert.Equal(t, "zh-CN", cfg.Language)
	assert.True(t, cfg.EnableCloud)
	assert.False(t, cfg.EnableAnalytics)
	assert.Equal(t, models.ProviderKimi, cfg.ProviderType)
}

func TestLoader_Load_FromYAML(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	data := []byte(`
data_dir: /tmp/medmemo-data
default_model: gpt-4o-mini
language: en-US
enable_cloud: false
provider_type: openai
api_endpoint: https://custom.api.com
`)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	loader := NewLoader(configPath)
	cfg, err := loader.Load()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/medmemo-data", cfg.DataDir)
	assert.Equal(t, "gpt-4o-mini", cfg.DefaultModel)
	assert.Equal(t, "en-US", cfg.Language)
	assert.False(t, cfg.EnableCloud)
	assert.Equal(t, models.ProviderOpenAI, cfg.ProviderType)
	assert.Equal(t, "https://custom.api.com", cfg.APIEndpoint)
}

func TestLoader_Load_FromJSON(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	data := []byte(`{
  "data_dir": "/tmp/medmemo-json",
  "default_model": "qwen-turbo",
  "provider_type": "qwen"
}`)
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	loader := NewLoader(configPath)
	cfg, err := loader.Load()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/medmemo-json", cfg.DataDir)
	assert.Equal(t, models.ProviderQwen, cfg.ProviderType)
}

func TestLoader_Load_EnvOverride(t *testing.T) {
	t.Setenv("MEDMEMO_DEFAULT_MODEL", "override-model")
	t.Setenv("MEDMEMO_PROVIDER_TYPE", "siliconflow")

	loader := NewLoader("")
	cfg, err := loader.Load()
	require.NoError(t, err)
	assert.Equal(t, "override-model", cfg.DefaultModel)
	assert.Equal(t, models.ProviderSiliconFlow, cfg.ProviderType)
}

func TestLoader_Load_InvalidPath(t *testing.T) {
	loader := NewLoader("/nonexistent/path/config.yaml")
	_, err := loader.Load()
	require.Error(t, err)
}

func TestExpandTilde(t *testing.T) {
	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, "/test"), expandTilde("~/test"))
	assert.Equal(t, "/absolute/path", expandTilde("/absolute/path"))
}

func TestLoader_Load_InvalidYAML(t *testing.T) {
	// 创建包含非法 YAML 语法的临时文件，验证 Load 返回包含 "yaml unmarshal failed" 的错误
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	data := []byte("data_dir: [unclosed bracket")
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	loader := NewLoader(configPath)
	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "yaml unmarshal failed")
}

func TestLoader_Load_InvalidJSON(t *testing.T) {
	// 创建包含非法 JSON 语法的临时文件，验证 Load 返回包含 "json unmarshal failed" 的错误
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	data := []byte("{invalid json")
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	loader := NewLoader(configPath)
	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "json unmarshal failed")
}

func TestLoader_Load_UnsupportedFormat(t *testing.T) {
	// 创建 .toml 扩展名的临时文件，验证 Load 返回包含 "unsupported config format" 的错误
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	data := []byte("data_dir = \"/tmp\"")
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	loader := NewLoader(configPath)
	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config format")
}

func TestLoader_Load_EnvOverride_DataRetentionDays(t *testing.T) {
	t.Run("有效数值", func(t *testing.T) {
		t.Setenv("MEDMEMO_DATA_RETENTION_DAYS", "90")
		loader := NewLoader("")
		cfg, err := loader.Load()
		require.NoError(t, err)
		assert.Equal(t, 90, cfg.DataRetentionDays)
	})

	t.Run("无效数值回退默认值", func(t *testing.T) {
		t.Setenv("MEDMEMO_DATA_RETENTION_DAYS", "not-a-number")
		loader := NewLoader("")
		cfg, err := loader.Load()
		require.NoError(t, err)
		assert.Equal(t, defaultDataRetentionDays, cfg.DataRetentionDays)
	})
}

func TestLoader_Load_EnvOverride_Boolean(t *testing.T) {
	t.Run("true 字符串", func(t *testing.T) {
		t.Setenv("MEDMEMO_UPDATE_CHECK", "true")
		loader := NewLoader("")
		cfg, err := loader.Load()
		require.NoError(t, err)
		assert.True(t, cfg.UpdateCheckEnabled)
	})

	t.Run("1 字符串", func(t *testing.T) {
		t.Setenv("MEDMEMO_UPDATE_CHECK", "1")
		loader := NewLoader("")
		cfg, err := loader.Load()
		require.NoError(t, err)
		assert.True(t, cfg.UpdateCheckEnabled)
	})

	t.Run("false 字符串", func(t *testing.T) {
		t.Setenv("MEDMEMO_UPDATE_CHECK", "false")
		loader := NewLoader("")
		cfg, err := loader.Load()
		require.NoError(t, err)
		assert.False(t, cfg.UpdateCheckEnabled)
	})
}

func TestLoader_Load_MissingFields_UseDefaults(t *testing.T) {
	// 仅设置 data_dir，验证其余字段均回退到硬编码默认值
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	data := []byte("data_dir: /tmp/medmemo-only\n")
	require.NoError(t, os.WriteFile(configPath, data, 0644))

	loader := NewLoader(configPath)
	cfg, err := loader.Load()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/medmemo-only", cfg.DataDir)
	assert.Equal(t, defaultModel, cfg.DefaultModel)
	assert.Equal(t, defaultLanguage, cfg.Language)
	assert.Equal(t, defaultEnableCloud, cfg.EnableCloud)
	assert.Equal(t, defaultEnableAnalytics, cfg.EnableAnalytics)
	assert.Equal(t, defaultProviderType, cfg.ProviderType)
	assert.Equal(t, defaultModelDir, cfg.ModelDir)
	assert.Equal(t, entity.ChannelBeta, cfg.UpdateChannel)
	assert.Equal(t, models.DesensitizationStandard, cfg.DesensitizationLevel)
	assert.Equal(t, defaultDataRetentionDays, cfg.DataRetentionDays)
}

func TestLoader_loadDefaults(t *testing.T) {
	// 直接调用 loadDefaults，验证返回的 rawConfig 包含所有预期默认值
	loader := NewLoader("")
	raw := loader.loadDefaults()
	require.NotNil(t, raw)
	assert.NotEmpty(t, raw.DataDir)
	assert.Equal(t, defaultModel, raw.DefaultModel)
	assert.Equal(t, defaultLanguage, raw.Language)
	require.NotNil(t, raw.EnableCloud)
	assert.Equal(t, defaultEnableCloud, *raw.EnableCloud)
	require.NotNil(t, raw.EnableAnalytics)
	assert.Equal(t, defaultEnableAnalytics, *raw.EnableAnalytics)
	assert.Equal(t, string(defaultProviderType), raw.ProviderType)
	assert.Empty(t, raw.APIEndpoint)
	assert.Empty(t, raw.APIKeyFile)
	assert.Equal(t, defaultModelDir, raw.ModelDir)
	require.NotNil(t, raw.UpdateCheckEnabled)
	assert.True(t, *raw.UpdateCheckEnabled)
	assert.Equal(t, string(entity.ChannelBeta), raw.UpdateChannel)
	assert.Equal(t, defaultDesensitizationLevel, raw.DesensitizationLevel)
	require.NotNil(t, raw.DataRetentionDays)
	assert.Equal(t, defaultDataRetentionDays, *raw.DataRetentionDays)
	assert.Empty(t, raw.EmbeddingModelDownloadURL)
}
