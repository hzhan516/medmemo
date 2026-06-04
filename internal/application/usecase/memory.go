package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/wire"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/hzhan516/medmemo/pkg/models"
)

// MemoryQuerier 记忆检索接口。
type MemoryQuerier interface {
	RetrieveForContext(ctx context.Context, query, sessionID string, limit int) ([]*entity.HealthMemory, error)
}

// MemoryRetriever 编排语义记忆检索与上下文注入用例。
// 整合向量搜索、时间衰减评分、实体提及检测、会话间隙检测和置信度过滤，
// 为对话提供相关历史记忆。
type MemoryRetriever struct {
	embeddingSvc       port.EmbeddingService
	embeddingRepo      repository.EmbeddingRepository
	factRepo           repository.FactRepository
	decayScorer        *DecayScorer
	minConfidence      float64
	tokenBudget        int
	enabled            bool
	sessionAccessTimes map[string]time.Time
	mu                 sync.RWMutex
}

// NewMemoryRetriever 构造函数，供 Wire 注入。
func NewMemoryRetriever(
	embeddingSvc port.EmbeddingService,
	embeddingRepo repository.EmbeddingRepository,
	factRepo repository.FactRepository,
	decayScorer *DecayScorer,
) *MemoryRetriever {
	return &MemoryRetriever{
		embeddingSvc:       embeddingSvc,
		embeddingRepo:      embeddingRepo,
		factRepo:           factRepo,
		decayScorer:        decayScorer,
		minConfidence:      0.6,
		tokenBudget:        500,
		enabled:            true,
		sessionAccessTimes: make(map[string]time.Time),
	}
}

// SetEnabled 设置记忆注入全局开关。
func (m *MemoryRetriever) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
}

// IsEnabled 返回记忆注入开关状态。
func (m *MemoryRetriever) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// SetSessionEnabled 设置指定会话的记忆注入开关。
// 当 sessionID 为空时，行为同 SetEnabled（全局开关）。
func (m *MemoryRetriever) SetSessionEnabled(sessionID string, enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sessionID == "" {
		m.enabled = enabled
		return
	}
	// 使用特殊前缀标记会话级禁用
	key := "_disabled_" + sessionID
	if enabled {
		delete(m.sessionAccessTimes, key)
	} else {
		m.sessionAccessTimes[key] = time.Now().UTC()
	}
}

// IsSessionEnabled 返回指定会话的记忆注入是否启用。
func (m *MemoryRetriever) IsSessionEnabled(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.enabled {
		return false
	}
	if sessionID == "" {
		return true
	}
	_, disabled := m.sessionAccessTimes["_disabled_"+sessionID]
	return !disabled
}

// memoryCandidate 内部候选记忆结构，用于排序和过滤
type memoryCandidate struct {
	memory *entity.HealthMemory
	score  float64
}

// RetrieveForContext 为当前对话检索相关记忆，返回用于注入上下文的记忆片段。
// 流程：实体提及检测 → 会话间隙检测 → 向量相似搜索 → 事实关联 → 时间衰减评分 → 置信度过滤 → Token 预算截断。
func (m *MemoryRetriever) RetrieveForContext(ctx context.Context, query, sessionID string, limit int) ([]*entity.HealthMemory, error) {
	if limit <= 0 {
		limit = 3
	}

	// 1. 检查开关状态
	if !m.IsSessionEnabled(sessionID) {
		return nil, nil
	}

	// 2. 实体提及检测（触发条件一）
	mentionMemories, _ := m.detectEntityMentions(ctx, query)

	// 3. 会话间隙检测（触发条件二）
	sessionGapTriggered := m.checkSessionGap(sessionID)

	// 记录本次访问时间
	m.recordSessionAccess(sessionID)

	// 4. 生成查询向量（语义搜索）
	// 使用独立超时，避免 ONNX 推理耗时影响整体对话流程
	var semanticMemories []*entity.HealthMemory
	embedCtx, embedCancel := context.WithTimeout(context.Background(), 30*time.Second)
	queryVector, err := m.embeddingSvc.EmbedSingle(embedCtx, query)
	embedCancel()
	if err == nil {
		var searchErr error
		semanticMemories, searchErr = m.semanticSearch(ctx, queryVector, limit)
		if searchErr != nil {
			fmt.Printf("[MemoryRetriever] semantic search failed, memory injection degraded: %v\n", searchErr)
		}
	} else {
		fmt.Printf("[MemoryRetriever] embedding 生成失败，语义搜索降级: %v\n", err)
	}

	// 5. 合并结果并去重
	merged := m.mergeMemories(mentionMemories, semanticMemories, sessionGapTriggered, sessionID)

	// 6. Token 预算截断
	return m.applyTokenBudget(merged, limit), nil
}

