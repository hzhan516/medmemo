package models

// DesensitizationLevel 表示数据脱敏处理强度级别。
// 该级别仅标记用户偏好，实际脱敏流水线的分级执行策略由 adapter 层根据此配置动态调整。
type DesensitizationLevel string

const (
	// DesensitizationStandard 标准级：启用 L1 规则脱敏 + L2 NER 模型脱敏。
	DesensitizationStandard DesensitizationLevel = "standard"
)
