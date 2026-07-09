package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain 为 Device Flow 测试设置全局环境变量。
func TestMain(m *testing.M) {
	os.Setenv("TEST_CLIENT_ID", "test-client")
	os.Exit(m.Run())
}

// TestOAuthDeviceFlowService_StartFlow_UnsupportedProvider 验证不支持的厂商类型。
func TestOAuthDeviceFlowService_StartFlow_UnsupportedProvider(t *testing.T) {
	store := newMockProviderStore()
	svc := NewOAuthDeviceFlowServiceWithClient(store, nil, &http.Client{Timeout: 5 * time.Second})

	_, err := svc.StartFlow("unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported provider type")
}

// TestOAuthDeviceFlowService_StartFlow_MissingClientID 验证缺少 client_id 时报错。
func TestOAuthDeviceFlowService_StartFlow_MissingClientID(t *testing.T) {
	store := newMockProviderStore()
	svc := NewOAuthDeviceFlowServiceWithClient(store, nil, &http.Client{Timeout: 5 * time.Second})

	// kimi 预置配置中 client_id 为空
	_, err := svc.StartFlow("kimi")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "需要配置 OAuth client_id")
}

// TestOAuthDeviceFlowService_FullFlow_Success 验证完整 Device Flow 成功链路。
func TestOAuthDeviceFlowService_FullFlow_Success(t *testing.T) {
	var deviceAuthCalled atomic.Bool
	var tokenPollCount atomic.Int32

	// 模拟厂商服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/device/auth":
			deviceAuthCalled.Store(true)
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

			r.ParseForm()
			assert.Equal(t, "test-client", r.PostForm.Get("client_id"))
			assert.Equal(t, "test-scope", r.PostForm.Get("scope"))

			resp := DeviceAuthResponse{
				DeviceCode:      "dev_abc123",
				UserCode:        "USER-CODE",
				VerificationURI: "https://auth.example.com/verify",
				ExpiresIn:       600,
				Interval:        1, // 测试用短间隔
			}
			_ = json.NewEncoder(w).Encode(resp)

		case "/token":
			cnt := tokenPollCount.Add(1)
			r.ParseForm()
			assert.Equal(t, "test-client", r.PostForm.Get("client_id"))
			assert.Equal(t, "dev_abc123", r.PostForm.Get("device_code"))
			assert.Equal(t, "urn:ietf:params:oauth:grant-type:device_code", r.PostForm.Get("grant_type"))

			// 第 1 次返回 authorization_pending，第 2 次成功
			if cnt < 2 {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
				return
			}

			resp := DeviceTokenResponse{
				AccessToken:  "acc_token_xyz",
				RefreshToken: "refresh_token_abc",
				ExpiresIn:    3600,
				TokenType:    "Bearer",
			}
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	// 覆盖预置配置为测试服务器地址
	origConfig := deviceFlowConfigs["testprovider"]
	deviceFlowConfigs["testprovider"] = struct {
		DeviceAuthURL string
		TokenURL      string
		Scope         string
		EnvClientID   string
	}{
		DeviceAuthURL: server.URL + "/device/auth",
		TokenURL:      server.URL + "/token",
		Scope:         "test-scope",
		EnvClientID:   "TEST_CLIENT_ID",
	}
	defer func() {
		if origConfig.DeviceAuthURL == "" {
			delete(deviceFlowConfigs, "testprovider")
		} else {
			deviceFlowConfigs["testprovider"] = origConfig
		}
	}()

	store := newMockProviderStore()
	refreshSvc := NewTokenRefreshService(store, WithTokenRefreshHTTPClient(&http.Client{Timeout: 5 * time.Second}))
	svc := NewOAuthDeviceFlowServiceWithClient(store, refreshSvc, server.Client())

	var successEvent struct {
		deviceCode   string
		providerType string
		cfg          *models.ProviderConfig
	}
	var mu sync.Mutex
	svc.SetCallbacks(
		func(dc, pt string, cfg *models.ProviderConfig) {
			mu.Lock()
			defer mu.Unlock()
			successEvent = struct {
				deviceCode   string
				providerType string
				cfg          *models.ProviderConfig
			}{dc, pt, cfg}
		},
		func(dc, pt string, err error) { t.Logf("error: %v", err) },
		func(dc, pt string) { t.Logf("pending: %s", dc) },
		func(dc, pt string, interval int) { t.Logf("slow_down: %d", interval) },
	)

	// 1. 启动 Flow
	result, err := svc.StartFlow("testprovider")
	require.NoError(t, err)
	assert.True(t, deviceAuthCalled.Load())
	assert.Equal(t, "USER-CODE", result.UserCode)
	assert.Equal(t, "https://auth.example.com/verify", result.VerificationURI)
	assert.Equal(t, "dev_abc123", result.DeviceCode)

	// 2. 等待轮询完成（最多 5 秒）
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return successEvent.cfg != nil
	}, 5*time.Second, 100*time.Millisecond, "等待 oauth:success 事件超时")

	// 3. 验证事件
	assert.Equal(t, "dev_abc123", successEvent.deviceCode)
	assert.Equal(t, "testprovider", successEvent.providerType)
	assert.NotNil(t, successEvent.cfg)
	assert.Equal(t, models.AuthMethodOAuthDevice, successEvent.cfg.AuthMethod)
	assert.Equal(t, "acc_token_xyz", successEvent.cfg.AuthParams.OAuthAccessToken)
	assert.Equal(t, "refresh_token_abc", successEvent.cfg.AuthParams.OAuthRefreshToken)

	// 4. 验证已持久化到 store
	ctx := t.Context()
	list, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "acc_token_xyz", list[0].AuthParams.OAuthAccessToken)

	// 5. 验证轮询了多次
	assert.GreaterOrEqual(t, tokenPollCount.Load(), int32(2))

	// 6. 验证状态查询
	status := svc.GetStatus("dev_abc123")
	assert.NotNil(t, status)
	assert.Equal(t, DeviceFlowStatusSuccess, status.Status)
}

