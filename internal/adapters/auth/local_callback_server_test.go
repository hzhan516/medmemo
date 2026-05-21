package auth

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalCallbackServer_Start_DynamicPort(t *testing.T) {
	server := NewLocalCallbackServer()
	port, err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// 端口应在 IANA 动态端口范围内
	assert.GreaterOrEqual(t, port, 49152)
	assert.LessOrEqual(t, port, 65535)

	// 验证 server 确实在监听
	assert.NotEmpty(t, server.GetRedirectURI())
	assert.True(t, strings.HasPrefix(server.GetRedirectURI(), "http://127.0.0.1:"))
	assert.True(t, strings.HasSuffix(server.GetRedirectURI(), "/callback"))

	// state 应已生成
	assert.NotEmpty(t, server.GetState())
	assert.Len(t, server.GetState(), 64) // 32 字节 hex = 64 字符
}

func TestLocalCallbackServer_Callback_Success(t *testing.T) {
	server := NewLocalCallbackServer()
	_, err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	state := server.GetState()
	redirectURI := server.GetRedirectURI()

	// 在后台等待回调
	resultCh := make(chan *CallbackResult, 1)
	errCh := make(chan error, 1)
	go func() {
		res, err := server.WaitForCallback(5 * time.Second)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- res
	}()

	// 模拟浏览器发送成功回调
	client := &http.Client{Timeout: 3 * time.Second}
	callbackURL := fmt.Sprintf("%s?code=auth_code_123&state=%s", redirectURI, state)
	resp, err := client.Get(callbackURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "授权完成")

	// 验证 WaitForCallback 收到正确结果
	select {
	case res := <-resultCh:
		assert.Equal(t, "auth_code_123", res.Code)
		assert.Equal(t, state, res.State)
		assert.Empty(t, res.Error)
	case err := <-errCh:
		t.Fatalf("WaitForCallback failed: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("WaitForCallback timeout")
	}
}

func TestLocalCallbackServer_Callback_InvalidState(t *testing.T) {
	server := NewLocalCallbackServer()
	_, err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	redirectURI := server.GetRedirectURI()

	// 使用错误的 state 发送回调
	client := &http.Client{Timeout: 3 * time.Second}
	callbackURL := fmt.Sprintf("%s?code=xxx&state=invalid_state", redirectURI)
	resp, err := client.Get(callbackURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// server 不应被关闭，done channel 仍开放
	select {
	case <-server.done:
		t.Fatal("server should not be closed on invalid state")
	default:
		// 正确：server 仍在运行
	}
}

func TestLocalCallbackServer_Callback_ErrorResponse(t *testing.T) {
	server := NewLocalCallbackServer()
	_, err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	state := server.GetState()
	redirectURI := server.GetRedirectURI()

	resultCh := make(chan *CallbackResult, 1)
	go func() {
		res, _ := server.WaitForCallback(5 * time.Second)
		if res != nil {
			resultCh <- res
		}
	}()

	// 模拟 OAuth 错误回调
	client := &http.Client{Timeout: 3 * time.Second}
	callbackURL := fmt.Sprintf("%s?error=access_denied&error_description=user+denied&state=%s", redirectURI, state)
	resp, err := client.Get(callbackURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, string(body), "授权失败")

	select {
	case res := <-resultCh:
		assert.Equal(t, "access_denied", res.Error)
		assert.Equal(t, "user denied", res.ErrorDescription)
		assert.Empty(t, res.Code)
	case <-time.After(3 * time.Second):
		t.Fatal("WaitForCallback timeout")
	}
}

func TestLocalCallbackServer_WaitForCallback_Timeout(t *testing.T) {
	server := NewLocalCallbackServer()
	_, err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// 设置很短超时，且不发回调
	res, err := server.WaitForCallback(100 * time.Millisecond)
	assert.Nil(t, res)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestLocalCallbackServer_Stop(t *testing.T) {
	server := NewLocalCallbackServer()
	port, err := server.Start()
	require.NoError(t, err)

	// 确认 server 在运行
	redirectURI := server.GetRedirectURI()
	assert.NotEmpty(t, redirectURI)

	// 停止 server
	err = server.Stop()
	assert.NoError(t, err)

	// 确认端口已释放：尝试连接应失败
	client := &http.Client{Timeout: 1 * time.Second}
	_, err = client.Get(fmt.Sprintf("http://127.0.0.1:%d/callback", port))
	assert.Error(t, err)
}

func TestLocalCallbackServer_MultipleServers_DifferentPorts(t *testing.T) {
	// 验证多个 callback server 可同时运行在不同端口
	server1 := NewLocalCallbackServer()
	port1, err := server1.Start()
	require.NoError(t, err)
	defer server1.Stop()

	server2 := NewLocalCallbackServer()
	port2, err := server2.Start()
	require.NoError(t, err)
	defer server2.Stop()

	assert.NotEqual(t, port1, port2)
	assert.NotEqual(t, server1.GetState(), server2.GetState())
}

func TestLocalCallbackServer_Callback_MethodNotAllowed(t *testing.T) {
	server := NewLocalCallbackServer()
	_, err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	redirectURI := server.GetRedirectURI()

	// POST 请求应返回 405
	resp, err := http.Post(redirectURI, "application/json", strings.NewReader("{}"))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}
