package port

import (
	"context"
	"time"
)

// HealthStatus 表示 Provider 的健康状态。
type HealthStatus string

// noinspection GoUnusedConst

const (
	// HealthGreen 表示连通且延迟低（<2s）。
	HealthGreen HealthStatus = "green"
	// HealthYellow 表示连通但延迟较高（2s~5s）。
	HealthYellow HealthStatus = "yellow"
	// HealthRed 表示不连通或延迟极高（>5s）。
	HealthRed HealthStatus = "red"
	// HealthUnknown 表示尚未检测。
	HealthUnknown HealthStatus = "unknown"
)

// HealthResult 表示单个 Provider 的健康检测结果。
type HealthResult struct {
	ProviderID string       `json:"provider_id"`
	Status     HealthStatus `json:"status"`
	LatencyMs  int64        `json:"latency_ms"`
	CheckedAt  time.Time    `json:"checked_at"`
	Error      string       `json:"error,omitempty"`
}

// HealthChecker 定义健康检测引擎接口。
// 由 application/healthcheck 实现，供 WailsApp 消费。
type HealthChecker interface {
	// Start 启动后台周期性检测 goroutine。
	Start(ctx context.Context)
	// Stop 优雅停止后台检测 goroutine。
	Stop()
	// CheckNow 对指定 Provider 执行一次即时检测。
	CheckNow(ctx context.Context, providerID string) (HealthResult, error)
	// GetStatus 从缓存中获取指定 Provider 最近一次检测结果。
	GetStatus(providerID string) (HealthResult, bool)
	// SetOnChange 设置状态变更回调，仅在状态发生变更时触发。
	SetOnChange(cb func(HealthResult))
}
