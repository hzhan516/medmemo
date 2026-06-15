package models

// DesensitizationLevel 表示数据脱敏处理强度级别。
// 该级别仅标记用户偏好，实际脱敏流水线的分级执行策略由 adapter 层根据此配置动态调整。
type DesensitizationLevel string

const (
	// DesensitizationStandard 标准级：启用 L1 规则脱敏 + L2 NER 模型脱敏。
	DesensitizationStandard DesensitizationLevel = "standard"
	// DesensitizationStrict 严格级：启用 L1 + L2 + L3 关键词字典兜底三重脱敏。
	DesensitizationStrict DesensitizationLevel = "strict"
	// DesensitizationOff 关闭：不进行任何脱敏处理，数据以明文形式传输至云端。
	// 仅在用户明确知情并确认后可选，且每次会话顶部需追加风险提示。
	DesensitizationOff DesensitizationLevel = "off"
)
