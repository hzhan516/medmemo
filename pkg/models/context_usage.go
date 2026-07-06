// Package models 定义跨层共享的基础数据模型。
package models

// 上下文用量相关常量。
const (
	// MinContextLength 是模型允许的最小上下文长度。
	MinContextLength = 256
	// MaxContextLengthCap 是上下文长度的全局上限，防止配置异常导致过大内存占用。
	MaxContextLengthCap = 2000000
	// DefaultMaxContextLen 是默认最大上下文长度。
	DefaultMaxContextLen = 8192
	// WarningThreshold 是上下文用量警告阈值（75%）。
	WarningThreshold = 0.75
	// AutoCompressionThreshold 是自动压缩触发阈值（90%）。
	AutoCompressionThreshold = 0.90
)

// ContextUsage 表示会话上下文用量状态，供前端进度条展示。
type ContextUsage struct {
	UsedTokens  int     `json:"usedTokens"`
	MaxTokens   int     `json:"maxTokens"`
	Ratio       float64 `json:"ratio"`
	Approximate bool    `json:"approximate"`
}
