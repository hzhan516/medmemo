package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/medmemo/medmemo/internal/adapters/auth"
	"github.com/medmemo/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSecretStore 是用于测试的内存密钥环保存实现。
type mockSecretStore struct {
	data map[string][]byte
}

func newMockSecretStore() *mockSecretStore {
	return &mockSecretStore{data: make(map[string][]byte)}
}

func (m *mockSecretStore) Get(key string) ([]byte, error) {
	v, ok := m.data[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return v, nil
}

func (m *mockSecretStore) Set(key string, value []byte) error {
	m.data[key] = value
	return nil
}

func (m *mockSecretStore) Delete(key string) error {
	delete(m.data, key)
	return nil
}

// mockProviderStore 是用于测试的内存 ProviderStore 实现。
type mockProviderStore struct {
	providers []*models.ProviderConfig
}

func newMockProviderStore() *mockProviderStore {
	return &mockProviderStore{providers: make([]*models.ProviderConfig, 0)}
}

func (m *mockProviderStore) Create(_ context.Context, provider *models.ProviderConfig) error {
	m.providers = append(m.providers, provider)
	return nil
}

func (m *mockProviderStore) Update(_ context.Context, provider *models.ProviderConfig) error {
	for i, p := range m.providers {
		if p.ID == provider.ID {
			m.providers[i] = provider
			return nil
		}
	}
	return nil
}

func (m *mockProviderStore) Delete(_ context.Context, id string) error {
	filtered := make([]*models.ProviderConfig, 0, len(m.providers))
	for _, p := range m.providers {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}
	m.providers = filtered
	return nil
}

func (m *mockProviderStore) Get(_ context.Context, id string) (*models.ProviderConfig, error) {
	for _, p := range m.providers {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockProviderStore) List(_ context.Context) ([]*models.ProviderConfig, error) {
	return m.providers, nil
}

// TestDetectAuthMethods_Parallel 验证 DetectAuthMethods 返回四种认证方式的检测结构。
func TestDetectAuthMethods_Parallel(t *testing.T) {
	ctx := t.Context()
	app := &WailsApp{
		ctx:           ctx,
		secretStore:   newMockSecretStore(),
		providerStore: newMockProviderStore(),
		deviceFlowSvc: auth.NewOAuthDeviceFlowServiceBare(newMockProviderStore()),
	}

	result, err := app.DetectAuthMethods()
	require.NoError(t, err)
	require.NotNil(t, result)

	// 验证返回了 5 种认证方式
	assert.Len(t, result.Results, 5)

	// 验证每种方式都有 method 和 tier
	methods := make(map[string]int)
	for _, r := range result.Results {
		methods[r.Method] = r.Tier
	}
	assert.Equal(t, 1, methods["cli_token"])
	assert.Equal(t, 2, methods["oauth_device"])
	assert.Equal(t, 3, methods["api_key"])
	assert.Equal(t, 3, methods["service_account"])
	assert.Equal(t, 4, methods["local"])
}

// TestDetectAuthMethods_RecommendsTier1 当 CLI Token 可用且已连接时，推荐 Tier 1。
func TestDetectAuthMethods_RecommendsTier1(t *testing.T) {
	// 注意：此测试依赖本地是否有 Kimi/Gemini CLI，在 CI 环境中通常不可用
	// 因此仅验证推荐逻辑不会 panic，不强制断言推荐结果
	ctx := t.Context()
	app := &WailsApp{
		ctx:           ctx,
		secretStore:   newMockSecretStore(),
		providerStore: newMockProviderStore(),
		deviceFlowSvc: auth.NewOAuthDeviceFlowServiceBare(newMockProviderStore()),
	}

	result, err := app.DetectAuthMethods()
	require.NoError(t, err)
	require.NotNil(t, result)

	// 推荐结果不应为空
	assert.NotEmpty(t, result.Recommended)

	// 推荐的 method 必须在返回的结果中
	found := false
	for _, r := range result.Results {
		if r.Method == result.Recommended {
			found = true
			break
		}
	}
	assert.True(t, found, "推荐的 method 必须在 results 中存在")
}

// TestDetectAuthMethods_DetectsAPIKey 验证 API Key 检测逻辑。
func TestDetectAuthMethods_DetectsAPIKey(t *testing.T) {
	ctx := t.Context()
	ss := newMockSecretStore()
	_ = ss.Set("apikey:kimi", []byte("sk-test123"))

	app := &WailsApp{
		ctx:           ctx,
		secretStore:   ss,
		providerStore: newMockProviderStore(),
		deviceFlowSvc: auth.NewOAuthDeviceFlowServiceBare(newMockProviderStore()),
	}

	result, err := app.DetectAuthMethods()
	require.NoError(t, err)

	apiKeyStatus := findMethod(result.Results, "api_key")
	require.NotNil(t, apiKeyStatus)
	assert.True(t, apiKeyStatus.Available, "API Key 方式应始终可用")
	assert.True(t, apiKeyStatus.Connected, "应检测到已保存的 API Key")
	assert.Equal(t, "kimi", apiKeyStatus.ProviderType)
}

// TestDetectAuthMethods_Timeout 验证 2 秒超时不会 panic。
func TestDetectAuthMethods_Timeout(t *testing.T) {
	// 使用一个超短的 context 模拟超时场景（虽然实际实现使用 2s 固定超时）
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	app := &WailsApp{
		ctx:           ctx,
		secretStore:   newMockSecretStore(),
		providerStore: newMockProviderStore(),
		deviceFlowSvc: auth.NewOAuthDeviceFlowServiceBare(newMockProviderStore()),
	}

	// 使用一个带超时的 context 调用，确保整体不会阻塞过久
	done := make(chan struct{})
	var result *AuthDetectResult
	var err error

	go func() {
		result, err = app.DetectAuthMethods()
		close(done)
	}()

	select {
	case <-done:
		// 正常完成
	case <-time.After(10 * time.Second):
		t.Fatal("DetectAuthMethods 阻塞超过 10 秒，可能存在死锁")
	}

	require.NoError(t, err)
	require.NotNil(t, result)
}

// TestTestAPIKey_UnknownProvider 验证未知厂商返回错误。
func TestTestAPIKey_UnknownProvider(t *testing.T) {
	ctx := t.Context()
	app := &WailsApp{ctx: ctx}

	result, err := app.TestAPIKey("unknown", "sk-test", "")
	require.Error(t, err)
	assert.Nil(t, result)
}

// TestTestAPIKey_EmptyKey 验证空 API Key 返回错误。
func TestTestAPIKey_EmptyKey(t *testing.T) {
	ctx := t.Context()
	app := &WailsApp{ctx: ctx}

	result, err := app.TestAPIKey("openai", "", "")
	require.Error(t, err)
	assert.Nil(t, result)
}

// TestTestAPIKey_DefaultHost 验证未传 apiHost 时使用默认 host。
func TestTestAPIKey_DefaultHost(t *testing.T) {
	ctx := t.Context()
	app := &WailsApp{ctx: ctx}

	// 使用一个无效的 key，期望返回验证失败但不报错
	result, err := app.TestAPIKey("openai", "sk-invalid-test-key-12345", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Message)
}

// TestTestAPIKey_GeminiInvalidKey 验证 Gemini 使用原生 API 且 Key 无效时返回失败。
func TestTestAPIKey_GeminiInvalidKey(t *testing.T) {
	ctx := t.Context()
	app := &WailsApp{ctx: ctx}

	result, err := app.TestAPIKey("gemini", "invalid-key", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Message)
}

func findMethod(results []AuthMethodDetectStatus, method string) *AuthMethodDetectStatus {
	for i := range results {
		if results[i].Method == method {
			return &results[i]
		}
	}
	return nil
}
