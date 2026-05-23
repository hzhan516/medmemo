// Package healthcheck Provider 健康检测引擎。
package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/pkg/models"
)

const (
	defaultCheckInterval = 60 * time.Second
	defaultCheckTimeout  = 2 * time.Second
)

// HealthEngine 周期性轮询所有已启用 Provider 的健康状态。
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
func NewHealthEngine(store port.ProviderStore) *HealthEngine {
	return &HealthEngine{
		store:        store,
		client:       &http.Client{Timeout: defaultCheckTimeout},
		checkTimeout: defaultCheckTimeout,
		interval:     defaultCheckInterval,
	}
}

// NewHealthEngineWithClient 使用自定义 HTTP 客户端创建引擎（主要用于测试）。
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

// Stop 优雅停止后台检测 goroutine。
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
		// 存储层错误静默处理，避免日志刷屏
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

// checkOne 对单个 Provider 执行连通性检测（2 秒超时）。
func (e *HealthEngine) checkOne(ctx context.Context, p *models.ProviderConfig) port.HealthResult {
	start := time.Now().UTC()

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

	authToken, err := p.ResolveAuthToken()
	if err != nil {
		return port.HealthResult{
			ProviderID: p.ID,
			Status:     port.HealthRed,
			LatencyMs:  time.Since(start).Milliseconds(),
			CheckedAt:  start,
			Error:      fmt.Sprintf("认证令牌解析失败: %v", err),
		}
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
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
	_ = resp.Body.Close()

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

// updateResult 写入缓存并在状态变更时触发回调。
func (e *HealthEngine) updateResult(result port.HealthResult) {
	oldValue, existed := e.results.Load(result.ProviderID)
	e.results.Store(result.ProviderID, result)

	if !existed {
		// 首次检测触发回调
		if e.onChange != nil {
			e.onChange(result)
		}
		return
	}

	old, ok := oldValue.(port.HealthResult)
	if !ok {
		// 类型断言失败，视为首次检测，触发回调
		if e.onChange != nil {
			e.onChange(result)
		}
		return
	}
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
	result, ok := val.(port.HealthResult)
	if !ok {
		return port.HealthResult{}, false
	}
	return result, true
}

// SetOnChange 设置状态变更回调。
func (e *HealthEngine) SetOnChange(cb func(port.HealthResult)) {
	e.onChange = cb
}
