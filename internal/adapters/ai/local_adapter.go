package ai

import (
	"context"
	"fmt"

	"github.com/medmemo/medmemo/pkg/models"
)

// LocalAdapter 适配本地 Ollama / llama.cpp 端点。
type LocalAdapter struct {
	endpoint string // 如 http://localhost:11434
	model    string
}

// NewLocalAdapter 构造函数。
func NewLocalAdapter(endpoint, model string) *LocalAdapter {
	return &LocalAdapter{endpoint: endpoint, model: model}
}

// Chat 实现 port.LLMClient。
func (a *LocalAdapter) Chat(ctx context.Context, messages []models.Message) (string, error) {
	// TODO(作者): 实现 Ollama API 调用 [Issue#010]
	return "", fmt.Errorf("LocalAdapter.Chat not implemented")
}

// StreamChat 实现 port.LLMClient。
func (a *LocalAdapter) StreamChat(ctx context.Context, messages []models.Message, callback func(chunk string)) error {
	// TODO(作者): 实现本地流式推理 [Issue#011]
	return fmt.Errorf("LocalAdapter.StreamChat not implemented")
}

// CheckAvailability 实现 port.LLMClient。
func (a *LocalAdapter) CheckAvailability(ctx context.Context) (bool, string) {
	// TODO(作者): 健康检查端点探测 [Issue#012]
	return false, "not implemented"
}
