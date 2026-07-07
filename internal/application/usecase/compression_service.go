package usecase

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
)

// CompressionStrategyKind 定义会话压缩策略类型。
type CompressionStrategyKind string

const (
	// StrategySummarizeAndReplace 使用本地确定性摘要替换中间消息块。
	StrategySummarizeAndReplace CompressionStrategyKind = "summarize_and_replace"
	// StrategyDropEarliestN 删除最早 N 条消息，保留最近消息。
	StrategyDropEarliestN CompressionStrategyKind = "drop_earliest_n"
	// StrategyLLMSelfSummarize 调用 LLM 对中间消息块生成摘要。
	StrategyLLMSelfSummarize CompressionStrategyKind = "llm_self_summarization"
)

// CompressionConfig 是会话压缩配置。
type CompressionConfig struct {
	Strategy    CompressionStrategyKind
	AnchorCount int
	RecentCount int
	DropN       int
}

// CompressionResult 是会话压缩结果。
type CompressionResult struct {
	Messages         []models.Message
	UsedBefore       int
	UsedAfter        int
	FallbackOccurred bool
	Strategy         CompressionStrategyKind
}

// CompressionService 提供会话上下文压缩能力。
type CompressionService struct {
	estimator  *ContextEstimator
	llmFactory port.LLMClientFactory
	providers  port.ProviderStore
	msgRepo    port.MessageRepository
}

// NewCompressionService 创建一个新的压缩服务实例。
func NewCompressionService(
	estimator *ContextEstimator,
	llmFactory port.LLMClientFactory,
	providers port.ProviderStore,
	msgRepo port.MessageRepository,
) *CompressionService {
	return &CompressionService{
		estimator:  estimator,
		llmFactory: llmFactory,
		providers:  providers,
		msgRepo:    msgRepo,
	}
}

var messageIDCounter atomic.Int64

// generateMessageID 生成消息 ID，使用纳秒时间戳叠加原子计数器后缀，降低并发冲突风险。
func generateMessageID() string {
	return fmt.Sprintf("msg_%d_%d", time.Now().UnixNano(), messageIDCounter.Add(1))
}

// reverseEntities 原地反转实体消息切片。
func reverseEntities(entities []*entity.Message) {
	for i, j := 0, len(entities)-1; i < j; i, j = i+1, j-1 {
		entities[i], entities[j] = entities[j], entities[i]
	}
}

// CompressMessages 对给定历史消息做纯内存压缩，不触碰 DB。
func (s *CompressionService) CompressMessages(ctx context.Context, history []models.Message, providerID, modelID string, cfg CompressionConfig) (CompressionResult, error) {
	beforeResult, err := s.estimator.Estimate(ctx, EstimatorInput{
		Messages:   history,
		ProviderID: providerID,
		ModelID:    modelID,
	})
	if err != nil {
		return CompressionResult{}, fmt.Errorf("failed to estimate context usage before compression: %w", err)
	}
	usedBefore := beforeResult.UsedTokens

	compressed, strategy, fallback, err := s.applyStrategy(ctx, history, providerID, cfg)
	if err != nil {
		return CompressionResult{}, fmt.Errorf("failed to apply compression strategy %s: %w", cfg.Strategy, err)
	}

	afterResult, err := s.estimator.Estimate(ctx, EstimatorInput{
		Messages:   compressed,
		ProviderID: providerID,
		ModelID:    modelID,
	})
	if err != nil {
		return CompressionResult{}, fmt.Errorf("failed to estimate context usage after compression: %w", err)
	}
	usedAfter := afterResult.UsedTokens

	// 压缩未减少 token 时回退到 drop 策略；仍不减少则报错且不返回变更。
	if usedAfter >= usedBefore {
		if strategy != StrategyDropEarliestN {
			compressed = applyDropEarliestN(history, cfg.DropN, cfg.RecentCount)
			afterResult, err = s.estimator.Estimate(ctx, EstimatorInput{
				Messages:   compressed,
				ProviderID: providerID,
				ModelID:    modelID,
			})
			if err != nil {
				return CompressionResult{}, fmt.Errorf("failed to estimate after fallback drop: %w", err)
			}
			usedAfter = afterResult.UsedTokens
			strategy = StrategyDropEarliestN
			fallback = true
		}
		if usedAfter >= usedBefore {
			return CompressionResult{}, fmt.Errorf(
				"compression did not reduce token usage (before=%d, after=%d, strategy=%s): no changes persisted",
				usedBefore, usedAfter, strategy,
			)
		}
	}

	return CompressionResult{
		Messages:         compressed,
		UsedBefore:       usedBefore,
		UsedAfter:        usedAfter,
		FallbackOccurred: fallback,
		Strategy:         strategy,
	}, nil
}

