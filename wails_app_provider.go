package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hzhan516/medmemo/pkg/models"
)

// ModelInfo 模型信息。
type ModelInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

// GetModels 获取可用模型列表。
func (a *WailsApp) GetModels() ([]ModelInfo, error) {
	return []ModelInfo{
		{ID: "kimi-lite", Name: "Kimi Lite", Provider: "kimi"},
		{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Provider: "openai"},
		{ID: "qwen-turbo", Name: "通义千问 Turbo", Provider: "qwen"},
		{ID: "llama3.1-8b", Name: "Llama 3.1 8B (本地)", Provider: "ollama"},
	}, nil
}

// CreateProvider 创建新的 Provider 配置。
func (a *WailsApp) CreateProvider(config models.ProviderConfig) error {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	if err := a.providerStore.Create(ctx, &config); err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	return nil
}

// UpdateProvider 更新已有 Provider 配置。
func (a *WailsApp) UpdateProvider(config models.ProviderConfig) error {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	if err := a.providerStore.Update(ctx, &config); err != nil {
		return fmt.Errorf("failed to update provider: %w", err)
	}
	return nil
}

// DeleteProvider 删除指定 Provider 配置。
func (a *WailsApp) DeleteProvider(id string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	if err := a.providerStore.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}
	return nil
}

// ListProviders 获取全部 Provider 配置列表。
func (a *WailsApp) ListProviders() ([]models.ProviderConfig, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	list, err := a.providerStore.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}

	result := make([]models.ProviderConfig, len(list))
	for i, p := range list {
		result[i] = *p
	}
	return result, nil
}

// HealthResultResponse 健康检测结果响应（供前端序列化）。
type HealthResultResponse struct {
	ProviderID string `json:"provider_id"`
	Status     string `json:"status"`
	LatencyMs  int64  `json:"latency_ms"`
	CheckedAt  string `json:"checked_at"`
	Error      string `json:"error,omitempty"`
}

// CheckProviderHealth 对指定 Provider 执行一次即时健康检测。
func (a *WailsApp) CheckProviderHealth(providerID string) (*HealthResultResponse, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.healthChecker == nil {
		return nil, fmt.Errorf("health checker not initialized")
	}

	result, err := a.healthChecker.CheckNow(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check provider health: %w", err)
	}

	return &HealthResultResponse{
		ProviderID: result.ProviderID,
		Status:     string(result.Status),
		LatencyMs:  result.LatencyMs,
		CheckedAt:  result.CheckedAt.Format(time.RFC3339),
		Error:      result.Error,
	}, nil
}

// GetProviderHealthStatus 查询指定 Provider 的缓存健康状态（无需网络请求）。
func (a *WailsApp) GetProviderHealthStatus(providerID string) (*HealthResultResponse, error) {
	if a.healthChecker == nil {
		return nil, fmt.Errorf("health checker not initialized")
	}

	result, ok := a.healthChecker.GetStatus(providerID)
	if !ok {
		return nil, fmt.Errorf("provider %s health status not available", providerID)
	}

	return &HealthResultResponse{
		ProviderID: result.ProviderID,
		Status:     string(result.Status),
		LatencyMs:  result.LatencyMs,
		CheckedAt:  result.CheckedAt.Format(time.RFC3339),
		Error:      result.Error,
	}, nil
}
