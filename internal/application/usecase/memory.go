package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/wire"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/hzhan516/medmemo/pkg/models"
)

// MemoryRetriever 编排语义记忆检索与上下文注入用例。
// 整合向量搜索、时间衰减评分和置信度过滤，为对话提供相关历史记忆。
type MemoryRetriever struct {
	embeddingSvc  port.EmbeddingService
	embeddingRepo repository.EmbeddingRepository
	factRepo      repository.FactRepository
	decayScorer   *DecayScorer
	minConfidence float64
	tokenBudget   int
}

// NewMemoryRetriever 构造函数，供 Wire 注入。
func NewMemoryRetriever(
	embeddingSvc port.EmbeddingService,
	embeddingRepo repository.EmbeddingRepository,
	factRepo repository.FactRepository,
	decayScorer *DecayScorer,
) *MemoryRetriever {
	return &MemoryRetriever{
		embeddingSvc:  embeddingSvc,
		embeddingRepo: embeddingRepo,
		factRepo:      factRepo,
		decayScorer:   decayScorer,
		minConfidence: 0.6,
		tokenBudget:   500,
	}
}

// memoryCandidate 内部候选记忆结构，用于排序和过滤
type memoryCandidate struct {
	memory *entity.HealthMemory
	score  float64
}

// RetrieveForContext 为当前对话检索相关记忆，返回用于注入上下文的记忆片段。
// 流程：嵌入查询 → 向量相似搜索 → 事实关联 → 时间衰减评分 → 置信度过滤 → Token 预算截断。
func (m *MemoryRetriever) RetrieveForContext(ctx context.Context, query string, limit int) ([]*entity.HealthMemory, error) {
	if limit <= 0 {
		limit = 3
	}

	// 1. 生成查询向量
	queryVector, err := m.embeddingSvc.EmbedSingle(ctx, query)
	if err != nil {
		// 嵌入失败时降级返回空结果，不中断对话
		return nil, nil
	}

	// 2. 向量相似搜索（多检索一些，供后续过滤和衰减）
	searchLimit := limit * 3
	if searchLimit < 10 {
		searchLimit = 10
	}
	scoredEmbeddings, err := m.embeddingRepo.SearchSimilar(ctx, queryVector, searchLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar embeddings: %w", err)
	}

	now := time.Now().UTC()
	var candidates []memoryCandidate

	// 3. 关联事实、计算衰减分数
	for _, se := range scoredEmbeddings {
		fact, err := m.factRepo.GetByID(ctx, se.FactID)
		if err != nil {
			continue // 事实不存在或查询失败，跳过
		}
		if fact.Status != entity.FactStatusApproved {
			continue // 只使用已审批的事实
		}

		// 时间衰减评分：similarity * exp(-lambda * ageDays)
		decayScore := m.decayScorer.ScoreFromCreatedAt(se.Similarity, fact.CreatedAt, now)
		weightedConfidence := fact.Confidence * decayScore

		if weightedConfidence < m.minConfidence {
			continue // 置信度不足，跳过
		}

		candidates = append(candidates, memoryCandidate{
			memory: &entity.HealthMemory{
				ID:         models.MemoryID(fact.FactID),
				Content:    fmt.Sprintf("%s %s %s", fact.Subject, fact.Predicate, fact.Object),
				Confidence: weightedConfidence,
				CreatedAt:  fact.CreatedAt,
			},
			score: weightedConfidence,
		})
	}

	// 4. 按衰减后分数降序排序
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// 5. Token 预算截断（中文字符按 1 token 估算，保留首条完整记忆）
	var results []*entity.HealthMemory
	var tokenCount int
	for _, c := range candidates {
		memTokens := len([]rune(c.memory.Content))
		if len(results) > 0 && tokenCount+memTokens > m.tokenBudget {
			break
		}
		results = append(results, c.memory)
		tokenCount += memTokens
		if len(results) >= limit {
			break
		}
	}

	return results, nil
}

// ArchiveConversation 将对话归档为长期记忆（L2/L3）。
func (m *MemoryRetriever) ArchiveConversation(ctx context.Context, convID models.ConversationID) error {
	// TODO(作者): 实现对话摘要与实体提取后的记忆归档 [Issue#004]
	return fmt.Errorf("not implemented")
}

// FormatMemoriesForInjection 将记忆列表格式化为系统提示注入文本。
func FormatMemoriesForInjection(memories []*entity.HealthMemory) string {
	if len(memories) == 0 {
		return ""
	}
	var parts []string
	for i, m := range memories {
		parts = append(parts, fmt.Sprintf("%d. %s (可信度: %.0f%%)", i+1, m.Content, m.Confidence*100))
	}
	return "[相关记忆]\n" + strings.Join(parts, "\n")
}

var _ = wire.NewSet // 占位，确保 wire 包被引用
