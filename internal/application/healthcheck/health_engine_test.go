package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProviderStore 是用于测试的内存 ProviderStore 实现。
type mockProviderStore struct {
	mu        sync.RWMutex
	providers map[string]*models.ProviderConfig
}

func newMockProviderStore() *mockProviderStore {
	return &mockProviderStore{providers: make(map[string]*models.ProviderConfig)}
}

func (m *mockProviderStore) Create(_ context.Context, p *models.ProviderConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[p.ID] = p
	return nil
}

func (m *mockProviderStore) Update(_ context.Context, p *models.ProviderConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[p.ID] = p
	return nil
}

func (m *mockProviderStore) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.providers, id)
	return nil
}

func (m *mockProviderStore) Get(_ context.Context, id string) (*models.ProviderConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.providers[id]
	if !ok {
		return nil, entity.ErrNotFound
	}
	return p, nil
}

func (m *mockProviderStore) List(_ context.Context) ([]*models.ProviderConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.ProviderConfig, 0, len(m.providers))
	for _, p := range m.providers {
		result = append(result, p)
	}
	return result, nil
}

func TestHealthEngine_CheckNow_Green(t *testing.T) {
		t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/models", r.URL.Path)
		auth := r.Header.Get("Authorization")
		assert.True(t, strings.HasPrefix(auth, "Bearer "))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := newMockProviderStore()
	_ = store.Create(context.Background(), &models.ProviderConfig{
		ID:      "p1",
		APIHost: server.URL,
		APIKey:  "test-key",
		Enabled: true,
	})

	engine := NewHealthEngine(store)
	result, err := engine.CheckNow(context.Background(), "p1")
	require.NoError(t, err)

	assert.Equal(t, "p1", result.ProviderID)
	assert.Equal(t, port.HealthGreen, result.Status)
	assert.True(t, result.LatencyMs >= 0)
	assert.Empty(t, result.Error)
}

func TestHealthEngine_CheckNow_Yellow(t *testing.T) {
		t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟 2.5s 延迟，落在 Yellow 区间
		time.Sleep(2500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := newMockProviderStore()
	_ = store.Create(context.Background(), &models.ProviderConfig{
		ID:      "p1",
		APIHost: server.URL,
		APIKey:  "test-key",
		Enabled: true,
	})

	// 使用足够长的检测超时，让 2.5s 延迟落在 Yellow 区间
	engine := &HealthEngine{
		store:        store,
		client:       &http.Client{Timeout: 10 * time.Second},
		checkTimeout: 10 * time.Second,
		interval:     defaultCheckInterval,
	}
	result, err := engine.CheckNow(context.Background(), "p1")
	require.NoError(t, err)

	assert.Equal(t, port.HealthYellow, result.Status)
	assert.True(t, result.LatencyMs >= 2000)
}

func TestHealthEngine_CheckNow_Red_NonOK(t *testing.T) {
		t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	store := newMockProviderStore()
	_ = store.Create(context.Background(), &models.ProviderConfig{
		ID:      "p1",
		APIHost: server.URL,
		APIKey:  "test-key",
		Enabled: true,
	})

	engine := NewHealthEngine(store)
	result, err := engine.CheckNow(context.Background(), "p1")
	require.NoError(t, err)

	assert.Equal(t, port.HealthRed, result.Status)
	assert.Contains(t, result.Error, "403")
}

func TestHealthEngine_CheckNow_Red_500(t *testing.T) {
		t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	store := newMockProviderStore()
	_ = store.Create(context.Background(), &models.ProviderConfig{
		ID:      "p1",
		APIHost: server.URL,
		APIKey:  "test-key",
		Enabled: true,
	})

	engine := NewHealthEngine(store)
	result, err := engine.CheckNow(context.Background(), "p1")
	require.NoError(t, err)

	assert.Equal(t, port.HealthRed, result.Status)
	assert.Contains(t, result.Error, "500")
}

func TestHealthEngine_CheckNow_Red_Timeout(t *testing.T) {
		t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 响应延迟超过 client 超时（1ms）
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := newMockProviderStore()
	_ = store.Create(context.Background(), &models.ProviderConfig{
		ID:      "p1",
		APIHost: server.URL,
		APIKey:  "test-key",
		Enabled: true,
	})

	// 使用极短超时触发超时错误
	engine := NewHealthEngineWithClient(store, &http.Client{Timeout: 1 * time.Millisecond})
	result, err := engine.CheckNow(context.Background(), "p1")
	require.NoError(t, err)

	assert.Equal(t, port.HealthRed, result.Status)
	assert.Contains(t, result.Error, "请求失败")
}

func TestHealthEngine_CheckNow_NetworkError(t *testing.T) {
		t.Parallel()
	store := newMockProviderStore()
	_ = store.Create(context.Background(), &models.ProviderConfig{
		ID:      "p1",
		APIHost: "http://127.0.0.1:1", // 不可达端口
		APIKey:  "test-key",
		Enabled: true,
	})

	engine := NewHealthEngine(store)
	result, err := engine.CheckNow(context.Background(), "p1")
	require.NoError(t, err)

	assert.Equal(t, port.HealthRed, result.Status)
	assert.NotEmpty(t, result.Error)
}

func TestHealthEngine_OnChange(t *testing.T) {
		t.Parallel()
	callCount := atomic.Int32{}
	var lastResult port.HealthResult

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := newMockProviderStore()
	_ = store.Create(context.Background(), &models.ProviderConfig{
		ID:      "p1",
		APIHost: server.URL,
		APIKey:  "test-key",
		Enabled: true,
	})

	engine := NewHealthEngine(store)
	engine.SetOnChange(func(r port.HealthResult) {
		callCount.Add(1)
		lastResult = r
	})

	// 首次检测触发回调（Unknown -> Green）
	_, err := engine.CheckNow(context.Background(), "p1")
	require.NoError(t, err)

	assert.Equal(t, int32(1), callCount.Load())
	assert.Equal(t, port.HealthGreen, lastResult.Status)

	// 再次检测，状态未变，不应触发
	_, err = engine.CheckNow(context.Background(), "p1")
	require.NoError(t, err)

	assert.Equal(t, int32(1), callCount.Load())
}

func TestHealthEngine_OnChange_StatusTransition(t *testing.T) {
		t.Parallel()
	var statuses []port.HealthStatus
	var mu sync.Mutex

	// 第一次返回 200，第二次返回 403
	responseCode := atomic.Int32{}
	responseCode.Store(200)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := int(responseCode.Load())
		w.WriteHeader(code)
	}))
	defer server.Close()

	store := newMockProviderStore()
	_ = store.Create(context.Background(), &models.ProviderConfig{
		ID:      "p1",
		APIHost: server.URL,
		APIKey:  "test-key",
		Enabled: true,
	})

	engine := NewHealthEngine(store)
	engine.SetOnChange(func(r port.HealthResult) {
		mu.Lock()
		statuses = append(statuses, r.Status)
		mu.Unlock()
	})

	// 首次 Green
	_, err := engine.CheckNow(context.Background(), "p1")
	require.NoError(t, err)

	// 切换为 403
	responseCode.Store(403)

	// 再次检测，Red，触发回调
	_, err = engine.CheckNow(context.Background(), "p1")
	require.NoError(t, err)

	mu.Lock()
	assert.Equal(t, []port.HealthStatus{port.HealthGreen, port.HealthRed}, statuses)
	mu.Unlock()
}