// TestOAuthDeviceFlowService_SlowDown 验证 slow_down 处理。
func TestOAuthDeviceFlowService_SlowDown(t *testing.T) {
	var tokenPollCount atomic.Int32
	var slowDownInterval atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/device/auth" {
			_ = json.NewEncoder(w).Encode(DeviceAuthResponse{
				DeviceCode:      "dev_slow",
				UserCode:        "USER-SLOW",
				VerificationURI: "https://auth.example.com/verify",
				ExpiresIn:       60,
				Interval:        1,
			})
			return
		}
		if r.URL.Path == "/token" {
			cnt := tokenPollCount.Add(1)
			if cnt < 2 {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "slow_down"})
				return
			}
			_ = json.NewEncoder(w).Encode(DeviceTokenResponse{
				AccessToken: "acc_slow",
				ExpiresIn:   3600,
			})
		}
	}))
	defer server.Close()

	origConfig := deviceFlowConfigs["testprovider"]
	deviceFlowConfigs["testprovider"] = struct {
		DeviceAuthURL string
		TokenURL      string
		Scope         string
		EnvClientID   string
	}{
		DeviceAuthURL: server.URL + "/device/auth",
		TokenURL:      server.URL + "/token",
		Scope:         "test-scope",
		EnvClientID:   "TEST_CLIENT_ID",
	}
	defer func() {
		if origConfig.DeviceAuthURL == "" {
			delete(deviceFlowConfigs, "testprovider")
		} else {
			deviceFlowConfigs["testprovider"] = origConfig
		}
	}()

	store := newMockProviderStore()
	svc := NewOAuthDeviceFlowServiceWithClient(store, nil, server.Client())

	svc.SetCallbacks(
		func(_, _ string, _ *models.ProviderConfig) {},
		func(_, _ string, _ error) {},
		func(_, _ string) {},
		func(_, _ string, interval int) { slowDownInterval.Store(int32(interval)) },
	)

	_, err := svc.StartFlow("testprovider")
	require.NoError(t, err)

	// 等待成功（slow_down 后需要更长时间）
	require.Eventually(t, func() bool {
		st := svc.GetStatus("dev_slow")
		return st != nil && st.Status == DeviceFlowStatusSuccess
	}, 10*time.Second, 200*time.Millisecond)

	assert.GreaterOrEqual(t, tokenPollCount.Load(), int32(2))
	assert.GreaterOrEqual(t, slowDownInterval.Load(), int32(6)) // 初始 1 + slow_down 增加 5
}

