// Package network 封装 HTTP 客户端，提供重试、超时、断路器能力。
package network

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/wire"
)

// HTTPClient 带增强功能的 HTTP 客户端封装。
// 内置指数退避重试（最多 3 次）和 semaphore 并发限制（最大 4 并发）。
type HTTPClient struct {
	client     *http.Client
	baseURL    string
	headers    map[string]string
	sem        chan struct{} // semaphore 通道，限制并发数
	maxRetries int           // 最大重试次数
}

// NewHTTPClient 创建增强型 HTTP 客户端。
func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:    baseURL,
		headers:    make(map[string]string),
		sem:        make(chan struct{}, 4), // 最大 4 并发
		maxRetries: 3,
	}
}

// WithTimeout 设置请求超时，返回自身以支持链式调用。
func (c *HTTPClient) WithTimeout(timeout time.Duration) *HTTPClient {
	c.client.Timeout = timeout
	return c
}

// WithHeader 设置默认请求头，返回自身以支持链式调用。
func (c *HTTPClient) WithHeader(key, value string) *HTTPClient {
	c.headers[key] = value
	return c
}

// WithMaxRetries 设置最大重试次数。
func (c *HTTPClient) WithMaxRetries(n int) *HTTPClient {
	if n >= 0 {
		c.maxRetries = n
	}
	return c
}

// Do 执行 HTTP 请求，内置重试、超时控制和并发限制。
func (c *HTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return c.DoWithRetry(ctx, req)
}

// DoWithRetry 执行 HTTP 请求，带指数退避重试。
// 对以下错误进行重试：网络超时、连接失败、5xx 服务端错误。
func (c *HTTPClient) DoWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	// 注入默认请求头
	for k, v := range c.headers {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}

	// 获取 semaphore 许可（限制并发）
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return nil, fmt.Errorf("request cancelled waiting for semaphore: %w", ctx.Err())
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避：500ms * 2^(attempt-1)，最大 5s
			delay := min(500*time.Millisecond*(1<<(attempt-1)), 5*time.Second)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, fmt.Errorf("retry cancelled: %w", ctx.Err())
			}
		}

		resp, err := c.client.Do(req.WithContext(ctx))
		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed: %w", attempt+1, err)
			// 网络错误通常可重试
			continue
		}

		// 5xx 服务端错误 和 429 速率限制 可重试；其他 4xx 客户端错误不可重试
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("attempt %d received server error: HTTP %d", attempt+1, resp.StatusCode)
			_ = resp.Body.Close()
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("all %d attempts failed, last error: %w", c.maxRetries+1, lastErr)
}

// NetworkSet 供 Wire 使用的 ProviderSet。
var NetworkSet = wire.NewSet(
	NewHTTPClient,
)
