package entity

import "errors"

// 领域层预定义错误，供各层映射与比较。
// 采用哨兵错误模式，便于 errors.Is 判断。
var (
	ErrNotFound       = errors.New("record not found")
	ErrInvalidConfig  = errors.New("invalid configuration")
	ErrDuplicateEntry = errors.New("duplicate entry")
	ErrUnauthorized   = errors.New("unauthorized access")

	// ErrComplianceBlocked 表示内容因合规规则被阻断。
	ErrComplianceBlocked = errors.New("content blocked by compliance policy")

	// ErrSensitiveDataLeak 表示检测到敏感数据泄露风险。
	ErrSensitiveDataLeak = errors.New("potential sensitive data leak detected")
)
