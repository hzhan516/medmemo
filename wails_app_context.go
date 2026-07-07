package main

import (
	"context"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/hzhan516/medmemo/internal/application/usecase"
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
}

// CompressSession 触发当前会话的上下文压缩，并在完成后通知前端刷新用量。
func (a *WailsApp) CompressSession(req CompressSessionRequest) error {
	if a.compressionService == nil {
		return fmt.Errorf("compression service not initialized")
	}

	ctx, cancel := context.WithTimeout(a.ctx, defaultCompressSessionTimeout)
	defer cancel()

	cfg := usecase.CompressionConfig{
		Strategy:    usecase.StrategySummarizeAndReplace, // 默认使用确定性摘要替换策略
		AnchorCount: 1,
		RecentCount: 6,
	}

	_, err := a.compressionService.Compress(ctx, models.ConversationID(req.ConversationID), req.ProviderID, req.ModelID, cfg)
	if err != nil {
		return fmt.Errorf("failed to compress session: %w", err)
	}

	runtime.EventsEmit(a.ctx, "context:usage_refresh", map[string]any{
		"conversation_id": req.ConversationID,
	})
	return nil
}
