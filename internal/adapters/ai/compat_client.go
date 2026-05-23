// Package ai 实现 AI 模型客户端适配器簇。
package ai

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hzhan516/medmemo/pkg/models"
)

// OpenAICompatibleClient 通用 OpenAI-compatible HTTP 客户端。
// 所有连接参数通过 ProviderConfig 动态传入，不硬编码厂商信息。
type OpenAICompatibleClient struct {
	client *http.Client
}

// NewOpenAICompatibleClient 创建通用客户端，使用默认 HTTP 配置。
func NewOpenAICompatibleClient() *OpenAICompatibleClient {
	return &OpenAICompatibleClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			},
		},
	}
}

// NewOpenAICompatibleClientWithHTTPClient 使用外部提供的 HTTP 客户端创建实例。
func NewOpenAICompatibleClientWithHTTPClient(c *http.Client) *OpenAICompatibleClient {
	return &OpenAICompatibleClient{client: c}
}

// Chat 发送流式对话请求，返回 StreamChunk channel。
// 连接阶段错误直接返回 error；流读取阶段错误通过 channel 发送 ChunkError。
// channel 带 1 缓冲，由内部 goroutine 负责关闭，消费者只需 range 读取。
func (c *OpenAICompatibleClient) Chat(ctx context.Context, req ChatRequest, config models.ProviderConfig) (<-chan StreamChunk, error) {
	if config.APIHost == "" {
		return nil, &LLMError{Code: "missing_api_host", Message: "APIHost 不能为空", Retryable: false}
	}
	if config.ModelID == "" {
		return nil, &LLMError{Code: "missing_model_id", Message: "ModelID 不能为空", Retryable: false}
	}

	temp := config.Temperature
	if req.Temperature > 0 {
		temp = req.Temperature
	}
	if temp == 0 {
		temp = 0.7 // 默认温度
	}

	reqBody := chatCompletionRequest{
		Model:       config.ModelID,
		Messages:    toInternalMessages(req.Messages),
		Stream:      true,
		Temperature: temp,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, &LLMError{Code: "marshal_failed", Message: fmt.Sprintf("序列化请求失败: %v", err), Retryable: false}
	}

	url := config.APIHost + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, &LLMError{Code: "request_failed", Message: fmt.Sprintf("构造请求失败: %v", err), Retryable: false}
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	authToken, err := config.ResolveAuthToken()
	if err != nil {
		return nil, &LLMError{Code: "auth_failed", Message: fmt.Sprintf("认证失败: %v", err), Retryable: false}
	}
	if authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+authToken)
	}

	client := c.client
	if config.TimeoutMs > 0 {
		client = &http.Client{
			Transport: client.Transport,
			Timeout:   time.Duration(config.TimeoutMs) * time.Millisecond,
		}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		retryable := isNetworkError(err)
		return nil, &LLMError{Code: "network_error", Message: fmt.Sprintf("发送请求失败: %v", err), Retryable: retryable}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var errResp sseChunk
		_ = json.Unmarshal(body, &errResp)
		code, msg, retryable := classifyHTTPError(resp.StatusCode, errResp.Error)
		return nil, &LLMError{Code: code, Message: msg, Retryable: retryable}
	}

	ch := make(chan StreamChunk, 1)
	go c.readSSE(ctx, resp, ch)
	return ch, nil
}

