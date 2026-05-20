// Package healthcheck 实现 Provider 健康检测引擎。
// 通过周期性调用 /v1/models 端点验证 Provider 连通性，
// 并将状态变更通过回调通知上层（由 WailsApp 转发为 Wails Events）。
package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/pkg/models"
)

// defaultCheckInterval 默认检测周期。
const defaultCheckInterval = 60 * time.Second

// defaultCheckTimeout 单次检测请求超时。
const defaultCheckTimeout = 2 * time.Second

// HealthEngine 实现 port.HealthChecker 接口。
// 使用独立 goroutine 周期性轮询所有已启用的 Provider。
type HealthEngine struct {
	store        port.ProviderStore
	client       *http.Client
	checkTimeout time.Duration
	interval     time.Duration

	results  sync.Map // key: providerID, value: port.HealthResult
	mu       sync.RWMutex
	running  bool
	stopCh   chan struct{}
	wg       sync.WaitGroup
	onChange func(port.HealthResult)
}

// NewHealthEngine 创建健康检测引擎。
// 使用默认 HTTP 客户端和 60 秒检测周期。
func NewHealthEngine(store port.ProviderStore) *HealthEngine {
	return &HealthEngine{
		store:        store,
		client:       &http.Client{Timeout: defaultCheckTimeout},
		checkTimeout: defaultCheckTimeout,
		interval:     defaultCheckInterval,
	}
}

// NewHealthEngineWithClient 使用自定义 HTTP 客户端创建引擎。
// 主要用于测试注入 Mock HTTP Transport。
func NewHealthEngineWithClient(store port.ProviderStore, client *http.Client) *HealthEngine {
	return &HealthEngine{
		store:        store,
		client:       client,
		checkTimeout: defaultCheckTimeout,
		interval:     defaultCheckInterval,
	}
}

// Start 启动后台周期性检测 goroutine。
func (e *HealthEngine) Start(ctx context.Context) {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	e.running = true
	e.stopCh = make(chan struct{})
	e.mu.Unlock()

	e.wg.Add(1)
	go e.loop(ctx)
}

// Stop 优雅停止后台检测 goroutine，等待正在执行的检测完成。
func (e *HealthEngine) Stop() {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return
	}
	e.running = false
	close(e.stopCh)
	e.mu.Unlock()

	e.wg.Wait()
}

// loop 后台轮询主循环。
func (e *HealthEngine) loop(ctx context.Context) {
	defer e.wg.Done()

	// 首次立即检测
	e.checkAll(ctx)

	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.checkAll(ctx)
		case <-e.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkAll 获取全部启用的 Provider 并逐一检测。
func (e *HealthEngine) checkAll(ctx context.Context) {
	list, err := e.store.List(ctx)
	if err != nil {
		// 存储层错误静默处理，避免日志刷屏；下次轮询重试
		return
	}

	for _, p := range list {
		if !p.Enabled {
			continue
		}
		result := e.checkOne(ctx, p)
		e.updateResult(result)
	}
}

// checkOne 对单个 Provider 执行连通性检测。
// 发送 GET {APIHost}/v1/models 请求，2 秒超时。
func (e *HealthEngine) checkOne(ctx context.Context, p *models.ProviderConfig) port.HealthResult {
	start := time.Now()

	checkCtx, cancel := context.WithTimeout(ctx, e.checkTimeout)
	defer cancel()

	url := p.APIHost + "/v1/models"
	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, url, nil)
	if err != nil {
		return port.HealthResult{
			ProviderID: p.ID,
			Status:     port.HealthRed,
			LatencyMs:  time.Since(start).Milliseconds(),
			CheckedAt:  start,
			Error:      fmt.Sprintf("构造请求失败: %v", err),
		}
	}

	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	resp, err := e.client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return port.HealthResult{
			ProviderID: p.ID,
			Status:     port.HealthRed,
			LatencyMs:  latency.Milliseconds(),
			CheckedAt:  start,
			Error:      fmt.Sprintf("请求失败: %v", err),
		}
	}
	//nolint:errcheck // Body 不需要读取内容，直接关闭即可
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return port.HealthResult{
			ProviderID: p.ID,
			Status:     port.HealthRed,
			LatencyMs:  latency.Milliseconds(),
			CheckedAt:  start,
			Error:      fmt.Sprintf("HTTP %d", resp.StatusCode),
		}
	}

	status := classifyLatency(latency)
	return port.HealthResult{
		ProviderID: p.ID,
		Status:     status,
		LatencyMs:  latency.Milliseconds(),
		CheckedAt:  start,
	}
}

// classifyLatency 根据响应延迟判定健康等级。
func classifyLatency(d time.Duration) port.HealthStatus {
	ms := d.Milliseconds()
	switch {
	case ms < 2000:
		return port.HealthGreen
	case ms <= 5000:
		return port.HealthYellow
	default:
		return port.HealthRed
	}
}

// updateResult 将检测结果写入缓存，并在状态变更时触发回调。
func (e *HealthEngine) updateResult(result port.HealthResult) {
	oldValue, existed := e.results.Load(result.ProviderID)
	e.results.Store(result.ProviderID, result)

	if !existed {
		// 首次检测，从 Unknown 变为具体状态，触发回调
		if e.onChange != nil {
			e.onChange(result)
		}
		return
	}

	old := oldValue.(port.HealthResult)
	if old.Status != result.Status && e.onChange != nil {
		e.onChange(result)
	}
}

// CheckNow 对指定 Provider 执行一次即时检测。
func (e *HealthEngine) CheckNow(ctx context.Context, providerID string) (port.HealthResult, error) {
	p, err := e.store.Get(ctx, providerID)
	if err != nil {
		return port.HealthResult{}, fmt.Errorf("failed to get provider %s: %w", providerID, err)
	}
	if !p.Enabled {
		return port.HealthResult{}, fmt.Errorf("provider %s is disabled", providerID)
	}

	result := e.checkOne(ctx, p)
	e.updateResult(result)
	return result, nil
}

// GetStatus 从缓存中获取指定 Provider 最近一次检测结果。
func (e *HealthEngine) GetStatus(providerID string) (port.HealthResult, bool) {
	val, ok := e.results.Load(providerID)
	if !ok {
		return port.HealthResult{}, false
	}
	return val.(port.HealthResult), true
}

// SetOnChange 设置状态变更回调。
func (e *HealthEngine) SetOnChange(cb func(port.HealthResult)) {
	e.onChange = cb
}
