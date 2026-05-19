package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/medmemo/medmemo/pkg/models"
)

// LocalAdapter 适配本地 Ollama / llama.cpp 端点。
// 实现 OpenAI-compatible 的 Chat Completion 接口，通过 HTTP 调用 Ollama REST API。
type LocalAdapter struct {
	endpoint string // 如 http://localhost:11434
	model    string
	client   *http.Client
}

// NewLocalAdapter 构造函数。
func NewLocalAdapter(endpoint, model string) *LocalAdapter {
	return &LocalAdapter{
		endpoint: endpoint,
		model:    model,
		client:   &http.Client{Timeout: 0}, // 流式请求不设超时，由 context 控制
	}
}

// ollamaMessage 表示 Ollama API 的消息格式。
type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ollamaChatRequest 表示 Ollama /api/chat 请求体。
type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

// ollamaChatResponse 表示 Ollama /api/chat 非流式响应。
type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
	Error   string        `json:"error,omitempty"`
}

// toOllamaMessages 将领域消息转换为 Ollama 消息格式。
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

// Chat 实现 port.LLMClient，发送非流式对话请求。
func (a *LocalAdapter) Chat(ctx context.Context, messages []models.Message) (string, error) {
	reqBody := ollamaChatRequest{
		Model:    a.model,
		Messages: toOllamaMessages(messages),
		Stream:   false,
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
	defer resp.Body.Close()

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

// StreamChat 实现 port.LLMClient，发送流式对话请求。
// 逐行读取 Ollama 返回的 NDJSON，通过 callback 推送 content。
func (a *LocalAdapter) StreamChat(ctx context.Context, messages []models.Message, callback func(chunk string)) error {
	reqBody := ollamaChatRequest{
		Model:    a.model,
		Messages: toOllamaMessages(messages),
		Stream:   true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal ollama stream request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create ollama stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send ollama stream request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama stream returned HTTP %d: %s", resp.StatusCode, string(body))
	}

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
			return fmt.Errorf("ollama stream error: %s", chunk.Error)
		}

		content := chunk.Message.Content
		if content != "" {
			callback(content)
		}

		if chunk.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ollama stream interrupted: %w", err)
	}

	return nil
}

// CheckAvailability 实现 port.LLMClient，探测 Ollama 服务可用性。
func (a *LocalAdapter) CheckAvailability(ctx context.Context) (bool, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.endpoint+"/api/tags", nil)
	if err != nil {
		return false, fmt.Sprintf("failed to create availability request: %v", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("ollama not reachable: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, "available"
	}
	return false, fmt.Sprintf("unexpected status: %d", resp.StatusCode)
}