// readSSE 在独立 goroutine 中读取 SSE 流并写入 channel。
// 负责在流结束、出错或 context 取消时关闭 channel。
func (c *OpenAICompatibleClient) readSSE(ctx context.Context, resp *http.Response, ch chan<- StreamChunk) {
	defer close(ch)
	defer resp.Body.Close()

	// context 取消时关闭 resp.Body，中断 scanner 阻塞读取
	stopEarly := context.AfterFunc(ctx, func() {
		resp.Body.Close()
	})
	defer stopEarly()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			ch <- StreamChunk{Type: ChunkError, Payload: fmt.Sprintf("context cancelled: %v", ctx.Err())}
			return
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}
		if line == "data: [DONE]" {
			ch <- StreamChunk{Type: ChunkDone, Payload: ""}
			return
		}
		const sseDataPrefix = "data: "
		if len(line) <= len(sseDataPrefix) || !strings.HasPrefix(line, sseDataPrefix) {
			continue
		}

		data := line[len(sseDataPrefix):]
		var chunk sseChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// 忽略无法解析的行
			continue
		}

		if chunk.Error != nil {
			ch <- StreamChunk{Type: ChunkError, Payload: chunk.Error.Error()}
			return
		}

		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				ch <- StreamChunk{Type: ChunkContent, Payload: content}
			}
			if chunk.Choices[0].FinishReason == "stop" {
				ch <- StreamChunk{Type: ChunkDone, Payload: ""}
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Type: ChunkError, Payload: fmt.Sprintf("stream interrupted: %v", err)}
	} else {
		ch <- StreamChunk{Type: ChunkDone, Payload: ""}
	}
}

// FetchModels 调用 /v1/models 端点拉取可用模型列表。
func (c *OpenAICompatibleClient) FetchModels(ctx context.Context, config models.ProviderConfig) ([]ModelInfo, error) {
	if config.APIHost == "" {
		return nil, &LLMError{Code: "missing_api_host", Message: "APIHost 不能为空", Retryable: false}
	}

	url := config.APIHost + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &LLMError{Code: "request_failed", Message: fmt.Sprintf("构造请求失败: %v", err), Retryable: false}
	}

	authToken, err := config.ResolveAuthToken()
	if err != nil {
		return nil, &LLMError{Code: "auth_failed", Message: fmt.Sprintf("认证失败: %v", err), Retryable: false}
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	client := c.client
	if config.TimeoutMs > 0 {
		client = &http.Client{
			Transport: client.Transport,
			Timeout:   time.Duration(config.TimeoutMs) * time.Millisecond,
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		retryable := isNetworkError(err)
		return nil, &LLMError{Code: "network_error", Message: fmt.Sprintf("发送请求失败: %v", err), Retryable: retryable}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &LLMError{Code: "read_failed", Message: fmt.Sprintf("读取响应失败: %v", err), Retryable: true}
	}

	if resp.StatusCode != http.StatusOK {
		var errResp sseChunk
		_ = json.Unmarshal(body, &errResp)
		code, msg, retryable := classifyHTTPError(resp.StatusCode, errResp.Error)
		return nil, &LLMError{Code: code, Message: msg, Retryable: retryable}
	}

	var result modelsListResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, &LLMError{Code: "unmarshal_failed", Message: fmt.Sprintf("解析响应失败: %v", err), Retryable: false}
	}

	return result.Data, nil
}

// classifyHTTPError 根据 HTTP 状态码分类错误。
func classifyHTTPError(statusCode int, apiErr *apiError) (code, message string, retryable bool) {
	if apiErr != nil && apiErr.Code != "" {
		code = apiErr.Code
	} else {
		code = fmt.Sprintf("http_%d", statusCode)
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return code, "API 认证失败，请检查 API Key 是否有效", false
	case http.StatusForbidden:
		return code, "API 访问被拒绝，请检查权限配置", false
	case http.StatusTooManyRequests:
		return code, "请求过于频繁，请稍后再试", true
	case http.StatusNotFound:
		return code, "请求的端点或模型不存在", false
	default:
		if statusCode >= 500 {
			msg := "服务端错误"
			if apiErr != nil {
				msg = apiErr.Message
			}
			return code, msg, true
		}
		msg := "API 调用失败"
		if apiErr != nil {
			msg = apiErr.Message
		}
		return code, msg, false
	}
}

// isNetworkError 判断错误是否为网络层可重试错误。
// 超时、连接失败等可重试；TLS 证书错误不可重试。
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if urlErr, ok := errors.AsType[*url.Error](err); ok {
		if urlErr.Timeout() {
			return true
		}
		// TLS 证书错误不可重试
		var tlsErr *tls.CertificateVerificationError
		return !errors.As(urlErr.Err, &tlsErr)
	}
	return errors.Is(err, context.DeadlineExceeded)
}
