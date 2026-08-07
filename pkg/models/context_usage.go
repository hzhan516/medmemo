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
	// WarningThreshold 是上下文用量警告阈值（75%）。WARN: 与前端 web/src/utils/contextUsage.ts 保持一致。
	WarningThreshold = 0.75
	// AutoCompressionThreshold 是自动压缩触发阈值（90%）。WARN: 与前端 web/src/utils/contextUsage.ts 保持一致。
	AutoCompressionThreshold = 0.90
)

// 防止 WarningThreshold 被静态分析标记为未使用：该常量与前端共享，属于公共契约。
var _ = WarningThreshold

// CompressionSettings 会话压缩相关的用户配置。
type CompressionSettings struct {
	UseModel    bool   `json:"useModel"`
	ProviderID  string `json:"providerId,omitempty"`
	ModelID     string `json:"modelId,omitempty"`
	AnchorCount int    `json:"anchorCount,omitempty"`
	RecentCount int    `json:"recentCount,omitempty"`
}

// ContextUsage 表示会话上下文用量状态，供前端进度条展示。
type ContextUsage struct {
	UsedTokens  int     `json:"usedTokens"`
	MaxTokens   int     `json:"maxTokens"`
	Ratio       float64 `json:"ratio"`
	Approximate bool    `json:"approximate"`
}
