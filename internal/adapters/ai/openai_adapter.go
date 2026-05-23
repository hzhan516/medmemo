// Package ai 实现 AI 模型客户端适配器簇。
package ai

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hzhan516/medmemo/pkg/models"
)

// OpenAIAdapter 适配 OpenAI 兼容 API。
type OpenAIAdapter struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewOpenAIAdapter 构造函数，返回具体类型供 Wire 绑定。
// timeout 为 HTTP 客户端整体超时（含连接、发送、读取响应体）；
// 若传入 <=0 则默认 120 秒，以覆盖流式长连接场景。
func NewOpenAIAdapter(apiKey, baseURL, model string, timeout time.Duration) *OpenAIAdapter {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &OpenAIAdapter{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			},
		},
	}
}

// chatRequest OpenAI Chat Completion 请求体。
type chatRequest struct {
	Model         string         `json:"model"`
	Messages      []message      `json:"messages"`
	Stream        bool           `json:"stream,omitempty"`
	StreamOptions *streamOptions `json:"stream_options,omitempty"`
}

// streamOptions 控制流式输出的附加选项。
type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// chatResponse OpenAI Chat Completion 非流式响应。
type chatResponse struct {
	Choices []choice  `json:"choices"`
	Error   *apiError `json:"error,omitempty"`
}

