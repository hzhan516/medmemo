package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/medmemo/medmemo/pkg/models"
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
