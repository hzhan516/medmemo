// Package ai 实现 AI 模型客户端适配器簇。
// 本文件定义 OpenAICompatibleClient 使用的通用类型与 DTO。
package ai

import (
	"fmt"

	"github.com/hzhan516/medmemo/pkg/models"
)

// ChatRequest 表示对话请求 DTO。
type ChatRequest struct {
	Messages    []models.Message // 对话消息列表
	Temperature float64          // 可覆盖 ProviderConfig 中的温度值；为 0 时表示使用 ProviderConfig 的默认值
}

// StreamChunkType 表示流式输出块的类型。
type StreamChunkType string

const (
	// ChunkContent 表示正常内容输出。
	ChunkContent StreamChunkType = "content"
	// ChunkError 表示流式过程中发生的错误。
	ChunkError StreamChunkType = "error"
	// ChunkDone 表示流式输出正常结束。
	ChunkDone StreamChunkType = "done"
)

// StreamChunk 表示 SSE 流式响应的统一输出块。
// 供 TASK-037 流式响应统一处理层进一步包装扩展 Metadata。
type StreamChunk struct {
	Type    StreamChunkType // content | error | done
	Payload string          // 内容文本或错误信息
}

// LLMError 表示标准化的 LLM 调用错误，实现 error 接口。
type LLMError struct {
	Code      string // 错误码，如 invalid_api_key / rate_limit_exceeded
	Message   string // 用户可读错误信息
	Retryable bool   // 是否可重试（5xx / 网络超时为 true；4xx 为 false）
}

// Error 实现 error 接口。
func (e *LLMError) Error() string {
	return fmt.Sprintf("llm error [%s]: %s (retryable=%v)", e.Code, e.Message, e.Retryable)
}

// ModelInfo 表示 /v1/models 端点返回的单个模型信息。
type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// modelsListResponse 表示 /v1/models 的原始响应结构。
type modelsListResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

// chatCompletionRequest 表示 OpenAI Chat Completion 请求体。
type chatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Stream      bool      `json:"stream"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// message 表示单条对话消息（内部序列化用）。
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// sseDelta 表示 SSE 流式响应中的 delta 内容。
type sseDelta struct {
	Content string `json:"content"`
}

// sseChoice 表示 SSE 流式响应中的单个选择项。
type sseChoice struct {
	Delta        sseDelta `json:"delta"`
	FinishReason string   `json:"finish_reason"`
}

// sseChunk 表示 SSE 流式响应的单个数据块。
type sseChunk struct {
	Choices []sseChoice `json:"choices"`
	Error   *apiError   `json:"error,omitempty"`
}

// apiError 表示 API 返回的错误信息（复用 openai_adapter.go 中的定义以保持一致）。
type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

// Error 实现 error 接口。
func (e *apiError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("api error [%s]: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("api error: %s", e.Message)
}

// toInternalMessages 将领域消息转换为内部序列化消息格式。
func toInternalMessages(msgs []models.Message) []message {
	result := make([]message, len(msgs))
	for i, m := range msgs {
		result[i] = message{
			Role:    string(m.Role),
			Content: m.Content,
		}
	}
	return result
}