// detectEntityMentions 检测 query 中是否包含已记忆实体的关键词。
// 原逻辑只匹配 subject，现扩展为匹配完整事实内容（subject + predicate + object）中的关键词。
// 命中时返回相关记忆，未命中时返回 nil 和 false。
func (m *MemoryRetriever) detectEntityMentions(ctx context.Context, query string) ([]*entity.HealthMemory, bool) {
	facts, err := m.factRepo.ListByStatus(ctx, entity.FactStatusApproved, 0, 1000)
	if err != nil || len(facts) == 0 {
		return nil, false
	}

	queryLower := strings.ToLower(query)
	var matched []*entity.HealthMemory
	seen := make(map[string]bool)

	for _, f := range facts {
		content := fmt.Sprintf("%s %s %s", f.Subject, f.Predicate, f.Object)
		contentLower := strings.ToLower(content)

		// 原逻辑：query 包含 subject
		if f.Subject != "" && strings.Contains(queryLower, strings.ToLower(f.Subject)) {
			if !seen[f.FactID] {
				seen[f.FactID] = true
				matched = append(matched, &entity.HealthMemory{
					ID:         models.MemoryID(f.FactID),
					Content:    content,
					Confidence: f.Confidence,
					CreatedAt:  f.CreatedAt,
				})
			}
			continue
		}

		// 新增：query 中包含事实内容里的非停用词
		if m.hasKeywordMatch(queryLower, contentLower) {
			if !seen[f.FactID] {
				seen[f.FactID] = true
				matched = append(matched, &entity.HealthMemory{
					ID:         models.MemoryID(f.FactID),
					Content:    content,
					Confidence: f.Confidence,
					CreatedAt:  f.CreatedAt,
				})
			}
		}
	}
	return matched, len(matched) > 0
}

// 常见中文停用词，用于关键词匹配时过滤。
var stopWords = map[string]bool{
	"的": true, "了": true, "是": true, "在": true, "我": true, "你": true,
	"吗": true, "知道": true, "现在": true, "有": true, "什么": true,
	"呢": true, "啊": true, "吧": true, "就": true, "都": true,
	"很": true, "还": true, "也": true, "要": true, "会": true,
	"能": true, "可以": true, "和": true, "或": true, "与": true,
	"对": true, "为": true, "从": true, "到": true, "把": true,
	"被": true, "给": true, "让": true, "向": true, "比": true,
	"跟": true, "同": true, "及": true, "而": true, "但": true,
	"不过": true, "虽然": true, "如果": true, "因为": true, "所以": true,
	"怎样": true, "怎么": true, "如何": true, "多少": true, "几": true,
}

// hasKeywordMatch 检查 query 中是否包含事实内容里的任何非停用词。
func (m *MemoryRetriever) hasKeywordMatch(queryLower, contentLower string) bool {
	// 提取事实内容中的候选词（2-8 个字符的连续词组）
	runes := []rune(contentLower)
	for i := 0; i < len(runes); i++ {
		for length := 2; length <= 8 && i+length <= len(runes); length++ {
			word := string(runes[i : i+length])
			if stopWords[word] {
				continue
			}
			// 过滤纯数字、纯英文短词
			if isMeaninglessWord(word) {
				continue
			}
			if strings.Contains(queryLower, word) {
				return true
			}
		}
	}
	return false
}

// isMeaninglessWord 过滤无意义的短词（纯数字或纯单字母）。
func isMeaninglessWord(word string) bool {
	if len(word) <= 1 {
		return true
	}
	// 纯数字
	allDigit := true
	for _, r := range word {
		if r < '0' || r > '9' {
			allDigit = false
			break
		}
	}
	if allDigit && len(word) <= 3 {
		return true
	}
	return false
}

// checkSessionGap 检测会话消息间隔是否超过 10 分钟。
func (m *MemoryRetriever) checkSessionGap(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	m.mu.RLock()
	lastAccess, ok := m.sessionAccessTimes[sessionID]
	m.mu.RUnlock()
	if !ok {
		// 首次访问该会话，不视为间隙（因为没有历史）
		return false
	}
	return time.Since(lastAccess) > 10*time.Minute
}

// recordSessionAccess 记录会话访问时间。
func (m *MemoryRetriever) recordSessionAccess(sessionID string) {
	if sessionID == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionAccessTimes[sessionID] = time.Now().UTC()
}

// semanticSearch 执行语义向量搜索。
func (m *MemoryRetriever) semanticSearch(ctx context.Context, queryVector []float32, limit int) ([]*entity.HealthMemory, error) {
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

	// 按衰减后分数降序排序
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	var results []*entity.HealthMemory
	for _, c := range candidates {
		results = append(results, c.memory)
	}
	return results, nil
}

// mergeMemories 合并实体提及和语义搜索结果，去重并按优先级排序。
// 实体提及的记忆优先级更高（排在前面）。
func (m *MemoryRetriever) mergeMemories(mentionMemories, semanticMemories []*entity.HealthMemory, sessionGapTriggered bool, sessionID string) []*entity.HealthMemory {
	seen := make(map[models.MemoryID]bool)
	var merged []*entity.HealthMemory

	// 先加入实体提及的记忆（高优先级）
	for _, mem := range mentionMemories {
		if seen[mem.ID] {
			continue
		}
		seen[mem.ID] = true
		merged = append(merged, mem)
	}

	// 会话间隙触发时，额外召回该会话相关的记忆
	if sessionGapTriggered && sessionID != "" {
		_ = sessionID // TODO: 通过 raw_dialogues 关联实现会话间隙记忆召回
	}

	// 再加入语义搜索的结果
	for _, mem := range semanticMemories {
		if seen[mem.ID] {
			continue
		}
		seen[mem.ID] = true
		merged = append(merged, mem)
	}

	return merged
}

// applyTokenBudget 应用 Token 预算截断。
func (m *MemoryRetriever) applyTokenBudget(memories []*entity.HealthMemory, limit int) []*entity.HealthMemory {
	var results []*entity.HealthMemory
	var tokenCount int
	for _, mem := range memories {
		memTokens := len([]rune(mem.Content))
		if len(results) > 0 && tokenCount+memTokens > m.tokenBudget {
			break
		}
		results = append(results, mem)
		tokenCount += memTokens
		if len(results) >= limit {
			break
		}
	}
	return results
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
