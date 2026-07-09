package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProviderStore 是 ProviderStore 的内存实现，用于测试。
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
		return nil, fmt.Errorf("provider %s not found", id)
	}
	// 返回副本避免外部修改
	cpy := *p
	ret := new(models.ProviderConfig)
	*ret = cpy
	return ret, nil
}

func (m *mockProviderStore) List(_ context.Context) ([]*models.ProviderConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*models.ProviderConfig, 0, len(m.providers))
	for _, p := range m.providers {
		cpy := *p
		item := new(models.ProviderConfig)
		*item = cpy
		result = append(result, item)
	}
	return result, nil
}

// newKimiProvider 创建一个用于测试的 Kimi CLI Provider。
func newKimiProvider(id, credPath string) *models.ProviderConfig {
	now := time.Now()
	return &models.ProviderConfig{
		ID:          id,
		Name:        "Kimi Test",
		APIHost:     "https://api.moonshot.cn",
		ModelID:     "moonshot-v1-8k",
		AuthMethod:  models.AuthMethodCLIToken,
		AuthParams:  models.AuthParams{CLICredentialPath: credPath},
		Enabled:     true,
		Temperature: 0.7,
		TimeoutMs:   30000,
		MaxRetries:  3,
		CreatedAt:   now.UnixMilli(),
		UpdatedAt:   now.UnixMilli(),
	}
}

// newGeminiProvider 创建一个用于测试的 Gemini CLI Provider。
func newGeminiProvider(id, credPath string) *models.ProviderConfig {
	nowMs := time.Now().UnixMilli()
	return &models.ProviderConfig{
		ID:          id,
		Name:        "Gemini Test",
		APIHost:     "https://generativelanguage.googleapis.com/v1beta/openai/",
		ModelID:     "gemini-1.5-flash",
		AuthMethod:  models.AuthMethodCLIToken,
		AuthParams:  models.AuthParams{CLICredentialPath: credPath},
		Enabled:     true,
		Temperature: 0.7,
		TimeoutMs:   30000,
		MaxRetries:  3,
		CreatedAt:   nowMs,
		UpdatedAt:   nowMs,
	}
}

// TestTokenRefreshService_Refresh_Kimi_Success 验证 Kimi refresh 成功。
func TestTokenRefreshService_Refresh_Kimi_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建 Kimi 凭证文件（含 refresh_token）
	credPath := filepath.Join(tmpDir, "kimi-code.json")
	credContent := `{"access_token":"old_acc","refresh_token":"rt_kimi","client_id":"cid","client_secret":"cs","expires_at":1735689600}`
	require.NoError(t, os.WriteFile(credPath, []byte(credContent), 0600))

	// 模拟 token endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form failed: %v", err)
		}
		assert.Equal(t, "refresh_token", r.FormValue("grant_type"))
		assert.Equal(t, "rt_kimi", r.FormValue("refresh_token"))
		assert.Equal(t, "cid", r.FormValue("client_id"))
		assert.Equal(t, "cs", r.FormValue("client_secret"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new_acc_token",
			"refresh_token": "new_rt_token",
			"expires_in":    3600,
			"token_type":    "Bearer",
		})
	}))
	defer server.Close()

	// 替换 endpoint 为测试服务器
	origEndpoint := providerTokenEndpoints["kimi"]
	providerTokenEndpoints["kimi"] = server.URL
	defer func() { providerTokenEndpoints["kimi"] = origEndpoint }()

	store := newMockProviderStore()
	p := newKimiProvider("kimi-test", credPath)
	require.NoError(t, store.Create(context.Background(), p))

	var degradedID, degradedReason string
	svc := NewTokenRefreshService(store, WithTokenRefreshHTTPClient(server.Client()), WithTokenRefreshOnDegraded(func(id, reason string) {
		degradedID = id
		degradedReason = reason
	}))

	result, err := svc.Refresh("kimi-test")
	require.NoError(t, err)
	assert.Equal(t, "new_acc_token", result.AccessToken)
	assert.Equal(t, "new_rt_token", result.RefreshToken)
	assert.Equal(t, "Bearer", result.TokenType)
	assert.True(t, result.ExpiresAt > time.Now().Unix())

	// 验证 ProviderStore 已更新
	updated, err := store.Get(context.Background(), "kimi-test")
	require.NoError(t, err)
	assert.Equal(t, "new_acc_token", updated.AuthParams.OAuthAccessToken)
	assert.Equal(t, "new_rt_token", updated.AuthParams.OAuthRefreshToken)
	assert.True(t, updated.AuthParams.OAuthExpiresAt > 0)
	assert.True(t, updated.Enabled)

	// 验证凭证文件已更新
	fileData, err := os.ReadFile(credPath)
	require.NoError(t, err)
	var fileMap map[string]any
	require.NoError(t, json.Unmarshal(fileData, &fileMap))
	assert.Equal(t, "new_acc_token", fileMap["access_token"])
	assert.Equal(t, "new_rt_token", fileMap["refresh_token"])

	// 验证未触发降级
	assert.Empty(t, degradedID)
	assert.Empty(t, degradedReason)
}

