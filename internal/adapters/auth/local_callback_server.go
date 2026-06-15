// Package auth 实现本地 HTTP 回调服务器。
// 用于 OAuth Device Flow 完成后的本地通知，以及未来 Authorization Code Flow 扩展。
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CallbackResult 表示本地回调服务器接收到的授权结果。
type CallbackResult struct {
	Code             string // 授权码（Authorization Code Flow 使用）
	State            string // CSRF state 参数
	Error            string // OAuth 错误码
	ErrorDescription string // OAuth 错误描述
}

// LocalCallbackServer 是轻量级 localhost HTTP 回调服务器。
// 每次 Device Flow 会话独立创建，接收回调后自动关闭。
type LocalCallbackServer struct {
	mu       sync.Mutex
	server   *http.Server
	listener net.Listener
	state    string
	result   *CallbackResult
	done     chan struct{}
}

// NewLocalCallbackServer 创建新的回调服务器实例。
func NewLocalCallbackServer() *LocalCallbackServer {
	return &LocalCallbackServer{
		done: make(chan struct{}),
	}
}

// Start 启动 HTTP 服务器，绑定动态端口。
// 使用操作系统分配的端口（127.0.0.1:0），避免伪随机端口选择的冲突与可预测性问题。
// 返回实际绑定的端口号。
func (s *LocalCallbackServer) Start() (int, error) {
	// 生成随机 CSRF state
	state, err := generateState()
	if err != nil {
		return 0, fmt.Errorf("failed to generate csrf state: %w", err)
	}
	s.state = state

	// 使用操作系统分配端口，避免冲突与可预测性
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("failed to bind to localhost: %w", err)
	}

	s.listener = listener
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		_ = s.server.Serve(listener)
	}()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("listener address is not *net.TCPAddr, got %T", listener.Addr())
	}
	return addr.Port, nil
}

// Stop 关闭 HTTP 服务器和监听器。
func (s *LocalCallbackServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.server.Shutdown(ctx)
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	// 确保 done channel 被关闭，避免 WaitForCallback 永久阻塞
	select {
	case <-s.done:
		// 已关闭
	default:
		close(s.done)
	}
	return nil
}

// WaitForCallback 阻塞等待回调到达或超时。
// 超时返回 error；回调到达返回 CallbackResult。
func (s *LocalCallbackServer) WaitForCallback(timeout time.Duration) (*CallbackResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-s.done:
		s.mu.Lock()
		result := s.result
		s.mu.Unlock()
		if result == nil {
			return nil, fmt.Errorf("callback server closed without result")
		}
		return result, nil
	case <-ctx.Done():
		_ = s.Stop()
		return nil, fmt.Errorf("callback wait timeout after %v", timeout)
	}
}

// GetRedirectURI 返回完整的回调 URL，供 OAuth 授权请求使用。
func (s *LocalCallbackServer) GetRedirectURI() string {
	if s.listener == nil {
		return ""
	}
	addr, ok := s.listener.Addr().(*net.TCPAddr)
	if !ok {
		return ""
	}
	port := addr.Port
	return fmt.Sprintf("http://127.0.0.1:%d/callback", port)
}

// GetState 返回本次会话的 CSRF state 参数。
func (s *LocalCallbackServer) GetState() string {
	return s.state
}

// handleCallback 处理 OAuth 回调请求。
func (s *LocalCallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	receivedState := query.Get("state")

	// CSRF 校验：state 不匹配时返回 403，但不关闭 server（可能是扫描/探测请求）
	if receivedState != s.state {
		http.Error(w, "invalid state", http.StatusForbidden)
		return
	}

	result := &CallbackResult{
		State:            receivedState,
		Code:             query.Get("code"),
		Error:            query.Get("error"),
		ErrorDescription: query.Get("error_description"),
	}

	s.mu.Lock()
	s.result = result
	s.mu.Unlock()

	// 关闭 done channel，通知 WaitForCallback
	select {
	case <-s.done:
		// 已关闭
	default:
		close(s.done)
	}

	// 异步延迟关闭 server，给 HTTP 响应留出时间
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = s.Stop()
	}()

	// 向浏览器返回友好页面
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if result.Error != "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(errorHTMLPage(result.Error, result.ErrorDescription)))
		return
	}
	_, _ = w.Write([]byte(successHTMLPage()))
}

// generateState 生成 32 字节随机 CSRF state。
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// successHTMLPage 返回授权成功时的浏览器展示页。
func successHTMLPage() string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>授权完成 - MedMemo</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#f5f5f5}
.card{background:#fff;padding:40px 48px;border-radius:16px;box-shadow:0 4px 20px rgba(0,0,0,0.08);text-align:center;max-width:360px}
.icon{width:64px;height:64px;margin:0 auto 20px;background:#4caf50;border-radius:50%;display:flex;align-items:center;justify-content:center;color:#fff;font-size:32px}
h1{margin:0 0 12px;font-size:20px;color:#333}
p{margin:0;color:#666;line-height:1.6;font-size:14px}
</style>
</head>
<body>
<div class="card">
<div class="icon">&#10003;</div>
<h1>授权完成</h1>
<p>您已成功授权 MedMemo 访问您的账户。<br>请返回应用继续使用。</p>
</div>
</body>
</html>`
}

// errorHTMLPage 返回授权失败时的浏览器展示页。
func errorHTMLPage(errCode, errDesc string) string {
	desc := errDesc
	if desc == "" {
		desc = "授权过程中发生错误，请返回应用重试。"
	}
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>授权失败 - MedMemo</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#f5f5f5}
.card{background:#fff;padding:40px 48px;border-radius:16px;box-shadow:0 4px 20px rgba(0,0,0,0.08);text-align:center;max-width:360px}
.icon{width:64px;height:64px;margin:0 auto 20px;background:#f44336;border-radius:50%;display:flex;align-items:center;justify-content:center;color:#fff;font-size:32px}
h1{margin:0 0 12px;font-size:20px;color:#333}
p{margin:0 0 8px;color:#666;line-height:1.6;font-size:14px}
code{display:inline-block;background:#f0f0f0;padding:2px 8px;border-radius:4px;font-size:12px;color:#555}
</style>
</head>
<body>
<div class="card">
<div class="icon">&#10007;</div>
<h1>授权失败</h1>
<p>{{DESC}}</p>
<code>{{CODE}}</code>
</div>
</body>
</html>`
	html = strings.ReplaceAll(html, "{{DESC}}", desc)
	html = strings.ReplaceAll(html, "{{CODE}}", errCode)
	return html
}
