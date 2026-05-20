// Package stream 的单元测试。
package stream

import (
	"testing"
	"time"

	"github.com/medmemo/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
)

// TestBroker_Flow 验证完整的 Start → Content → Done 流程。
func TestBroker_Flow(t *testing.T) {
	var chunks []models.StreamChunk
	b := NewBroker("kimi-lite", "provider-1", func(c models.StreamChunk) {
		chunks = append(chunks, c)
	})

	b.Start()
	b.Content("你好")
	b.Content("世界")
	b.Done()

	assert.Len(t, chunks, 4)

	// start
	assert.Equal(t, models.StreamChunkStart, chunks[0].Type)
	assert.Equal(t, "", chunks[0].Payload)
	assert.Equal(t, "kimi-lite", chunks[0].Metadata.Model)
	assert.Equal(t, "provider-1", chunks[0].Metadata.ProviderID)

	// content 1
	assert.Equal(t, models.StreamChunkContent, chunks[1].Type)
	assert.Equal(t, "你好", chunks[1].Payload)

	// content 2
	assert.Equal(t, models.StreamChunkContent, chunks[2].Type)
	assert.Equal(t, "世界", chunks[2].Payload)

	// done
	assert.Equal(t, models.StreamChunkDone, chunks[3].Type)
	assert.Equal(t, "", chunks[3].Payload)
	assert.GreaterOrEqual(t, chunks[3].Metadata.LatencyMs, int64(0))
}

// TestBroker_AutoStart 验证未显式调用 Start 时，Content 自动补发 start。
func TestBroker_AutoStart(t *testing.T) {
	var chunks []models.StreamChunk
	b := NewBroker("gpt-4o", "", func(c models.StreamChunk) {
		chunks = append(chunks, c)
	})

	b.Content("自动开始")

	assert.Len(t, chunks, 2)
	assert.Equal(t, models.StreamChunkStart, chunks[0].Type)
	assert.Equal(t, models.StreamChunkContent, chunks[1].Type)
	assert.Equal(t, "自动开始", chunks[1].Payload)
}

// TestBroker_Error 验证 Error 输出 error chunk。
func TestBroker_Error(t *testing.T) {
	var chunks []models.StreamChunk
	b := NewBroker("", "", func(c models.StreamChunk) {
		chunks = append(chunks, c)
	})

	b.Error("连接超时")

	assert.Len(t, chunks, 1)
	assert.Equal(t, models.StreamChunkError, chunks[0].Type)
	assert.Equal(t, "连接超时", chunks[0].Payload)
}

// TestBroker_Latency 验证 Done 的 latencyMs 为合理正值。
func TestBroker_Latency(t *testing.T) {
	var chunks []models.StreamChunk
	b := NewBroker("", "", func(c models.StreamChunk) {
		chunks = append(chunks, c)
	})

	b.Start()
	time.Sleep(10 * time.Millisecond)
	b.Done()

	assert.Len(t, chunks, 2)
	assert.Equal(t, models.StreamChunkDone, chunks[1].Type)
	// 允许一定调度误差，latency 应 >= 5ms（睡了 10ms）
	assert.GreaterOrEqual(t, chunks[1].Metadata.LatencyMs, int64(5))
}

// TestBroker_Metadata 验证 start 与 done 的 Metadata 附加规则。
func TestBroker_Metadata(t *testing.T) {
	var chunks []models.StreamChunk
	b := NewBroker("qwen-turbo", "p-001", func(c models.StreamChunk) {
		chunks = append(chunks, c)
	})

	b.Start()
	b.Done()

	assert.Len(t, chunks, 2)

	// start 携带 model + providerID
	assert.Equal(t, "qwen-turbo", chunks[0].Metadata.Model)
	assert.Equal(t, "p-001", chunks[0].Metadata.ProviderID)
	assert.Zero(t, chunks[0].Metadata.LatencyMs)

	// done 携带 latencyMs，不携带 model/providerID
	assert.GreaterOrEqual(t, chunks[1].Metadata.LatencyMs, int64(0))
	assert.Empty(t, chunks[1].Metadata.Model)
	assert.Empty(t, chunks[1].Metadata.ProviderID)
}

// TestBroker_ErrorAfterContent 验证 content 之后出现 error 的场景。
func TestBroker_ErrorAfterContent(t *testing.T) {
	var chunks []models.StreamChunk
	b := NewBroker("", "", func(c models.StreamChunk) {
		chunks = append(chunks, c)
	})

	b.Start()
	b.Content("部分")
	b.Error("流已中断")

	assert.Len(t, chunks, 3)
	assert.Equal(t, models.StreamChunkContent, chunks[1].Type)
	assert.Equal(t, models.StreamChunkError, chunks[2].Type)
}
