// Package stream 实现流式响应统一处理层。
// 将底层适配器输出的原始流式内容包装为结构化的 StreamChunk 序列，
// 负责元数据附加、latency 计算与错误处理。
package stream

import (
	"time"

	"github.com/medmemo/medmemo/pkg/models"
)

// Broker 将原始流式 callback 输出包装为统一 StreamChunk 序列。
type Broker struct {
	modelID    string
	providerID string
	startTime  time.Time
	started    bool
	emit       func(models.StreamChunk)
}

// NewBroker 创建流式处理 Broker。
// emit 回调负责将包装后的 StreamChunk 推送到消费者（如 Wails Events）。
func NewBroker(modelID, providerID string, emit func(models.StreamChunk)) *Broker {
	return &Broker{
		modelID:    modelID,
		providerID: providerID,
		emit:       emit,
	}
}

// Start 发送 start chunk，记录开始时间。
// Metadata 附加 model 与 providerID。
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

// Content 发送 content chunk。
// 若 Start 未被显式调用，会自动补发 start chunk，确保序列完整性。
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

// Done 发送 done chunk，附加从开始到结束的 latency 元数据（毫秒）。
func (b *Broker) Done() {
	latencyMs := time.Since(b.startTime).Milliseconds()
	b.emit(models.StreamChunk{
		Type:    models.StreamChunkDone,
		Payload: "",
		Metadata: models.StreamChunkMetadata{
			LatencyMs: latencyMs,
		},
	})
}
