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

	// ErrFactNotFound 表示事实记录不存在。
	ErrFactNotFound = errors.New("fact not found")
	// ErrEmbeddingNotFound 表示嵌入向量记录不存在。
	ErrEmbeddingNotFound = errors.New("embedding not found")
	// ErrInvalidFact 表示事实三元组不完整或无效。
	ErrInvalidFact = errors.New("invalid fact: subject, predicate, object and source messages are required")
	// ErrInvalidConfidence 表示置信度超出有效范围 [0,1]。
	ErrInvalidConfidence = errors.New("invalid confidence: must be in range [0,1]")
	// ErrInvalidVector 表示向量数据无效（维度错误或字节长度不匹配）。
	ErrInvalidVector = errors.New("invalid vector data")
)