func TestHealthEngine_GetStatus(t *testing.T) {
		t.Parallel()
	store := newMockProviderStore()
	engine := NewHealthEngine(store)

	// 未检测过
	_, ok := engine.GetStatus("p1")
	assert.False(t, ok)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_ = store.Create(context.Background(), &models.ProviderConfig{
		ID:      "p1",
		APIHost: server.URL,
		APIKey:  "test-key",
		Enabled: true,
	})

	_, err := engine.CheckNow(context.Background(), "p1")
	require.NoError(t, err)

	result, ok := engine.GetStatus("p1")
	assert.True(t, ok)
	assert.Equal(t, port.HealthGreen, result.Status)
}

func TestHealthEngine_PeriodicCheck(t *testing.T) {
		t.Parallel()
	callCount := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := newMockProviderStore()
	_ = store.Create(context.Background(), &models.ProviderConfig{
		ID:      "p1",
		APIHost: server.URL,
		APIKey:  "test-key",
		Enabled: true,
	})

	engine := NewHealthEngineWithClient(store, server.Client())
	engine.interval = 100 * time.Millisecond // 加速轮询
	engine.SetOnChange(func(r port.HealthResult) {
		callCount.Add(1)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
	defer cancel()

	engine.Start(ctx)
	<-ctx.Done()
	engine.Stop()

	// 首次检测 + 约 3 次 ticker 触发，但状态始终为 Green，
	// 只有首次 Unknown->Green 会触发回调
	assert.Equal(t, int32(1), callCount.Load())
}

func TestHealthEngine_CheckNow_ProviderNotFound(t *testing.T) {
		t.Parallel()
	store := newMockProviderStore()
	engine := NewHealthEngine(store)

	_, err := engine.CheckNow(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestHealthEngine_CheckNow_Disabled(t *testing.T) {
		t.Parallel()
	store := newMockProviderStore()
	_ = store.Create(context.Background(), &models.ProviderConfig{
		ID:      "p1",
		APIHost: "http://example.com",
		APIKey:  "test-key",
		Enabled: false,
	})

	engine := NewHealthEngine(store)
	_, err := engine.CheckNow(context.Background(), "p1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestHealthEngine_StartStop_Idempotent(t *testing.T) {
		t.Parallel()
	store := newMockProviderStore()
	engine := NewHealthEngine(store)

	// 多次停止不应 panic
	engine.Stop()
	engine.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine.Start(ctx)
	engine.Start(ctx) // 重复启动应被忽略
	engine.Stop()
	engine.Stop() // 重复停止应被忽略
}

func TestClassifyLatency(t *testing.T) {
		t.Parallel()
	tests := []struct {
		latency  time.Duration
		expected port.HealthStatus
	}{
		{0, port.HealthGreen},
		{1 * time.Second, port.HealthGreen},
		{1999 * time.Millisecond, port.HealthGreen},
		{2000 * time.Millisecond, port.HealthYellow},
		{3500 * time.Millisecond, port.HealthYellow},
		{5000 * time.Millisecond, port.HealthYellow},
		{5001 * time.Millisecond, port.HealthRed},
		{10 * time.Second, port.HealthRed},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.latency), func(t *testing.T) {
			assert.Equal(t, tt.expected, classifyLatency(tt.latency))
		})
	}
}

func TestHealthEngine_CheckNow_NoAPIKey(t *testing.T) {
		t.Parallel()
	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := newMockProviderStore()
	_ = store.Create(context.Background(), &models.ProviderConfig{
		ID:      "p1",
		APIHost: server.URL,
		APIKey:  "", // 空 Key
		Enabled: true,
	})

	engine := NewHealthEngine(store)
	result, err := engine.CheckNow(context.Background(), "p1")
	require.NoError(t, err)

	assert.Equal(t, port.HealthGreen, result.Status)
	assert.Empty(t, authHeader) // 不应发送 Authorization header
}

// TestHealthEngine_CheckNow_CLIToken_Success 验证 cli_token 方式正确读取凭证并通过健康检测。
func TestHealthEngine_CheckNow_CLIToken_Success(t *testing.T) {
		t.Parallel()
	tmpDir := t.TempDir()
	credPath := tmpDir + "/kimi-code.json"
	require.NoError(t, os.WriteFile(credPath, []byte(`{"access_token":"cli-token-hc"}`), 0600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer cli-token-hc", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	store := newMockProviderStore()
	_ = store.Create(context.Background(), &models.ProviderConfig{
		ID:         "cli-p1",
		APIHost:    server.URL,
		APIKey:     "",
		ModelID:    "gpt-4o",
		Enabled:    true,
		AuthMethod: models.AuthMethodCLIToken,
		AuthParams: models.AuthParams{CLICredentialPath: credPath},
	})

	engine := NewHealthEngineWithClient(store, server.Client())
	result, err := engine.CheckNow(context.Background(), "cli-p1")
	require.NoError(t, err)

	assert.Equal(t, port.HealthGreen, result.Status)
	assert.Empty(t, result.Error)
}

// TestHealthEngine_CheckNow_OAuthDevice_CacheExpired 验证 oauth_device 方式缓存过期时返回 Red。
func TestHealthEngine_CheckNow_OAuthDevice_CacheExpired(t *testing.T) {
		t.Parallel()
	store := newMockProviderStore()
	_ = store.Create(context.Background(), &models.ProviderConfig{
		ID:         "oauth-p1",
		APIHost:    "https://api.example.com",
		APIKey:     "",
		ModelID:    "gpt-4o",
		Enabled:    true,
		AuthMethod: models.AuthMethodOAuthDevice,
		AuthParams: models.AuthParams{
			OAuthClientID: "client-123",
			OAuthTokenURL: "https://auth.example.com/token",
		},
	})

	engine := NewHealthEngine(store)
	result, err := engine.CheckNow(context.Background(), "oauth-p1")
	require.NoError(t, err)

	assert.Equal(t, port.HealthRed, result.Status)
	assert.Contains(t, result.Error, "access_token expired, refresh required")
}

// TestHealthEngine_CheckNow_ServiceAccount_NotImplemented 验证 service_account 方式返回 Red。
func TestHealthEngine_CheckNow_ServiceAccount_NotImplemented(t *testing.T) {
		t.Parallel()
	store := newMockProviderStore()
	_ = store.Create(context.Background(), &models.ProviderConfig{
		ID:         "sa-p1",
		APIHost:    "https://us-central1-aiplatform.googleapis.com",
		APIKey:     "",
		ModelID:    "gemini-pro",
		Enabled:    true,
		AuthMethod: models.AuthMethodServiceAccount,
		AuthParams: models.AuthParams{
			GCPProjectID: "my-project",
			SAJSON:       `{"type":"service_account"}`,
		},
	})

	engine := NewHealthEngine(store)
	result, err := engine.CheckNow(context.Background(), "sa-p1")
	require.NoError(t, err)

	assert.Equal(t, port.HealthRed, result.Status)
	assert.Contains(t, result.Error, "not yet implemented")
}
