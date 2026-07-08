package models

import "strings"

// DesensitizationLevel 表示数据脱敏处理强度级别。
// 该级别仅标记用户偏好，实际脱敏流水线的分级执行策略由 usecase 层根据此配置动态调整。
type DesensitizationLevel string

const (
	// DesensitizationStandard 标准级：启用 L1 规则脱敏 + L2 NER 模型脱敏。
	DesensitizationStandard DesensitizationLevel = "standard"

	// DesensitizationStrict 严格级：在 standard 基础上追加兜底策略（待产品/合规确认后细化）。
	DesensitizationStrict DesensitizationLevel = "strict"

	// DesensitizationOff 关闭级：跳过出网脱敏。仅应在用户明确知情并承担风险时使用。
	DesensitizationOff DesensitizationLevel = "off"
)

var validDesensitizationLevels = map[DesensitizationLevel]struct{}{
	DesensitizationStandard: {},
	DesensitizationStrict:   {},
	DesensitizationOff:      {},
}

// IsValid 判断当前级别是否为合法值（区分大小写）。
func (d DesensitizationLevel) IsValid() bool {
	_, ok := validDesensitizationLevels[d]
	return ok
}

// Normalize 将空值或非法值归一化为 standard，合法值保持原样。
// 统一使用小写形式，避免配置文件大小写不一致导致的行为差异。
func (d DesensitizationLevel) Normalize() DesensitizationLevel {
	normalized := DesensitizationLevel(strings.ToLower(string(d)))
	if normalized.IsValid() {
		return normalized
	}
	return DesensitizationStandard
}

// NormalizeDesensitizationLevel 将字符串形式的级别归一化为合法枚举值。
// 空值或非法值回退到 standard，大小写不敏感。
func NormalizeDesensitizationLevel(level string) DesensitizationLevel {
	return DesensitizationLevel(level).Normalize()
}
