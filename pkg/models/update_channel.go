// Package models 定义跨层共享的纯数据类型。
// 更新通道类型与常量，供 domain、infrastructure、application、adapters 各层使用。
package models

// UpdateChannel 定义更新通道类型。
type UpdateChannel string

const (
	ChannelStable UpdateChannel = "stable" // 稳定版通道，仅包含正式发布版本
	ChannelBeta   UpdateChannel = "beta"   // 测试版通道，包含预发布版本
)
