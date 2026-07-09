package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hzhan516/medmemo/pkg/models"
)

// LocalAdapter 适配本地 Ollama / llama.cpp 端点。
type LocalAdapter struct {
	endpoint  string // 如 http://localhost:11434
	model     string
	maxTokens int
	client    *http.Client
}

// NewLocalAdapter 构造函数。
// timeout 为 HTTP 客户端整体超时；若传入 <=0 则默认 120 秒。
func NewLocalAdapter(endpoint, model string, maxTokens int, timeout time.Duration) *LocalAdapter {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	return &LocalAdapter{
		endpoint:  endpoint,
		model:     model,
		maxTokens: maxTokens,
		client:    &http.Client{Timeout: timeout},
	}
}

// ollamaMessage Ollama API 消息格式。
type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ollamaChatRequest Ollama /api/chat 请求体。
type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  ollamaOptions   `json:"options,omitempty"`
}

type ollamaOptions struct {
	NumPredict int `json:"num_predict,omitempty"`
}

// ollamaChatResponse Ollama /api/chat 响应体。
type ollamaChatResponse struct {
	Message         ollamaMessage `json:"message"`
	Done            bool          `json:"done"`
	Error           string        `json:"error,omitempty"`
	PromptEvalCount int           `json:"prompt_eval_count,omitempty"`
	EvalCount       int           `json:"eval_count,omitempty"`
}

// toOllamaMessages 领域消息转换为 Ollama 消息格式。
func toOllamaMessages(msgs []models.Message) []ollamaMessage {
	result := make([]ollamaMessage, len(msgs))
	for i, m := range msgs {
		result[i] = ollamaMessage{
			Role:    string(m.Role),
			Content: m.Content,
		}
	}
	return result
}

// Chat 发送非流式对话请求。
func (a *LocalAdapter) Chat(ctx context.Context, messages []models.Message) (string, error) {
	reqBody := ollamaChatRequest{
		Model:    a.model,
		Messages: toOllamaMessages(messages),
		Stream:   false,
		Options:  ollamaOptions{NumPredict: a.maxTokens},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ollama chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create ollama chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send ollama chat request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read ollama chat response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result ollamaChatResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal ollama response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("ollama error: %s", result.Error)
	}

	return result.Message.Content, nil
}

// StreamChat 发送流式对话请求，逐行读取 NDJSON 推送 content。
// 流式结束后返回 TokenUsage（若响应中未包含 usage 则为 nil）。
func (a *LocalAdapter) StreamChat(ctx context.Context, messages []models.Message, callback func(chunk string)) (*models.TokenUsage, error) {
	reqBody := ollamaChatRequest{
		Model:    a.model,
		Messages: toOllamaMessages(messages),
		Stream:   true,
		Options:  ollamaOptions{NumPredict: a.maxTokens},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ollama stream request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send ollama stream request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama stream returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenUsage *models.TokenUsage
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var chunk ollamaChatResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			// 忽略无法解析的行
			continue
		}

		if chunk.Error != "" {
			return nil, fmt.Errorf("ollama stream error: %s", chunk.Error)
		}

		content := chunk.Message.Content
		if content != "" {
			callback(content)
		}

		if chunk.Done {
			// Ollama 在最后一条中返回 prompt_eval_count / eval_count
			if chunk.PromptEvalCount > 0 || chunk.EvalCount > 0 {
				tokenUsage = &models.TokenUsage{
					PromptTokens:     chunk.PromptEvalCount,
					CompletionTokens: chunk.EvalCount,
					TotalTokens:      chunk.PromptEvalCount + chunk.EvalCount,
				}
			}
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ollama stream interrupted: %w", err)
	}

	return tokenUsage, nil
}

// CheckAvailability 探测 Ollama 服务可用性。
func (a *LocalAdapter) CheckAvailability(ctx context.Context) (bool, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.endpoint+"/api/tags", nil)
	if err != nil {
		return false, fmt.Sprintf("failed to create availability request: %v", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("ollama not reachable: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		return true, "available"
	}
	return false, fmt.Sprintf("unexpected status: %d", resp.StatusCode)
}
