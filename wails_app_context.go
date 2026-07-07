package main

import (
	"context"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/hzhan516/medmemo/internal/infrastructure/config"
	"github.com/hzhan516/medmemo/pkg/models"
)

// EstimateContextUsageRequest 前端估算上下文用量的请求。
type EstimateContextUsageRequest struct {
	ConversationID  string           `json:"conversationId"`
	Messages        []models.Message `json:"messages"`
	ProviderID      string           `json:"providerId"`
	ModelID         string           `json:"modelId"`
	AssembledPrompt []models.Message `json:"assembledPrompt,omitempty"`
}

// ContextUsageResponse 上下文用量估算结果。
type ContextUsageResponse struct {
	UsedTokens  int     `json:"usedTokens"`
	MaxTokens   int     `json:"maxTokens"`
	Ratio       float64 `json:"ratio"`
	Approximate bool    `json:"approximate"`
}

const (
	defaultResolveMaxContextLengthTimeout = 5 * time.Second
	defaultEstimateContextUsageTimeout    = 10 * time.Second
	defaultCompressSessionTimeout         = 30 * time.Second
)

// ResolveMaxContextLength 解析指定 provider 与 model 的最大上下文长度。
func (a *WailsApp) ResolveMaxContextLength(providerID, modelID string) (int, error) {
	if a.contextLengthResolver == nil {
		return 0, fmt.Errorf("context length resolver not initialized")
	}

	ctx, cancel := context.WithTimeout(a.ctx, defaultResolveMaxContextLengthTimeout)
	defer cancel()

	max := a.contextLengthResolver.Resolve(ctx, providerID, modelID)
	return max, nil
}

// EstimateContextUsage 估算当前会话上下文 token 用量及占比。
func (a *WailsApp) EstimateContextUsage(req EstimateContextUsageRequest) (*ContextUsageResponse, error) {
	if a.contextEstimator == nil {
		return nil, fmt.Errorf("context estimator not initialized")
	}

	ctx, cancel := context.WithTimeout(a.ctx, defaultEstimateContextUsageTimeout)
	defer cancel()

	assembled := a.chatOrchestrator.AssemblePromptForEstimate(ctx, usecase.ChatRequest{
		ConversationID: models.ConversationID(req.ConversationID),
		Messages:       req.Messages,
		Model:          models.ProviderType(req.ModelID),
		ProviderID:     req.ProviderID,
	})

	result, err := a.contextEstimator.Estimate(ctx, usecase.EstimatorInput{
		Messages:        req.Messages,
		ProviderID:      req.ProviderID,
		ModelID:         req.ModelID,
		AssembledPrompt: assembled,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to estimate context usage: %w", err)
	}

	return &ContextUsageResponse{
		UsedTokens:  result.UsedTokens,
		MaxTokens:   result.MaxTokens,
		Ratio:       result.Ratio,
		Approximate: result.Approximate,
	}, nil
}

// CompressSessionRequest 前端触发会话压缩的请求。
type CompressSessionRequest struct {
	ConversationID string `json:"conversationId"`
	ProviderID     string `json:"providerId"`
	ModelID        string `json:"modelId"`
	Strategy       string `json:"strategy,omitempty"`
	AnchorCount    int    `json:"anchorCount,omitempty"`
	RecentCount    int    `json:"recentCount,omitempty"`
}

// resolveStrategy 将字符串策略解析为合法策略，非法值回退默认。
func resolveStrategy(s string) usecase.CompressionStrategyKind {
	switch usecase.CompressionStrategyKind(s) {
	case usecase.StrategyDropEarliestN,
		usecase.StrategySummarizeAndReplace,
		usecase.StrategyLLMSelfSummarize:
		return usecase.CompressionStrategyKind(s)
	default:
		return usecase.StrategySummarizeAndReplace
	}
}

// buildCompressionConfigFrom 依据应用配置返回压缩配置与用于压缩的 provider/model。
func buildCompressionConfigFrom(s models.CompressionSettings, activeProviderID, activeModelID string) (usecase.CompressionConfig, string, string) {
	anchor, recent := 1, 6
	if s.AnchorCount > 0 {
		anchor = s.AnchorCount
	}
	if s.RecentCount > 0 {
		recent = s.RecentCount
	}

	if s.UseModel {
		providerID, modelID := s.ProviderID, s.ModelID
		if providerID == "" {
			providerID = activeProviderID
		}
		if modelID == "" {
			modelID = activeModelID
		}
		return usecase.CompressionConfig{
			Strategy:    usecase.StrategyLLMSelfSummarize,
			AnchorCount: anchor,
			RecentCount: recent,
		}, providerID, modelID
	}
	return usecase.CompressionConfig{
		Strategy:    usecase.StrategySummarizeAndReplace,
		AnchorCount: anchor,
		RecentCount: recent,
	}, activeProviderID, activeModelID
}

// buildCompressionConfig 依据应用配置返回压缩配置与用于压缩的 provider/model。
func (a *WailsApp) buildCompressionConfig(activeProviderID, activeModelID string) (usecase.CompressionConfig, string, string) {
	return buildCompressionConfigFrom(a.config.CompressionSettings, activeProviderID, activeModelID)
}

// CompressSession 触发当前会话的上下文压缩，并在完成后通知前端刷新用量。
func (a *WailsApp) CompressSession(req CompressSessionRequest) error {
	if a.compressionService == nil {
		return fmt.Errorf("compression service not initialized")
	}

	ctx, cancel := context.WithTimeout(a.ctx, defaultCompressSessionTimeout)
	defer cancel()

	cfg, providerID, modelID := a.buildCompressionConfig(req.ProviderID, req.ModelID)
	if req.Strategy != "" {
		cfg.Strategy = resolveStrategy(req.Strategy)
	}
	if req.AnchorCount > 0 {
		cfg.AnchorCount = req.AnchorCount
	}
	if req.RecentCount > 0 {
		cfg.RecentCount = req.RecentCount
	}

	_, err := a.compressionService.Compress(ctx, models.ConversationID(req.ConversationID), providerID, modelID, cfg)
	if err != nil {
		return fmt.Errorf("failed to compress session: %w", err)
	}

	runtime.EventsEmit(a.ctx, "context:usage_refresh", map[string]any{
		"conversation_id": req.ConversationID,
	})
	return nil
}

// GetCompressionSettings 返回当前会话压缩设置。
func (a *WailsApp) GetCompressionSettings() models.CompressionSettings {
	return a.config.CompressionSettings
}

// SetCompressionSettings 保存会话压缩设置。
func (a *WailsApp) SetCompressionSettings(s models.CompressionSettings) error {
	a.config.CompressionSettings = s
	if err := config.SaveCompressionSettings(s); err != nil {
		return fmt.Errorf("failed to save compression settings: %w", err)
	}
	return nil
}

// TestCompressionModel 测试选定的压缩模型是否可用。
func (a *WailsApp) TestCompressionModel(providerID, modelID string) (bool, error) {
	_ = modelID
	if a.compressionService == nil {
		return false, fmt.Errorf("compression service not initialized")
	}

	ctx, cancel := context.WithTimeout(a.ctx, defaultResolveMaxContextLengthTimeout)
	defer cancel()

	available, err := a.compressionService.TestModelAvailability(ctx, providerID)
	if err != nil {
		return false, fmt.Errorf("failed to check availability: %w", err)
	}
	return available, nil
}
