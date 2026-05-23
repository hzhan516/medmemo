// Package stream 流式响应统一处理层。
package stream

import (
	"time"

	"github.com/hzhan516/medmemo/pkg/models"
)

// Broker 将原始流式 callback 包装为统一 StreamChunk 序列。
type Broker struct {
	modelID    string
	providerID string
	startTime  time.Time
	started    bool
	emit       func(models.StreamChunk)
}

// NewBroker 创建流式处理 Broker。
func NewBroker(modelID, providerID string, emit func(models.StreamChunk)) *Broker {
	return &Broker{
		modelID:    modelID,
		providerID: providerID,
		emit:       emit,
	}
}

// Start 发送 start chunk，记录开始时间。
func (b *Broker) Start() {
	b.startTime = time.Now()
	b.started = true
	b.emit(models.StreamChunk{
		Type:    models.StreamChunkStart,
		Payload: "",
		Metadata: models.StreamChunkMetadata{
			Model:      b.modelID,
			ProviderID: b.providerID,
		},
	})
}

// Content 发送 content chunk，未调用 Start 时自动补发。
func (b *Broker) Content(payload string) {
	if !b.started {
		b.Start()
	}
	b.emit(models.StreamChunk{
		Type:    models.StreamChunkContent,
		Payload: payload,
	})
}

// Error 发送 error chunk。
func (b *Broker) Error(err string) {
	b.emit(models.StreamChunk{
		Type:    models.StreamChunkError,
		Payload: err,
	})
}

// Done 发送 done chunk，附加 latency 与 token 用量元数据。
func (b *Broker) Done(usage *models.TokenUsage) {
	latencyMs := time.Since(b.startTime).Milliseconds()
	meta := models.StreamChunkMetadata{
		LatencyMs: latencyMs,
	}
	if usage != nil {
		meta.PromptTokens = usage.PromptTokens
		meta.CompletionTokens = usage.CompletionTokens
	}
	b.emit(models.StreamChunk{
		Type:     models.StreamChunkDone,
		Payload:  "",
		Metadata: meta,
	})
}
