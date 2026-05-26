package ai

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
)

func TestProviderFactory_Ollama(t *testing.T) {
	cfg := &entity.AppConfig{
		ProviderType: models.ProviderOllama,
	}
	client := ProviderFactory(cfg)
	assert.NotNil(t, client)
}

func TestProviderFactory_Local(t *testing.T) {
	cfg := &entity.AppConfig{
		ProviderType: models.ProviderLocal,
	}
	client := ProviderFactory(cfg)
	assert.NotNil(t, client)
}

func TestProviderFactory_Cloud(t *testing.T) {
	cfg := &entity.AppConfig{
		ProviderType: models.ProviderKimi,
	}
	client := ProviderFactory(cfg)
	assert.NotNil(t, client)
}

func TestProviderFactory_Cloud_WithEndpoint(t *testing.T) {
	cfg := &entity.AppConfig{
		ProviderType: models.ProviderOpenAI,
		APIEndpoint:  "https://custom.api.com",
	}
	client := ProviderFactory(cfg)
	assert.NotNil(t, client)
}

func TestNewLLMClientFactory(t *testing.T) {
	factory := NewLLMClientFactory()
	assert.NotNil(t, factory)
}

func TestInferProviderTypeFromHost(t *testing.T) {
	assert.Equal(t, models.ProviderKimi, inferProviderTypeFromHost("https://api.moonshot.cn"))
	assert.Equal(t, models.ProviderOpenAI, inferProviderTypeFromHost("https://api.openai.com"))
	assert.Equal(t, models.ProviderOllama, inferProviderTypeFromHost("http://localhost:11434"))
	assert.Equal(t, models.ProviderKimi, inferProviderTypeFromHost("unknown.host"))
}

func TestResolveTimeout(t *testing.T) {
	assert.Equal(t, 5*time.Second, resolveTimeout(5000))
	assert.Equal(t, 120*time.Second, resolveTimeout(0))
}

func TestResolveEndpoint(t *testing.T) {
	assert.Equal(t, "custom", resolveEndpoint("custom", "default"))
	assert.Equal(t, "default", resolveEndpoint("", "default"))
}

func TestResolveModel(t *testing.T) {
	assert.Equal(t, "model-a", resolveModel("model-a", nil, "default"))
	assert.Equal(t, "model-b", resolveModel("", []models.ProviderModel{{ID: "model-b"}}, "default"))
	assert.Equal(t, "default", resolveModel("", nil, "default"))
}

func TestLLMClientFactory_CreateClient(t *testing.T) {
	factory := NewLLMClientFactory()

	// nil config
	_, err := factory.CreateClient(nil)
	assert.Error(t, err)

	// cloud client
	cfg := &models.ProviderConfig{
		APIHost: "https://api.moonshot.cn",
		ModelID: "kimi-v1",
	}
	client, err := factory.CreateClient(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// ollama client
	cfg2 := &models.ProviderConfig{
		APIHost: "http://localhost:11434",
	}
	client, err = factory.CreateClient(cfg2)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// local client
	cfg3 := &models.ProviderConfig{
		APIHost: "http://localhost:8080",
	}
	client, err = factory.CreateClient(cfg3)
	assert.NoError(t, err)
	assert.NotNil(t, client)
}