// TestOAuthDeviceFlowService_AccessDenied 验证用户拒绝授权。
func TestOAuthDeviceFlowService_AccessDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/device/auth" {
			_ = json.NewEncoder(w).Encode(DeviceAuthResponse{
				DeviceCode:      "dev_deny",
				UserCode:        "USER-DENY",
				VerificationURI: "https://auth.example.com/verify",
				ExpiresIn:       60,
				Interval:        1,
			})
			return
		}
		if r.URL.Path == "/token" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "access_denied"})
		}
	}))
	defer server.Close()

	origConfig := deviceFlowConfigs["testprovider"]
	deviceFlowConfigs["testprovider"] = struct {
		DeviceAuthURL string
		TokenURL      string
		Scope         string
		EnvClientID   string
	}{
		DeviceAuthURL: server.URL + "/device/auth",
		TokenURL:      server.URL + "/token",
		Scope:         "test-scope",
		EnvClientID:   "TEST_CLIENT_ID",
	}
	defer func() {
		if origConfig.DeviceAuthURL == "" {
			delete(deviceFlowConfigs, "testprovider")
		} else {
			deviceFlowConfigs["testprovider"] = origConfig
		}
	}()

	store := newMockProviderStore()
	svc := NewOAuthDeviceFlowServiceWithClient(store, nil, server.Client())

	var errEvent atomic.Pointer[error]
	svc.SetCallbacks(
		func(_, _ string, _ *models.ProviderConfig) {},
		func(_, _ string, err error) { errEvent.Store(&err) },
		func(_, _ string) {},
		func(_, _ string, _ int) {},
	)

	_, err := svc.StartFlow("testprovider")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return errEvent.Load() != nil
	}, 5*time.Second, 100*time.Millisecond)

	eventErr := *errEvent.Load()
	assert.Contains(t, eventErr.Error(), "user denied authorization")

	status := svc.GetStatus("dev_deny")
	require.NotNil(t, status)
	assert.Equal(t, DeviceFlowStatusError, status.Status)
}

// TestOAuthDeviceFlowService_ExpiredToken 验证设备码过期。
func TestOAuthDeviceFlowService_ExpiredToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/device/auth" {
			_ = json.NewEncoder(w).Encode(DeviceAuthResponse{
				DeviceCode:      "dev_expired",
				UserCode:        "USER-EXP",
				VerificationURI: "https://auth.example.com/verify",
				ExpiresIn:       2, // 2 秒后过期
				Interval:        1,
			})
			return
		}
		if r.URL.Path == "/token" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "expired_token"})
		}
	}))
	defer server.Close()

	origConfig := deviceFlowConfigs["testprovider"]
	deviceFlowConfigs["testprovider"] = struct {
		DeviceAuthURL string
		TokenURL      string
		Scope         string
		EnvClientID   string
	}{
		DeviceAuthURL: server.URL + "/device/auth",
		TokenURL:      server.URL + "/token",
		Scope:         "test-scope",
		EnvClientID:   "TEST_CLIENT_ID",
	}
	defer func() {
		if origConfig.DeviceAuthURL == "" {
			delete(deviceFlowConfigs, "testprovider")
		} else {
			deviceFlowConfigs["testprovider"] = origConfig
		}
	}()

	store := newMockProviderStore()
	svc := NewOAuthDeviceFlowServiceWithClient(store, nil, server.Client())

	var errEvent atomic.Pointer[error]
	svc.SetCallbacks(
		func(_, _ string, _ *models.ProviderConfig) {},
		func(_, _ string, err error) { errEvent.Store(&err) },
		func(_, _ string) {},
		func(_, _ string, _ int) {},
	)

	_, err := svc.StartFlow("testprovider")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return errEvent.Load() != nil
	}, 10*time.Second, 200*time.Millisecond)

	eventErr := *errEvent.Load()
	assert.Contains(t, eventErr.Error(), "device code expired")
}

