package usecase

import (
	"context"

	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/pkg/models"
)

const perMessageOverhead = 4

// EstimatorInput 是上下文用量估算器的输入。
type EstimatorInput struct {
	Messages        []models.Message
	ProviderID      string
	ModelID         string
	AssembledPrompt []models.Message
}

// EstimateResult 是上下文用量估算结果。
type EstimateResult struct {
	UsedTokens  int
	MaxTokens   int
	Ratio       float64
	Approximate bool
}

// ContextEstimator 估算当前会话占用的上下文 token 比例。
type ContextEstimator struct {
	counter  port.TokenCounter
	resolver *ContextLengthResolver
}

// NewContextEstimator 创建一个新的上下文用量估算器。
func NewContextEstimator(counter port.TokenCounter, resolver *ContextLengthResolver) *ContextEstimator {
	return &ContextEstimator{
		counter:  counter,
		resolver: resolver,
	}
}

// Estimate 估算给定输入的上下文 token 用量及占最大上下文长度的比例。
func (e *ContextEstimator) Estimate(ctx context.Context, in EstimatorInput) (EstimateResult, error) {
	messages := in.AssembledPrompt
	if messages == nil {
		messages = in.Messages
	}

	used := 0
	approximate := false
	for _, m := range messages {
		n, ok := e.counter.Count(ctx, in.ModelID, m.Content)
		if !ok {
			approximate = true
		}
		used += n + perMessageOverhead
	}

	maxTokens := e.resolver.Resolve(ctx, in.ProviderID, in.ModelID)

	return EstimateResult{
		UsedTokens:  used,
		MaxTokens:   maxTokens,
		Ratio:       boundedRatio(used, maxTokens),
		Approximate: approximate,
	}, nil
}

// boundedRatio 计算用量比例，并将其限制在 [0, 1] 范围内。
func boundedRatio(used, max int) float64 {
	if max <= 0 {
		return 0
	}
	r := float64(used) / float64(max)
	if r < 0 {
		return 0
	}
	if r > 1 {
		return 1
	}
	return r
}
