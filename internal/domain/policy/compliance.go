// Package policy 定义合规策略与敏感分级策略的抽象。
package policy

// RiskLevel 表示合规风险等级。
type RiskLevel int

const (
	L1Blocked RiskLevel = iota + 1 // L1-阻断级：诊断/处方/治疗
	L2Warning                      // L2-警告级：暗示性诊断/药物推荐
	L3Notice                       // L3-提示级：严重疾病科普
	L4Normal                       // L4-正常级：一般健康科普/生活方式
)

// String 返回风险等级的可读名称。
func (r RiskLevel) String() string {
	switch r {
	case L1Blocked:
		return "BLOCKED"
	case L2Warning:
		return "WARNING"
	case L3Notice:
		return "NOTICE"
	case L4Normal:
		return "NORMAL"
	default:
		return "UNKNOWN"
	}
}