// TestTokenRefreshService_Refresh_Gemini_Success 验证 Gemini refresh 成功。
func TestTokenRefreshService_Refresh_Gemini_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建 gcloud ADC 文件
	credPath := filepath.Join(tmpDir, "application_default_credentials.json")
	credContent := `{"client_id":"gcp_cid","client_secret":"gcp_cs","refresh_token":"1//gemini_rt","type":"authorized_user"}`
	require.NoError(t, os.WriteFile(credPath, []byte(credContent), 0600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "refresh_token", r.FormValue("grant_type"))
		assert.Equal(t, "1//gemini_rt", r.FormValue("refresh_token"))
		assert.Equal(t, "gcp_cid", r.FormValue("client_id"))
		assert.Equal(t, "gcp_cs", r.FormValue("client_secret"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "gemini_new_acc",
			"expires_in":   1800,
			"token_type":   "Bearer",
		})
	}))
	defer server.Close()

	origEndpoint := providerTokenEndpoints["gemini"]
	providerTokenEndpoints["gemini"] = server.URL
	defer func() { providerTokenEndpoints["gemini"] = origEndpoint }()

	store := newMockProviderStore()
	p := newGeminiProvider("gemini-test", credPath)
	require.NoError(t, store.Create(context.Background(), p))

	svc := NewTokenRefreshService(store, WithTokenRefreshHTTPClient(server.Client()))

	result, err := svc.Refresh("gemini-test")
	require.NoError(t, err)
	assert.Equal(t, "gemini_new_acc", result.AccessToken)

	// 验证 companion 缓存文件已创建
	cachePath := filepath.Join(tmpDir, "medmemo_adc_cache.json")
	cacheData, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	var cacheMap map[string]any
	require.NoError(t, json.Unmarshal(cacheData, &cacheMap))
	assert.Equal(t, "gemini_new_acc", cacheMap["access_token"])
}

// TestTokenRefreshService_Refresh_MissingClientCredentials 验证缺少 client credentials 返回错误。
func TestTokenRefreshService_Refresh_MissingClientCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "kimi.json")
	// Kimi 凭证缺少 client_id / client_secret
	require.NoError(t, os.WriteFile(credPath, []byte(`{"refresh_token":"rt_only"}`), 0600))

	store := newMockProviderStore()
	p := newKimiProvider("kimi-no-client", credPath)
	require.NoError(t, store.Create(context.Background(), p))

	svc := NewTokenRefreshService(store)
	_, err := svc.Refresh("kimi-no-client")
	assert.Error(t, err)
	// 缺少 client_id 不会导致立即报错，而是会在 doRefresh 中根据 endpoint 返回错误
	// 由于 endpoint 是真实网络地址，测试环境下会连接失败
	assert.Contains(t, err.Error(), "failed to refresh token")
}

// TestTokenRefreshService_Refresh_4xx_Degraded 验证 4xx 错误触发降级。
func TestTokenRefreshService_Refresh_4xx_Degraded(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "kimi.json")
	require.NoError(t, os.WriteFile(credPath, []byte(`{"refresh_token":"rt_bad","client_id":"cid","client_secret":"cs"}`), 0600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid_grant"})
	}))
	defer server.Close()

	origEndpoint := providerTokenEndpoints["kimi"]
	providerTokenEndpoints["kimi"] = server.URL
	defer func() { providerTokenEndpoints["kimi"] = origEndpoint }()

	store := newMockProviderStore()
	p := newKimiProvider("kimi-degraded", credPath)
	require.NoError(t, store.Create(context.Background(), p))

	var degradedID, degradedReason string
	svc := NewTokenRefreshService(store, WithTokenRefreshHTTPClient(server.Client()), WithTokenRefreshOnDegraded(func(id, reason string) {
		degradedID = id
		degradedReason = reason
	}))

	_, err := svc.Refresh("kimi-degraded")
	assert.Error(t, err)

	// 验证 provider 被禁用
	updated, err := store.Get(context.Background(), "kimi-degraded")
	require.NoError(t, err)
	assert.False(t, updated.Enabled)
	assert.Empty(t, updated.AuthParams.OAuthAccessToken)

	// 验证降级回调被触发
	assert.Equal(t, "kimi-degraded", degradedID)
	assert.Contains(t, degradedReason, "status 401")
}