// choice 响应选择项。
type choice struct {
	Message      message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// usageInfo 表示 OpenAI 兼容 API 返回的 usage 统计。
type usageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// streamChunk SSE 流式响应数据块。
type streamChunk struct {
	Choices []streamChoice `json:"choices"`
	Usage   *usageInfo     `json:"usage,omitempty"`
	Error   *apiError      `json:"error,omitempty"`
}

// streamChoice 流式响应选择项。
type streamChoice struct {
	Delta        delta  `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

// delta 流式响应内容增量。
type delta struct {
	Content string `json:"content"`
}

// toMessages 领域消息转换为 OpenAI 消息格式。
func toMessages(msgs []models.Message) []message {
	result := make([]message, len(msgs))
	for i, m := range msgs {
		result[i] = message{
			Role:    string(m.Role),
			Content: m.Content,
		}
	}
	return result
}

// mapAPIError HTTP 状态码和 API 错误映射为用户友好错误。
func mapAPIError(statusCode int, apiErr *apiError) error {
	switch statusCode {
	case http.StatusUnauthorized:
		if apiErr != nil {
			return fmt.Errorf("API 认证失败，请检查 API Key 是否有效: %w", apiErr)
		}
		return fmt.Errorf("API 认证失败，请检查 API Key 是否有效")
	case http.StatusTooManyRequests:
		if apiErr != nil {
			return fmt.Errorf("请求过于频繁，请稍后再试: %w", apiErr)
		}
		return fmt.Errorf("请求过于频繁，请稍后再试")
	case http.StatusNotFound:
		if apiErr != nil {
			return fmt.Errorf("请求的模型不存在或接口地址错误: %w", apiErr)
		}
		return fmt.Errorf("请求的模型不存在或接口地址错误")
	default:
		if apiErr != nil {
			return fmt.Errorf("API 调用失败: %w", apiErr)
		}
		return fmt.Errorf("API 调用失败，HTTP %d", statusCode)
	}
}

// doWithRetry 执行 HTTP 请求，遇到 429 速率限制时指数退避重试。
// 最多重试 3 次，退避间隔：500ms、1s、2s（最大 5s）。
func (a *OpenAIAdapter) doWithRetry(req *http.Request) (*http.Response, error) {
	const maxRetries = 3
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 重试前重置请求体
			if req.GetBody != nil {
				req.Body, _ = req.GetBody() // GetBody 返回的 error 通常为 nil（请求体已在前面正确构造）
			} else if req.Body != nil {
				if seeker, ok := req.Body.(io.Seeker); ok {
					_, _ = seeker.Seek(0, io.SeekStart) // 对内存 buffer 执行 Seek 不会失败
				}
			}
			delay := min(500*time.Millisecond*(1<<(attempt-1)), 5*time.Second)
			select {
			case <-time.After(delay):
			case <-req.Context().Done():
				return nil, fmt.Errorf("retry cancelled: %w", req.Context().Err())
			}
		}

		resp, err := a.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("attempt %d: rate limited (HTTP 429)", attempt+1)
			_ = resp.Body.Close() // 429 重试前关闭响应体，关闭错误不影响重试逻辑
			continue
		}

		return resp, nil
	}
	return nil, fmt.Errorf("请求过于频繁，请稍后再试 (HTTP 429, 已重试 %d 次): %w", maxRetries, lastErr)
}

// Chat 发送非流式对话请求。
func (a *OpenAIAdapter) Chat(ctx context.Context, messages []models.Message) (string, error) {
	reqBody := chatRequest{
		Model:    a.model,
		Messages: toMessages(messages),
		Stream:   false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create chat request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.doWithRetry(req)
	if err != nil {
		return "", fmt.Errorf("failed to send chat request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read chat response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp chatResponse
		_ = json.Unmarshal(body, &errResp) // 非 200 响应尝试解析错误体，解析失败不影响主错误映射
		return "", mapAPIError(resp.StatusCode, errResp.Error)
	}

	var result chatResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal chat response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response from API: no choices returned")
	}

	return result.Choices[0].Message.Content, nil
}

// StreamChat 发送 SSE 流式对话请求。
// 流式结束后返回 TokenUsage（若响应中未包含 usage 则为 nil）。
func (a *OpenAIAdapter) StreamChat(ctx context.Context, messages []models.Message, callback func(chunk string)) (*models.TokenUsage, error) {
	reqBody := chatRequest{
		Model:         a.model,
		Messages:      toMessages(messages),
		Stream:        true,
		StreamOptions: &streamOptions{IncludeUsage: true},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal stream request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create stream request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := a.doWithRetry(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send stream request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errResp streamChunk
		_ = json.Unmarshal(body, &errResp) // 非 200 响应尝试解析错误体，解析失败不影响主错误映射
		return nil, mapAPIError(resp.StatusCode, errResp.Error)
	}

	return a.readStream(resp.Body, callback)
}

// readStream 读取 SSE 响应体，逐条解析 chunk 并调用回调。
func (a *OpenAIAdapter) readStream(body io.Reader, callback func(chunk string)) (*models.TokenUsage, error) {
	var tokenUsage *models.TokenUsage
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		chunk, done, err := parseStreamLine(line)
		if err != nil {
			return nil, err
		}
		if done {
			break
		}
		if chunk == nil {
			continue
		}

		if chunk.Usage != nil {
			tokenUsage = extractTokenUsage(chunk.Usage)
		}

		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				callback(content)
			}
			if chunk.Choices[0].FinishReason == "stop" {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("stream interrupted: %w", err)
	}

	return tokenUsage, nil
}

// parseStreamLine 解析单行 SSE 数据，返回解析后的 chunk、是否结束、错误。
func parseStreamLine(line string) (*streamChunk, bool, error) {
	const sseDataPrefix = "data: "
	if len(line) <= len(sseDataPrefix) || !strings.HasPrefix(line, sseDataPrefix) {
		return nil, false, nil
	}
	if line == "data: [DONE]" {
		return nil, true, nil
	}

	data := line[len(sseDataPrefix):]
	var chunk streamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return nil, false, nil
	}
	if chunk.Error != nil {
		return nil, false, fmt.Errorf("stream error: %w", chunk.Error)
	}
	return &chunk, false, nil
}

// extractTokenUsage 从 usageInfo 结构提取 TokenUsage 模型。
func extractTokenUsage(u *usageInfo) *models.TokenUsage {
	return &models.TokenUsage{
		PromptTokens:     u.PromptTokens,
		CompletionTokens: u.CompletionTokens,
		TotalTokens:      u.TotalTokens,
	}
}

// CheckAvailability 轻量级连通性检测。
func (a *OpenAIAdapter) CheckAvailability(ctx context.Context) (bool, string) {
	if a.apiKey == "" {
		return false, "API key not configured"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/v1/models", nil)
	if err != nil {
		return false, fmt.Sprintf("failed to create availability request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, "available"
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return false, "authentication failed: invalid API key"
	}

	return false, fmt.Sprintf("unexpected status: %d", resp.StatusCode)
}