// TestOAuthDeviceFlowService_CancelFlow 验证取消轮询。
func TestOAuthDeviceFlowService_CancelFlow(t *testing.T) {
	var pollCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/device/auth" {
			_ = json.NewEncoder(w).Encode(DeviceAuthResponse{
				DeviceCode:      "dev_cancel",
				UserCode:        "USER-CANCEL",
				VerificationURI: "https://auth.example.com/verify",
				ExpiresIn:       600,
				Interval:        1,
			})
			return
		}
		if r.URL.Path == "/token" {
			pollCount.Add(1)
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
		}
	}))
	defer server.Close()

	origConfig := deviceFlowConfigs["testprovider"]
	deviceFlowConfigs["testprovider"] = struct {
		DeviceAuthURL string
		TokenURL      string
		Scope         string
		EnvClientID   string
	}{
		DeviceAuthURL: server.URL + "/device/auth",
		TokenURL:      server.URL + "/token",
		Scope:         "test-scope",
		EnvClientID:   "TEST_CLIENT_ID",
	}
	defer func() {
		if origConfig.DeviceAuthURL == "" {
			delete(deviceFlowConfigs, "testprovider")
		} else {
			deviceFlowConfigs["testprovider"] = origConfig
		}
	}()

	store := newMockProviderStore()
	svc := NewOAuthDeviceFlowServiceWithClient(store, nil, server.Client())

	_, err := svc.StartFlow("testprovider")
	require.NoError(t, err)

	// 等待至少一次轮询
	time.Sleep(1500 * time.Millisecond)
	require.GreaterOrEqual(t, pollCount.Load(), int32(1))

	// 取消
	svc.CancelFlow("dev_cancel")

	// 验证状态
	status := svc.GetStatus("dev_cancel")
	assert.Nil(t, status) // 取消后已从 sessions 中删除
}

// TestOAuthDeviceFlowService_Shutdown 验证关闭所有会话。
func TestOAuthDeviceFlowService_Shutdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/device/auth" {
			_ = json.NewEncoder(w).Encode(DeviceAuthResponse{
				DeviceCode:      "dev_shutdown",
				UserCode:        "USER-SD",
				VerificationURI: "https://auth.example.com/verify",
				ExpiresIn:       600,
				Interval:        1,
			})
			return
		}
		if r.URL.Path == "/token" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
		}
	}))
	defer server.Close()

	origConfig := deviceFlowConfigs["testprovider"]
	deviceFlowConfigs["testprovider"] = struct {
		DeviceAuthURL string
		TokenURL      string
		Scope         string
		EnvClientID   string
	}{
		DeviceAuthURL: server.URL + "/device/auth",
		TokenURL:      server.URL + "/token",
		Scope:         "test-scope",
		EnvClientID:   "TEST_CLIENT_ID",
	}
	defer func() {
		if origConfig.DeviceAuthURL == "" {
			delete(deviceFlowConfigs, "testprovider")
		} else {
			deviceFlowConfigs["testprovider"] = origConfig
		}
	}()

	store := newMockProviderStore()
	svc := NewOAuthDeviceFlowServiceWithClient(store, nil, server.Client())

	_, err := svc.StartFlow("testprovider")
	require.NoError(t, err)

	// 等待一次轮询
	time.Sleep(1500 * time.Millisecond)

	// Shutdown 应取消所有会话
	svc.Shutdown()

	// 验证 sessions 已清空
	svc.mu.Lock()
	count := len(svc.sessions)
	svc.mu.Unlock()
	assert.Equal(t, 0, count)
}

// TestOAuthDeviceFlowService_GetStatus_NotFound 验证查询不存在的 session。
func TestOAuthDeviceFlowService_GetStatus_NotFound(t *testing.T) {
	store := newMockProviderStore()
	svc := NewOAuthDeviceFlowServiceWithClient(store, nil, &http.Client{Timeout: 5 * time.Second})

	status := svc.GetStatus("nonexistent")
	assert.Nil(t, status)
}

// TestOAuthDeviceFlowService_DeviceAuthEndpointError 验证设备授权端点错误。
func TestOAuthDeviceFlowService_DeviceAuthEndpointError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/device/auth" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}))
	defer server.Close()

	origConfig := deviceFlowConfigs["testprovider"]
	deviceFlowConfigs["testprovider"] = struct {
		DeviceAuthURL string
		TokenURL      string
		Scope         string
		EnvClientID   string
	}{
		DeviceAuthURL: server.URL + "/device/auth",
		TokenURL:      server.URL + "/token",
		Scope:         "test-scope",
		EnvClientID:   "TEST_CLIENT_ID",
	}
	defer func() {
		if origConfig.DeviceAuthURL == "" {
			delete(deviceFlowConfigs, "testprovider")
		} else {
			deviceFlowConfigs["testprovider"] = origConfig
		}
	}()

	store := newMockProviderStore()
	svc := NewOAuthDeviceFlowServiceWithClient(store, nil, server.Client())

	_, err := svc.StartFlow("testprovider")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

