// Package models 定义跨层共享的基础数据模型。
package models

// TokenUsage 表示单次 LLM 调用的 Token 用量统计。
type TokenUsage struct {
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	FinishReason     string `json:"finish_reason"` // 如 "stop" / "length" / ""，用于检测输出是否被截断
}
