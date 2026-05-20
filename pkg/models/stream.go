// Package models 定义跨层共享的基础数据模型。
package models

// StreamChunkType 表示流式输出块的类型。
type StreamChunkType string

const (
	// StreamChunkStart 表示流式输出开始。
	StreamChunkStart StreamChunkType = "start"
	// StreamChunkContent 表示正常内容输出。
	StreamChunkContent StreamChunkType = "content"
	// StreamChunkDone 表示流式输出正常结束。
	StreamChunkDone StreamChunkType = "done"
	// StreamChunkError 表示流式过程中发生的错误。
	StreamChunkError StreamChunkType = "error"
)

// StreamChunkMetadata 表示流式 chunk 的元数据。
type StreamChunkMetadata struct {
	Model      string `json:"model,omitempty"`
	ProviderID string `json:"provider_id,omitempty"`
	LatencyMs  int64  `json:"latency_ms,omitempty"`
	TokenCount int    `json:"token_count,omitempty"`
}

// StreamChunk 表示 SSE 流式响应的统一输出块。
// 供 application 层流式统一处理层包装后通过 Wails Events 推送给前端。
type StreamChunk struct {
	Type     StreamChunkType     `json:"type"`
	Payload  string              `json:"payload"`
	Metadata StreamChunkMetadata `json:"metadata,omitempty"`
}