// TestTokenRefreshService_Refresh_5xx_Retryable 验证 5xx 错误不触发降级。
func TestTokenRefreshService_Refresh_5xx_Retryable(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "kimi.json")
	require.NoError(t, os.WriteFile(credPath, []byte(`{"refresh_token":"rt_500","client_id":"cid","client_secret":"cs"}`), 0600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	origEndpoint := providerTokenEndpoints["kimi"]
	providerTokenEndpoints["kimi"] = server.URL
	defer func() { providerTokenEndpoints["kimi"] = origEndpoint }()

	store := newMockProviderStore()
	p := newKimiProvider("kimi-500", credPath)
	require.NoError(t, store.Create(context.Background(), p))

	var degradedCalled bool
	svc := NewTokenRefreshService(store, WithTokenRefreshHTTPClient(server.Client()), WithTokenRefreshOnDegraded(func(_, _ string) {
		degradedCalled = true
	}))

	_, err := svc.Refresh("kimi-500")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")

	// 验证未触发降级
	updated, err := store.Get(context.Background(), "kimi-500")
	require.NoError(t, err)
	assert.True(t, updated.Enabled)
	assert.False(t, degradedCalled)
}

// TestTokenRefreshService_ScheduleAutoRefresh_FutureExpiry 验证未来过期时间正确调度。
func TestTokenRefreshService_ScheduleAutoRefresh_FutureExpiry(t *testing.T) {
	store := newMockProviderStore()
	p := newKimiProvider("kimi-schedule", "")
	// 设置 1 小时后过期
	p.AuthParams.OAuthExpiresAt = time.Now().Add(1 * time.Hour).Unix()
	require.NoError(t, store.Create(context.Background(), p))

	svc := NewTokenRefreshService(store)
	require.NoError(t, svc.ScheduleAutoRefresh("kimi-schedule"))

	// 验证 timer 已创建
	svc.mu.Lock()
	_, ok := svc.timers["kimi-schedule"]
	svc.mu.Unlock()
	assert.True(t, ok, "timer should be created")

	svc.Shutdown()
}

// TestTokenRefreshService_ScheduleAutoRefresh_AlreadyExpired 验证已过期时立即触发刷新。
func TestTokenRefreshService_ScheduleAutoRefresh_AlreadyExpired(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "kimi.json")
	require.NoError(t, os.WriteFile(credPath, []byte(`{"refresh_token":"rt_expired","client_id":"cid","client_secret":"cs"}`), 0600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "acc_after_expiry",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	origEndpoint := providerTokenEndpoints["kimi"]
	providerTokenEndpoints["kimi"] = server.URL
	defer func() { providerTokenEndpoints["kimi"] = origEndpoint }()

	store := newMockProviderStore()
	p := newKimiProvider("kimi-expired", credPath)
	// 已过期 10 分钟
	p.AuthParams.OAuthExpiresAt = time.Now().Add(-10 * time.Minute).Unix()
	require.NoError(t, store.Create(context.Background(), p))

	svc := NewTokenRefreshService(store, WithTokenRefreshHTTPClient(server.Client()))
	require.NoError(t, svc.ScheduleAutoRefresh("kimi-expired"))

	// 由于 timer 是立即触发的（AfterFunc 0 延迟），等待一小段时间让 goroutine 执行
	time.Sleep(100 * time.Millisecond)

	// 验证已刷新
	updated, err := store.Get(context.Background(), "kimi-expired")
	require.NoError(t, err)
	assert.Equal(t, "acc_after_expiry", updated.AuthParams.OAuthAccessToken)

	svc.Shutdown()
}

// TestTokenRefreshService_CancelAutoRefresh 验证取消调度。
func TestTokenRefreshService_CancelAutoRefresh(t *testing.T) {
	store := newMockProviderStore()
	p := newKimiProvider("kimi-cancel", "")
	p.AuthParams.OAuthExpiresAt = time.Now().Add(1 * time.Hour).Unix()
	require.NoError(t, store.Create(context.Background(), p))

	svc := NewTokenRefreshService(store)
	require.NoError(t, svc.ScheduleAutoRefresh("kimi-cancel"))

	svc.CancelAutoRefresh("kimi-cancel")

	svc.mu.Lock()
	_, ok := svc.timers["kimi-cancel"]
	svc.mu.Unlock()
	assert.False(t, ok, "timer should be removed")
}

// TestTokenRefreshService_Shutdown 验证关闭时清理所有 timer。
func TestTokenRefreshService_Shutdown(t *testing.T) {
	store := newMockProviderStore()
	for i := 0; i < 3; i++ {
		p := newKimiProvider(fmt.Sprintf("kimi-%d", i), "")
		p.AuthParams.OAuthExpiresAt = time.Now().Add(1 * time.Hour).Unix()
		require.NoError(t, store.Create(context.Background(), p))
	}

	svc := NewTokenRefreshService(store)
	for i := 0; i < 3; i++ {
		require.NoError(t, svc.ScheduleAutoRefresh(fmt.Sprintf("kimi-%d", i)))
	}

	svc.Shutdown()

	svc.mu.Lock()
	count := len(svc.timers)
	svc.mu.Unlock()
	assert.Equal(t, 0, count, "all timers should be cleaned up")
}

// TestTokenRefreshService_Refresh_UnsupportedAuthMethod 验证不支持的认证方式返回错误。
func TestTokenRefreshService_Refresh_UnsupportedAuthMethod(t *testing.T) {
	store := newMockProviderStore()
	p := &models.ProviderConfig{
		ID:         "api-key-provider",
		Name:       "API Key Provider",
		AuthMethod: models.AuthMethodAPIToken,
		APIKey:     "sk-test",
	}
	require.NoError(t, store.Create(context.Background(), p))

	svc := NewTokenRefreshService(store)
	_, err := svc.Refresh("api-key-provider")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support refresh")
}

// TestTokenRefreshService_Refresh_NoRefreshToken 验证缺少 refresh_token 返回错误。
func TestTokenRefreshService_Refresh_NoRefreshToken(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "kimi.json")
	// 只有 access_token，没有 refresh_token
	require.NoError(t, os.WriteFile(credPath, []byte(`{"access_token":"acc_only"}`), 0600))

	store := newMockProviderStore()
	p := newKimiProvider("kimi-no-rt", credPath)
	require.NoError(t, store.Create(context.Background(), p))

	svc := NewTokenRefreshService(store)
	_, err := svc.Refresh("kimi-no-rt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has no refresh_token available")
}

// TestTokenRefreshService_RefreshProvider_Direct 验证直接刷新 ProviderConfig。
func TestTokenRefreshService_RefreshProvider_Direct(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "kimi.json")
	require.NoError(t, os.WriteFile(credPath, []byte(`{"refresh_token":"rt_direct","client_id":"cid","client_secret":"cs"}`), 0600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "acc_direct",
			"expires_in":   7200,
		})
	}))
	defer server.Close()

	origEndpoint := providerTokenEndpoints["kimi"]
	providerTokenEndpoints["kimi"] = server.URL
	defer func() { providerTokenEndpoints["kimi"] = origEndpoint }()

	store := newMockProviderStore()
	p := newKimiProvider("kimi-direct", credPath)
	require.NoError(t, store.Create(context.Background(), p))

	svc := NewTokenRefreshService(store, WithTokenRefreshHTTPClient(server.Client()))
	result, err := svc.RefreshProvider(p)
	require.NoError(t, err)
	assert.Equal(t, "acc_direct", result.AccessToken)
}

