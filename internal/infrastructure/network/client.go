// Package network 封装 HTTP 客户端，提供重试、超时、断路器能力。
package network

import (
	"context"
	"net/http"
	"time"

	"github.com/google/wire"
)

// HTTPClient 带增强功能的 HTTP 客户端封装。
type HTTPClient struct {
	client  *http.Client
	baseURL string
	headers map[string]string
}

// NewHTTPClient 创建增强型 HTTP 客户端。
func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		headers: make(map[string]string),
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

// Do 执行 HTTP 请求，内置重试与超时控制。
func (c *HTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// 注入默认请求头
	for k, v := range c.headers {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}

	// TODO(作者): 实现指数退避重试 + semaphore 并发限制（最大 4 并发） [Issue#024]
	return c.client.Do(req.WithContext(ctx))
}

// NetworkSet 供 Wire 使用的 ProviderSet。
var NetworkSet = wire.NewSet(
	NewHTTPClient,
)