// TestOAuthDeviceFlowService_DeviceAuthMissingDeviceCode 验证设备授权响应缺少 device_code。
func TestOAuthDeviceFlowService_DeviceAuthMissingDeviceCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/device/auth" {
			_ = json.NewEncoder(w).Encode(map[string]string{"user_code": "USER"})
			return
		}
	}))
	defer server.Close()

	origConfig := deviceFlowConfigs["testprovider"]
	deviceFlowConfigs["testprovider"] = struct {
		DeviceAuthURL string
		TokenURL      string
		Scope         string
		EnvClientID   string
	}{
		DeviceAuthURL: server.URL + "/device/auth",
		TokenURL:      server.URL + "/token",
		Scope:         "test-scope",
		EnvClientID:   "TEST_CLIENT_ID",
	}
	defer func() {
		if origConfig.DeviceAuthURL == "" {
			delete(deviceFlowConfigs, "testprovider")
		} else {
			deviceFlowConfigs["testprovider"] = origConfig
		}
	}()

	store := newMockProviderStore()
	svc := NewOAuthDeviceFlowServiceWithClient(store, nil, server.Client())

	_, err := svc.StartFlow("testprovider")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing device_code")
}

// TestOAuthDeviceFlowService_inferProviderInfo 验证厂商信息推断。
func TestOAuthDeviceFlowService_inferProviderInfo(t *testing.T) {
	store := newMockProviderStore()
	svc := NewOAuthDeviceFlowServiceWithClient(store, nil, &http.Client{Timeout: 5 * time.Second})

	tests := []struct {
		providerType string
		wantHost     string
		wantModel    string
		wantName     string
	}{
		{"kimi", "https://api.moonshot.cn", "moonshot-v1-8k", "Kimi (OAuth)"},
		{"gemini", "https://generativelanguage.googleapis.com/v1beta/openai", "gemini-1.5-flash", "Gemini (OAuth)"},
		{"unknown", "", "", "OAuth unknown"},
	}

	for _, tt := range tests {
		host, model, name := svc.inferProviderInfo(tt.providerType)
		assert.Equal(t, tt.wantHost, host, tt.providerType)
		assert.Equal(t, tt.wantModel, model, tt.providerType)
		assert.Equal(t, tt.wantName, name, tt.providerType)
	}
}

// TestOAuthDeviceFlowService_SetRefreshService 验证设置 refresh service。
func TestOAuthDeviceFlowService_SetRefreshService(t *testing.T) {
	store := newMockProviderStore()
	svc := NewOAuthDeviceFlowServiceBare(store)
	assert.Nil(t, svc.refreshSvc)

	refreshSvc := NewTokenRefreshService(store, WithTokenRefreshHTTPClient(&http.Client{Timeout: 5 * time.Second}))
	svc.SetRefreshService(refreshSvc)
	assert.NotNil(t, svc.refreshSvc)
}