// Compress 从 DB 加载会话、压缩并持久化。
func (s *CompressionService) Compress(ctx context.Context, conversationID models.ConversationID, providerID, modelID string, cfg CompressionConfig) (CompressionResult, error) {
	entities, _, err := s.msgRepo.ListByConversation(ctx, conversationID, "", math.MaxInt32)
	if err != nil {
		return CompressionResult{}, fmt.Errorf("failed to list messages for conversation %s: %w", conversationID, err)
	}

	reverseEntities(entities)
	history := toModelMessages(entities)

	res, err := s.CompressMessages(ctx, history, providerID, modelID, cfg)
	if err != nil {
		return CompressionResult{}, err
	}

	if err := s.persist(ctx, conversationID, entities, history, res.Messages, res.Strategy); err != nil {
		return CompressionResult{}, fmt.Errorf("failed to persist compression result: %w", err)
	}

	return res, nil
}

// applyStrategy 根据配置分发到具体压缩策略。
func (s *CompressionService) applyStrategy(ctx context.Context, history []models.Message, providerID string, cfg CompressionConfig) ([]models.Message, CompressionStrategyKind, bool, error) {
	switch cfg.Strategy {
	case StrategyDropEarliestN:
		return applyDropEarliestN(history, cfg.DropN, cfg.RecentCount), StrategyDropEarliestN, false, nil
	case StrategySummarizeAndReplace:
		return applySummarizeAndReplace(history, cfg.AnchorCount, cfg.RecentCount, summarizeDeterministic), StrategySummarizeAndReplace, false, nil
	case StrategyLLMSelfSummarize:
		compressed, fallback, err := s.applyLLMSelfSummarize(ctx, history, providerID, cfg)
		if err != nil {
			return nil, "", false, fmt.Errorf("failed to apply LLM self summarization: %w", err)
		}
		return compressed, StrategyLLMSelfSummarize, fallback, nil
	default:
		return nil, "", false, fmt.Errorf("unsupported compression strategy: %s", cfg.Strategy)
	}
}

// applyLLMSelfSummarize 调用 LLM 生成中间消息摘要；若 LLM 不可用则回退到 drop。
func (s *CompressionService) applyLLMSelfSummarize(ctx context.Context, history []models.Message, providerID string, cfg CompressionConfig) ([]models.Message, bool, error) {
	provider, err := s.providers.Get(ctx, providerID)
	if err != nil {
		// LLM 不可用时回退到 drop 策略。
		return applyDropEarliestN(history, cfg.DropN, cfg.RecentCount), true, nil
	}

	client, err := s.llmFactory.CreateClient(provider)
	if err != nil {
		return applyDropEarliestN(history, cfg.DropN, cfg.RecentCount), true, nil
	}

	available, _ := client.CheckAvailability(ctx)
	if !available {
		return applyDropEarliestN(history, cfg.DropN, cfg.RecentCount), true, nil
	}

	middleStart := cfg.AnchorCount
	middleEnd := len(history) - cfg.RecentCount
	if middleStart < 0 {
		middleStart = 0
	}
	if middleEnd < middleStart {
		middleEnd = middleStart
	}
	middle := history[middleStart:middleEnd]
	if len(middle) == 0 {
		return history, false, nil
	}

	prompt := "请用一段话简要总结以下对话，保留关键信息：\n\n" + concatMessages(middle)
	summary, err := client.Chat(ctx, []models.Message{{Role: models.RoleUser, Content: prompt}})
	if err != nil {
		return applyDropEarliestN(history, cfg.DropN, cfg.RecentCount), true, nil
	}

	summary = strings.TrimSpace(summary)
	if summary == "" {
		return applyDropEarliestN(history, cfg.DropN, cfg.RecentCount), true, nil
	}

	return applySummarizeAndReplace(history, cfg.AnchorCount, cfg.RecentCount, func([]models.Message) string {
		return summary
	}), false, nil
}

// applyDropEarliestN 删除最早 N 条消息，但始终保留最近 recentCount 条。
func applyDropEarliestN(history []models.Message, N, recentCount int) []models.Message {
	if N <= 0 && recentCount <= 0 {
		return history
	}
	if recentCount < 0 {
		recentCount = 0
	}
	if len(history) <= recentCount {
		return history
	}

	maxDrop := len(history) - recentCount
	if N <= 0 {
		N = maxDrop
	}
	if N > maxDrop {
		N = maxDrop
	}

	result := make([]models.Message, 0, len(history)-N)
	result = append(result, history[N:]...)
	return result
}

// applySummarizeAndReplace 保留开头 anchorCount 条与结尾 recentCount 条，将中间块替换为一条系统摘要。
func applySummarizeAndReplace(history []models.Message, anchorCount, recentCount int, summarize func([]models.Message) string) []models.Message {
	if anchorCount < 0 {
		anchorCount = 0
	}
	if recentCount < 0 {
		recentCount = 0
	}

	middleStart := anchorCount
	middleEnd := len(history) - recentCount
	if middleStart < 0 {
		middleStart = 0
	}
	if middleEnd < middleStart {
		middleEnd = middleStart
	}

	anchors := history[:middleStart]
	recent := history[middleEnd:]
	middle := history[middleStart:middleEnd]

	if len(middle) == 0 {
		return history
	}

	summary := summarize(middle)
	result := make([]models.Message, 0, len(anchors)+1+len(recent))
	result = append(result, anchors...)
	result = append(result, models.Message{Role: models.RoleSystem, Content: summary})
	result = append(result, recent...)
	return result
}

