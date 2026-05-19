// Package ai 实现 AI 模型客户端适配器簇。
// 适配器实现 application/port 中定义的 LLMClient 接口。
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
	"time"

	"github.com/medmemo/medmemo/pkg/models"
)

// OpenAIAdapter 适配 OpenAI 兼容 API（含 Kimi、Qwen、SiliconFlow）。
type OpenAIAdapter struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewOpenAIAdapter 构造函数，返回具体类型供 Wire 绑定。
func NewOpenAIAdapter(apiKey, baseURL, model string) *OpenAIAdapter {
	return &OpenAIAdapter{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			},
		},
	}
}

// chatRequest 表示 OpenAI Chat Completion 请求体。
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}

// message 表示单条对话消息。
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse 表示 OpenAI Chat Completion 非流式响应。
type chatResponse struct {
	Choices []choice  `json:"choices"`
	Error   *apiError `json:"error,omitempty"`
}

// choice 表示响应中的选择项。
type choice struct {
	Message      message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// streamChunk 表示 SSE 流式响应的单个数据块。
type streamChunk struct {
	Choices []streamChoice `json:"choices"`
	Error   *apiError      `json:"error,omitempty"`
}

// streamChoice 表示流式响应中的选择项。
type streamChoice struct {
	Delta        delta  `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

// delta 表示流式响应中的内容增量。
type delta struct {
	Content string `json:"content"`
}

// apiError 表示 API 返回的错误信息。
type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

// Error 实现 error 接口，便于上层 errors.As 识别。
func (e *apiError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("api error [%s]: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("api error: %s", e.Message)
}

// toMessages 将领域消息转换为 OpenAI 消息格式。
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

// mapAPIError 将 HTTP 状态码和 API 错误映射为用户友好的错误信息。
func mapAPIError(statusCode int, apiErr *apiError) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("API 认证失败，请检查 API Key 是否有效: %w", apiErr)
	case http.StatusTooManyRequests:
		return fmt.Errorf("请求过于频繁，请稍后再试: %w", apiErr)
	case http.StatusNotFound:
		return fmt.Errorf("请求的模型不存在或接口地址错误: %w", apiErr)
	default:
		if apiErr != nil {
			return fmt.Errorf("API 调用失败: %w", apiErr)
		}
		return fmt.Errorf("API 调用失败，HTTP %d", statusCode)
	}
}

// Chat 实现 port.LLMClient，发送非流式对话请求。
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

	resp, err := a.client.Do(req)
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
		_ = json.Unmarshal(body, &errResp)
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

// StreamChat 实现 port.LLMClient，发送 SSE 流式对话请求。
func (a *OpenAIAdapter) StreamChat(ctx context.Context, messages []models.Message, callback func(chunk string)) error {
	reqBody := chatRequest{
		Model:    a.model,
		Messages: toMessages(messages),
		Stream:   true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal stream request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create stream request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send stream request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errResp streamChunk
		_ = json.Unmarshal(body, &errResp)
		return mapAPIError(resp.StatusCode, errResp.Error)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if line == "data: [DONE]" {
			break
		}
		if !bytes.HasPrefix([]byte(line), []byte("data: ")) {
			continue
		}

		data := line[len("data: "):]
		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// 忽略无法解析的行，继续处理后续内容
			continue
		}

		if chunk.Error != nil {
			return fmt.Errorf("stream error: %w", chunk.Error)
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
		return fmt.Errorf("stream interrupted: %w", err)
	}

	return nil
}

// CheckAvailability 实现 port.LLMClient，轻量级连通性检测。
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