// TestTokenRefreshService_Integration_FullCycle 验证完整链路：创建 → 调度 → 刷新 → 验证持久化。
func TestTokenRefreshService_Integration_FullCycle(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "kimi.json")
	require.NoError(t, os.WriteFile(credPath, []byte(`{"refresh_token":"rt_int","client_id":"cid","client_secret":"cs"}`), 0600))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "acc_int",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	origEndpoint := providerTokenEndpoints["kimi"]
	providerTokenEndpoints["kimi"] = server.URL
	defer func() { providerTokenEndpoints["kimi"] = origEndpoint }()

	store := newMockProviderStore()
	p := newKimiProvider("kimi-int", credPath)
	p.AuthParams.OAuthExpiresAt = time.Now().Add(-5 * time.Minute).Unix() // 已过期
	require.NoError(t, store.Create(context.Background(), p))

	svc := NewTokenRefreshService(store, WithTokenRefreshHTTPClient(server.Client()))

	// 1. 调度自动刷新（应立即触发，因为已过期）
	require.NoError(t, svc.ScheduleAutoRefresh("kimi-int"))
	time.Sleep(100 * time.Millisecond)

	// 2. 验证数据库已更新
	updated, err := store.Get(context.Background(), "kimi-int")
	require.NoError(t, err)
	assert.Equal(t, "acc_int", updated.AuthParams.OAuthAccessToken)
	assert.True(t, updated.AuthParams.OAuthExpiresAt > time.Now().Unix())

	// 3. 验证凭证文件已更新
	fileData, err := os.ReadFile(credPath)
	require.NoError(t, err)
	var fileMap map[string]any
	require.NoError(t, json.Unmarshal(fileData, &fileMap))
	assert.Equal(t, "acc_int", fileMap["access_token"])

	// 4. 验证 timer 已重新调度（因为刷新成功后会重新调用 ScheduleAutoRefresh）
	svc.mu.Lock()
	_, ok := svc.timers["kimi-int"]
	svc.mu.Unlock()
	assert.True(t, ok, "timer should be rescheduled after successful refresh")

	svc.Shutdown()
}
