// Package port 定义应用层的端口（接口契约）。
// 所有外部系统的抽象在此声明，由 adapters 层提供具体实现。
package port

import (
	"context"

	"github.com/medmemo/medmemo/pkg/models"
)

// LLMClient 定义大语言模型客户端接口。
// 实现者需支持云端 API（Kimi/OpenAI/Qwen）和本地端点（Ollama/llama.cpp）。
type LLMClient interface {
	// Chat 发送非流式对话请求，返回完整回复。
	Chat(ctx context.Context, messages []models.Message) (string, error)

	// StreamChat 发送流式对话请求，通过 callback 逐块推送内容。
	// 流式结束后返回 TokenUsage（若 Provider 未返回 usage 则为 nil）。
	StreamChat(ctx context.Context, messages []models.Message, callback func(chunk string)) (*models.TokenUsage, error)

	// CheckAvailability 检查当前模型是否可用，返回状态与原因。
	CheckAvailability(ctx context.Context) (bool, string)
}

// LLMClientFactory 根据 ProviderConfig 动态创建 LLMClient。
// 支持运行时根据用户配置的 provider 创建对应的适配器。
type LLMClientFactory interface {
	// CreateClient 根据 ProviderConfig 创建对应的 LLMClient。
	CreateClient(providerConfig *models.ProviderConfig) (LLMClient, error)
}

// NERDetector 定义命名实体识别检测端口。
// 实现者基于 DistilBERT-ONNX 等深度学习模型识别人名、地点、机构名等实体。
type NERDetector interface {
	// Predict 对文本执行 NER 推理，返回识别到的实体列表。
	Predict(ctx context.Context, text string) ([]models.SensitiveEntity, error)
	// IsAvailable 返回 NER 引擎是否已就绪（模型、动态库、Session 均初始化成功）。
	IsAvailable() bool
}

// RecordStore 定义记录存储端口（适配多种底层存储）。
type RecordStore interface {
	// Save 持久化键值记录。
	Save(ctx context.Context, key string, value []byte) error
	// Get 读取键值记录。
	Get(ctx context.Context, key string) ([]byte, error)
	// Delete 删除记录。
	Delete(ctx context.Context, key string) error
}

// SensitiveDetector 定义敏感信息检测端口。
type SensitiveDetector interface {
	// Detect 检测文本中的敏感实体，返回分级标记结果。
	Detect(ctx context.Context, text string) ([]models.SensitiveEntity, error)
}