// summarizeDeterministic 以确定性方式生成本地摘要。
func summarizeDeterministic(msgs []models.Message) string {
	if len(msgs) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("[历史摘要] 共 %d 条消息：", len(msgs)))
	for i, m := range msgs {
		if i > 0 {
			b.WriteString(" | ")
		}
		content := m.Content
		if len(content) > 80 {
			content = content[:80] + "…"
		}
		b.WriteString(fmt.Sprintf("%s: %s", m.Role, content))
	}
	return b.String()
}

// concatMessages 将多条消息拼接为用于 LLM 摘要的文本。
func concatMessages(msgs []models.Message) string {
	parts := make([]string, 0, len(msgs))
	for _, m := range msgs {
		parts = append(parts, fmt.Sprintf("%s: %s", m.Role, m.Content))
	}
	return strings.Join(parts, "\n\n")
}

// persist 将压缩后的消息结构持久化到仓库。
//
// 注意：当前实现先保存摘要再软删除被替换消息。若在软删除阶段失败，会尝试软删除已保存的摘要以回滚，
// 使会话尽量接近压缩前状态。真正的多语句原子性需要仓库层提供事务支持。
func (s *CompressionService) persist(ctx context.Context, conversationID models.ConversationID, entities []*entity.Message, before, after []models.Message, strategy CompressionStrategyKind) error {
	droppedIDs := make([]string, 0)
	var summaryID string

	switch strategy {
	case StrategyDropEarliestN:
		// 被删除的是前 len(before)-len(after) 条（按时间顺序）。
		dropCount := len(before) - len(after)
		if dropCount > len(entities) {
			dropCount = len(entities)
		}
		for i := 0; i < dropCount; i++ {
			droppedIDs = append(droppedIDs, entities[i].ID)
		}
	case StrategySummarizeAndReplace, StrategyLLMSelfSummarize:
		anchorCount := len(after) - 1 - cfgRecentCount(after)
		if anchorCount < 0 {
			anchorCount = 0
		}
		recentCount := len(after) - anchorCount - 1
		if recentCount < 0 {
			recentCount = 0
		}
		// 删除中间块对应的消息。
		for i := anchorCount; i < len(entities)-recentCount; i++ {
			droppedIDs = append(droppedIDs, entities[i].ID)
		}

		// 插入新的系统摘要消息。
		summaryIndex := anchorCount
		if summaryIndex >= len(after) {
			summaryIndex = len(after) - 1
		}

		// 摘要时间戳取最后一条锚点消息的时间，使其在按时间倒序加载时位于锚点之后、最近消息之前。
		// 若没有保留锚点，则使用当前时间。
		var summaryTimestamp time.Time
		if anchorCount > 0 {
			summaryTimestamp = entities[anchorCount-1].Timestamp
		} else {
			summaryTimestamp = time.Now().UTC()
		}
		summaryMsg := fromModelMessage(after[summaryIndex], summaryTimestamp)
		if err := s.msgRepo.Save(ctx, conversationID, &summaryMsg); err != nil {
			return fmt.Errorf("failed to save summary message: %w", err)
		}
		summaryID = summaryMsg.ID
	}

	for _, id := range droppedIDs {
		if err := s.msgRepo.SoftDelete(ctx, id); err != nil {
			// 若已插入摘要，先尝试回滚删除摘要，尽量恢复压缩前状态。
			if summaryID != "" {
				if rbErr := s.msgRepo.SoftDelete(ctx, summaryID); rbErr != nil {
					return fmt.Errorf("failed to soft delete message %s: %w; also failed to rollback summary %s: %w", id, err, summaryID, rbErr)
				}
			}
			return fmt.Errorf("failed to soft delete message %s: %w", id, err)
		}
	}

	return nil
}

// cfgRecentCount 从压缩结果中推断最近保留的消息数量。
func cfgRecentCount(after []models.Message) int {
	// 摘要后的结果格式为 anchors + summary + recent；最后 recentCount 条是用户/助手消息。
	count := 0
	for i := len(after) - 1; i >= 0; i-- {
		if after[i].Role == models.RoleSystem {
			break
		}
		count++
	}
	return count
}

// toModelMessages 将实体消息列表转换为 LLM 消息列表。
func toModelMessages(entities []*entity.Message) []models.Message {
	result := make([]models.Message, 0, len(entities))
	for _, e := range entities {
		if e == nil {
			continue
		}
		result = append(result, models.Message{
			Role:    e.Role,
			Content: e.Content,
		})
	}
	return result
}

// fromModelMessage 将 LLM 消息转换为领域实体消息，生成新 ID。
func fromModelMessage(m models.Message, timestamp time.Time) entity.Message {
	return entity.Message{
		ID:        generateMessageID(),
		Role:      m.Role,
		Content:   m.Content,
		Timestamp: timestamp,
	}
}