// TestOAuthDeviceFlowService_TriggerPoll 验证 TriggerPoll 触发立即轮询。
func TestOAuthDeviceFlowService_TriggerPoll(t *testing.T) {
	var tokenPollCount atomic.Int32
	var successEvent atomic.Pointer[models.ProviderConfig]

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/device/auth" {
			_ = json.NewEncoder(w).Encode(DeviceAuthResponse{
				DeviceCode:      "dev_trigger",
				UserCode:        "USER-TRIGGER",
				VerificationURI: "https://auth.example.com/verify",
				ExpiresIn:       600,
				Interval:        300, // 设置很长间隔，确保 ticker 不会自然触发
			})
			return
		}
		if r.URL.Path == "/token" {
			tokenPollCount.Add(1)
			_ = json.NewEncoder(w).Encode(DeviceTokenResponse{
				AccessToken: "acc_trigger",
				ExpiresIn:   3600,
			})
		}
	}))
	defer server.Close()

	origConfig := deviceFlowConfigs["testprovider"]
	deviceFlowConfigs["testprovider"] = struct {
		DeviceAuthURL string
		TokenURL      string
		Scope         string
		EnvClientID   string
	}{
		DeviceAuthURL: server.URL + "/device/auth",
		TokenURL:      server.URL + "/token",
		Scope:         "test-scope",
		EnvClientID:   "TEST_CLIENT_ID",
	}
	defer func() {
		if origConfig.DeviceAuthURL == "" {
			delete(deviceFlowConfigs, "testprovider")
		} else {
			deviceFlowConfigs["testprovider"] = origConfig
		}
	}()

	store := newMockProviderStore()
	svc := NewOAuthDeviceFlowServiceWithClient(store, nil, server.Client())

	svc.SetCallbacks(
		func(_, _ string, cfg *models.ProviderConfig) { successEvent.Store(cfg) },
		func(_, _ string, _ error) {},
		func(_, _ string) {},
		func(_, _ string, _ int) {},
	)

	// 启动 Flow，间隔设为 300 秒，确保不会自然触发
	result, err := svc.StartFlow("testprovider")
	require.NoError(t, err)
	assert.Equal(t, "dev_trigger", result.DeviceCode)

	// 等待一小段时间确认 ticker 未触发
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(0), tokenPollCount.Load(), "ticker 不应在短间隔内触发")

	// 调用 TriggerPoll 触发立即轮询
	err = svc.TriggerPoll("dev_trigger")
	require.NoError(t, err)

	// 等待轮询完成
	require.Eventually(t, func() bool {
		return successEvent.Load() != nil
	}, 3*time.Second, 100*time.Millisecond, "TriggerPoll 应触发成功")

	assert.Equal(t, int32(1), tokenPollCount.Load(), "TriggerPoll 应只触发一次轮询")
	assert.Equal(t, "acc_trigger", successEvent.Load().AuthParams.OAuthAccessToken)
}

// TestOAuthDeviceFlowService_TriggerPoll_SessionNotFound 验证无效 deviceCode。
func TestOAuthDeviceFlowService_TriggerPoll_SessionNotFound(t *testing.T) {
	store := newMockProviderStore()
	svc := NewOAuthDeviceFlowServiceWithClient(store, nil, &http.Client{Timeout: 5 * time.Second})

	err := svc.TriggerPoll("non_existent_code")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

// TestOAuthDeviceFlowService_TriggerPoll_NonPendingSession 验证非 pending 状态不触发。
func TestOAuthDeviceFlowService_TriggerPoll_NonPendingSession(t *testing.T) {
	var tokenPollCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/device/auth" {
			_ = json.NewEncoder(w).Encode(DeviceAuthResponse{
				DeviceCode:      "dev_nonpending",
				UserCode:        "USER-NP",
				VerificationURI: "https://auth.example.com/verify",
				ExpiresIn:       600,
				Interval:        1,
			})
			return
		}
		if r.URL.Path == "/token" {
			tokenPollCount.Add(1)
			_ = json.NewEncoder(w).Encode(DeviceTokenResponse{
				AccessToken: "acc_np",
				ExpiresIn:   3600,
			})
		}
	}))
	defer server.Close()

	origConfig := deviceFlowConfigs["testprovider"]
	deviceFlowConfigs["testprovider"] = struct {
		DeviceAuthURL string
		TokenURL      string
		Scope         string
		EnvClientID   string
	}{
		DeviceAuthURL: server.URL + "/device/auth",
		TokenURL:      server.URL + "/token",
		Scope:         "test-scope",
		EnvClientID:   "TEST_CLIENT_ID",
	}
	defer func() {
		if origConfig.DeviceAuthURL == "" {
			delete(deviceFlowConfigs, "testprovider")
		} else {
			deviceFlowConfigs["testprovider"] = origConfig
		}
	}()

	store := newMockProviderStore()
	svc := NewOAuthDeviceFlowServiceWithClient(store, nil, server.Client())

	svc.SetCallbacks(
		func(_, _ string, _ *models.ProviderConfig) {},
		func(_, _ string, _ error) {},
		func(_, _ string) {},
		func(_, _ string, _ int) {},
	)

	// 启动 Flow 并等待完成
	result, err := svc.StartFlow("testprovider")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		st := svc.GetStatus("dev_nonpending")
		return st != nil && st.Status == DeviceFlowStatusSuccess
	}, 5*time.Second, 100*time.Millisecond)

	// 完成后再次 TriggerPoll，应无错误且不触发额外轮询
	prevCount := tokenPollCount.Load()
	err = svc.TriggerPoll(result.DeviceCode)
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, prevCount, tokenPollCount.Load(), "成功后的 TriggerPoll 不应触发额外轮询")
}
