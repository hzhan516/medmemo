// Package ai 实现 AI 模型客户端适配器簇。
// 适配器实现 application/port 中定义的 LLMClient 接口。
package ai

import (
	"context"
	"fmt"

	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/pkg/models"
)

// OpenAIAdapter 适配 OpenAI 兼容 API（含 Kimi、Qwen、SiliconFlow）。
type OpenAIAdapter struct {
	apiKey  string
	baseURL string
	model   string
	client  port.RecordStore // 复用底层 HTTP 客户端抽象（如需要可替换为具体 *http.Client）
}

// NewOpenAIAdapter 构造函数，返回具体类型供 Wire 绑定。
func NewOpenAIAdapter(apiKey, baseURL, model string) *OpenAIAdapter {
	return &OpenAIAdapter{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
	}
}

// Chat 实现 port.LLMClient。
func (a *OpenAIAdapter) Chat(ctx context.Context, messages []models.Message) (string, error) {
	// TODO(作者): 实现 OpenAI-compatible API 非流式调用 [Issue#008]
	return "", fmt.Errorf("OpenAIAdapter.Chat not implemented")
}

// StreamChat 实现 port.LLMClient。
func (a *OpenAIAdapter) StreamChat(ctx context.Context, messages []models.Message, callback func(chunk string)) error {
	// TODO(作者): 实现 SSE 流式解析与分句缓冲合规检测 [Issue#009]
	return fmt.Errorf("OpenAIAdapter.StreamChat not implemented")
}

// CheckAvailability 实现 port.LLMClient。
func (a *OpenAIAdapter) CheckAvailability(ctx context.Context) (bool, string) {
	if a.apiKey == "" {
		return false, "API key not configured"
	}
	return true, "available"
}
