package usecase

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

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
// 整合多路召回（intent/keyword/vector/recent）、合并去重、基础重排和诊断日志，
// 为对话提供相关历史记忆。
type MemoryRetriever struct {
	embeddingSvc       port.EmbeddingService
	embeddingRepo      repository.EmbeddingRepository
	factRepo           repository.FactRepository
	memoryRepo         port.MemoryRepository
	decayScorer        *DecayScorer
	migrationState     *MigrationState
	intentResolver     *IntentResolver
	expansionSvc       *QueryExpansionService
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
	memoryRepo port.MemoryRepository,
	decayScorer *DecayScorer,
	migrationState *MigrationState,
	intentResolver *IntentResolver,
	expansionSvc *QueryExpansionService,
) *MemoryRetriever {
	return &MemoryRetriever{
		embeddingSvc:       embeddingSvc,
		embeddingRepo:      embeddingRepo,
		factRepo:           factRepo,
		memoryRepo:         memoryRepo,
		decayScorer:        decayScorer,
		migrationState:     migrationState,
		intentResolver:     intentResolver,
		expansionSvc:       expansionSvc,
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

// prepareRetrievalRequest 从原始 query 生成用于多路召回的 RetrievalRequest。
// 包含规范化、意图解析和本地 query expansion。
func (m *MemoryRetriever) prepareRetrievalRequest(query, sessionID string, limit int) *RetrievalRequest {
	req := &RetrievalRequest{
		RawQuery:  query,
		SessionID: sessionID,
		Limit:     limit,
		Subject:   "用户",
	}

	// 规范化
	if m.expansionSvc != nil {
		req.Normalized = m.expansionSvc.Normalize(query)
	} else {
		req.Normalized = query
	}

	// 意图解析
	if m.intentResolver != nil && req.Normalized != "" {
		req.Intent = m.intentResolver.Resolve(req.Normalized)
	}

	// 本地 query expansion
	req.ExpandedQuery = BuildExpandedQuery(req.Normalized, req.Intent)

	return req
}

// memoryCandidate 内部候选记忆结构，用于排序和过滤
type memoryCandidate struct {
	memory *entity.HealthMemory
	score  float64
}

// RetrieveForContext 为当前对话检索相关记忆，返回用于注入上下文的记忆片段。
// 多路召回流程：prepareRequest → intent/keyword/vector/recent → merge → rerank → tokenBudget。
func (m *MemoryRetriever) RetrieveForContext(ctx context.Context, query, sessionID string, limit int) ([]*entity.HealthMemory, error) {
	_, memories, err := m.retrieveWithDiagnostics(ctx, query, sessionID, limit)
	return memories, err
}

// retrieveWithDiagnostics 执行完整多路召回并返回诊断信息，供内部测试使用。
func (m *MemoryRetriever) retrieveWithDiagnostics(ctx context.Context, query, sessionID string, limit int) (*RetrievalDiagnostics, []*entity.HealthMemory, error) {
	if limit <= 0 {
		limit = 3
	}

	diag := &RetrievalDiagnostics{}

	// 1. 检查开关状态
	if !m.IsSessionEnabled(sessionID) {
		return diag, nil, nil
	}

	// 2. 准备请求
	req := m.prepareRetrievalRequest(query, sessionID, limit)
	diag.DetectedIntent = req.Intent
	diag.ExpandedQuery = req.ExpandedQuery

	// 3. 会话间隙检测
	sessionGapTriggered := m.checkSessionGap(sessionID)
	m.recordSessionAccess(sessionID)

	// 4. 四路并行召回（每个 goroutine 只写独占局部变量，消除 data race）
	var (
		intentCandidates     []RetrievalCandidate
		intentStatus         PathStatus
		keywordCandidates    []RetrievalCandidate
		keywordStatus        PathStatus
		vectorCandidates     []RetrievalCandidate
		vectorStatus         PathStatus
		recentCandidates     []RetrievalCandidate
		recentStatus         PathStatus
		sessionGapCandidates []RetrievalCandidate
		sessionGapStatus     PathStatus
	)

	var wg sync.WaitGroup

	// intent recall
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		candidates, ps := m.recallByIntent(ctx, req)
		ps.Latency = time.Since(start)
		intentCandidates = candidates
		intentStatus = ps
	}()

	// keyword recall（使用 recallByKeyword 返回的 PathStatus，不手写简化 status）
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		candidates, ps := m.recallByKeyword(ctx, req)
		ps.Latency = time.Since(start)
		keywordCandidates = candidates
		keywordStatus = ps
	}()

	// vector recall
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		candidates, ps := m.recallByVector(ctx, req)
		ps.Latency = time.Since(start)
		vectorCandidates = candidates
		vectorStatus = ps
	}()

	// recent same-intent recall
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		candidates, ps := m.recallRecentSameIntent(ctx, req)
		ps.Latency = time.Since(start)
		recentCandidates = candidates
		recentStatus = ps
	}()

	// session-gap recall：会话间隔超过阈值时，主动召回本会话已审批事实
	if sessionGapTriggered {
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			candidates, ps := m.recallBySessionGap(ctx, sessionID)
			ps.Latency = time.Since(start)
			sessionGapCandidates = candidates
			sessionGapStatus = ps
		}()
	} else {
		sessionGapStatus = PathStatus{Path: PathSessionGap, Status: "skipped", Reason: "session gap not triggered"}
	}

	wg.Wait()

	// wg.Wait() 后由主 goroutine 统一写入 diag，保证固定顺序和数量
	diag.IntentCandidates = intentCandidates
	diag.KeywordCandidates = keywordCandidates
	diag.VectorCandidates = vectorCandidates
	diag.RecentCandidates = recentCandidates
	diag.SessionGapCandidates = sessionGapCandidates
	diag.PathStatuses = []PathStatus{intentStatus, keywordStatus, vectorStatus, recentStatus, sessionGapStatus}

	// 5. 合并去重
	diag.MergedCandidates = mergeCandidates(
		diag.IntentCandidates,
		diag.KeywordCandidates,
		diag.VectorCandidates,
		diag.RecentCandidates,
		diag.SessionGapCandidates,
	)

	// 6. 重排
	diag.MergedCandidates = rerank(diag.MergedCandidates, req)

	// 7. Token 预算截断（使用配置化的 tokenBudget 而非硬编码）
	diag.SelectedMemories, diag.RejectedCandidates = applyTokenBudgetToCandidates(diag.MergedCandidates, limit, m.tokenBudget)
	diag.TotalApprovedFacts = len(diag.MergedCandidates)
	diag.TotalRejected = len(diag.RejectedCandidates)

	// 8. 诊断日志
	logDiagnostics(diag)

	// 9. 转换为 HealthMemory
	var memories []*entity.HealthMemory
	for _, c := range diag.SelectedMemories {
		memories = append(memories, &entity.HealthMemory{
			ID:         models.MemoryID(c.FactID),
			Content:    c.Content,
			Confidence: c.Confidence,
			CreatedAt:  c.CreatedAt,
		})
	}

	return diag, memories, nil
}

// recallByIntent 意图召回路径：通过 IntentResolver 的 predicates 查询 approved facts。
// 返回同意图候选列表，参与后续普通排序。
// 与 recallRecentSameIntent 区分：此为多候选列表，后者只取最新 1 条 boost。
func (m *MemoryRetriever) recallByIntent(ctx context.Context, req *RetrievalRequest) ([]RetrievalCandidate, PathStatus) {
	if req.Intent == nil || len(req.Intent.Predicates) == 0 {
		return nil, PathStatus{Path: PathIntent, Status: "skipped", Reason: "no intent detected or no predicates"}
	}

	facts, err := m.factRepo.FindApprovedByPredicates(ctx, req.Subject, req.Intent.Predicates, req.Limit)
	if err != nil {
		return nil, PathStatus{Path: PathIntent, Status: "failure", Reason: fmt.Sprintf("query failed: %v", err)}
	}

	now := time.Now().UTC()
	var candidates []RetrievalCandidate
	for _, f := range facts {
		content := fmt.Sprintf("%s %s %s", f.Subject, f.Predicate, f.Object)
		candidates = append(candidates, RetrievalCandidate{
			FactID:       f.FactID,
			Content:      content,
			Snippet:      truncateSnippet(content, 50),
			CreatedAt:    f.CreatedAt,
			Confidence:   f.Confidence,
			MatchedPaths: []RetrievalPath{PathIntent},
			IntentLevel:  intentConfidenceToLevel(req.Intent.Confidence),
			RecencyScore: m.decayScorer.ScoreFromCreatedAt(1.0, f.CreatedAt, now),
			Reasons:      []string{fmt.Sprintf("intent: %s, predicate: %s", req.Intent.Intent, f.Predicate)},
		})
	}

	status := PathStatus{Path: PathIntent, Status: "success"}
	if len(candidates) == 0 {
		status.Reason = "no approved facts matching intent"
	}
	return candidates, status
}

// intentConfidenceToLevel 将 IntentConfidence 枚举映射为数值级别。
// High→3, Medium→2, Low→1, 无→0。
func intentConfidenceToLevel(c IntentConfidence) int {
	switch c {
	case ConfidenceHigh:
		return 3
	case ConfidenceMedium:
		return 2
	case ConfidenceLow:
		return 1
	default:
		return 0
	}
}

// recallByKeyword 关键词召回路径：匹配 query 与 approved fact 的 retrieval text。
// 返回 RetrievalCandidate 列表和路径状态，供多路合并使用。
func (m *MemoryRetriever) recallByKeyword(ctx context.Context, req *RetrievalRequest) ([]RetrievalCandidate, PathStatus) {
	facts, err := m.factRepo.ListByStatus(ctx, entity.FactStatusApproved, 0, 1000)
	if err != nil || len(facts) == 0 {
		status := PathStatus{Path: PathKeyword, Status: "skipped"}
		if err != nil {
			status.Status = "failure"
			status.Reason = fmt.Sprintf("list approved facts failed: %v", err)
		} else {
			status.Reason = "no approved facts"
		}
		return nil, status
	}

	queryLower := strings.ToLower(req.Normalized)
	if queryLower == "" {
		queryLower = strings.ToLower(req.RawQuery)
	}

	var candidates []RetrievalCandidate
	seen := make(map[string]bool)
	now := time.Now().UTC()

	for _, f := range facts {
		content := fmt.Sprintf("%s %s %s", f.Subject, f.Predicate, f.Object)
		retrievalText := BuildFactRetrievalText(f)
		matchLower := strings.ToLower(retrievalText)

		var matched bool
		var score float64
		var reason string

		if f.Subject != "" && strings.Contains(queryLower, strings.ToLower(f.Subject)) {
			matched = true
			score = 1.0
			reason = fmt.Sprintf("subject match: %s", f.Subject)
		} else if m.hasKeywordMatch(queryLower, matchLower) {
			matched = true
			score = 0.5
			reason = fmt.Sprintf("keyword match: %s", retrievalText)
		}

		if matched && !seen[f.FactID] {
			seen[f.FactID] = true
			candidates = append(candidates, RetrievalCandidate{
				FactID:       f.FactID,
				Content:      content,
				Snippet:      truncateSnippet(content, 50),
				CreatedAt:    f.CreatedAt,
				Confidence:   f.Confidence,
				MatchedPaths: []RetrievalPath{PathKeyword},
				KeywordScore: score,
				RecencyScore: m.decayScorer.ScoreFromCreatedAt(1.0, f.CreatedAt, now),
				Reasons:      []string{reason},
			})
		}
	}

	status := PathStatus{Path: PathKeyword, Status: "success"}
	if len(candidates) == 0 {
		status.Status = "success"
		status.Reason = "no keyword matches"
	}
	return candidates, status
}

// truncateRunes 通用 rune 截断工具，按 rune 数截断，过长时追加 suffix。
func truncateRunes(s string, maxRunes int, suffix string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + suffix
}

// truncateSnippet 截断文本为指定长度的诊断用 snippet。
func truncateSnippet(s string, maxLen int) string {
	return truncateRunes(s, maxLen, "...")
}

// recallByVector 向量召回路径：使用 expanded_query 进行 embedding 和语义搜索。
// embedding 或搜索失败不影响其他路径，diagnostics 记录失败原因。
func (m *MemoryRetriever) recallByVector(ctx context.Context, req *RetrievalRequest) ([]RetrievalCandidate, PathStatus) {
	searchQuery := req.ExpandedQuery
	if searchQuery == "" {
		searchQuery = req.Normalized
	}
	if searchQuery == "" {
		return nil, PathStatus{Path: PathVector, Status: "skipped", Reason: "no expanded query"}
	}

	// 生成 embedding（独立超时，避免阻塞其他路径）
	embedCtx, embedCancel := context.WithTimeout(context.Background(), 30*time.Second)
	queryVector, err := m.embeddingSvc.EmbedSingle(embedCtx, searchQuery)
	embedCancel()
	if err != nil {
		return nil, PathStatus{Path: PathVector, Status: "failure", Reason: fmt.Sprintf("embedding failed: %v", err)}
	}

	searchLimit := req.Limit * 3
	if searchLimit < 10 {
		searchLimit = 10
	}

	var scoredEmbeddings []*entity.ScoredEmbedding
	if m.migrationState != nil && m.migrationState.IsComplete() {
		scoredEmbeddings, err = m.embeddingRepo.SearchSimilarFiltered(
			ctx, queryVector, searchLimit, m.embeddingSvc.ModelVersion())
	} else {
		scoredEmbeddings, err = m.embeddingRepo.SearchSimilar(ctx, queryVector, searchLimit)
	}
	if err != nil {
		return nil, PathStatus{Path: PathVector, Status: "failure", Reason: fmt.Sprintf("vector search failed: %v", err)}
	}

	now := time.Now().UTC()
	factIDs := make([]string, 0, len(scoredEmbeddings))
	for _, se := range scoredEmbeddings {
		factIDs = append(factIDs, se.FactID)
	}

	facts, err := m.factRepo.FindByIDs(ctx, factIDs)
	if err != nil {
		return nil, PathStatus{Path: PathVector, Status: "failure", Reason: fmt.Sprintf("batch load facts failed: %v", err)}
	}

	var candidates []RetrievalCandidate
	for _, se := range scoredEmbeddings {
		fact, ok := facts[se.FactID]
		if !ok || fact.Status != entity.FactStatusApproved {
			continue
		}

		decayScore := m.decayScorer.ScoreFromCreatedAt(se.Similarity, fact.CreatedAt, now)
		weightedConf := fact.Confidence * decayScore
		if weightedConf < m.minConfidence {
			continue
		}

		content := fmt.Sprintf("%s %s %s", fact.Subject, fact.Predicate, fact.Object)
		candidates = append(candidates, RetrievalCandidate{
			FactID:           fact.FactID,
			Content:          content,
			Snippet:          truncateSnippet(content, 50),
			CreatedAt:        fact.CreatedAt,
			Confidence:       weightedConf,
			MatchedPaths:     []RetrievalPath{PathVector},
			VectorSimilarity: se.Similarity,
			RecencyScore:     decayScore,
			Reasons:          []string{fmt.Sprintf("vector similarity: %.3f", se.Similarity)},
		})
	}

	// 按加权分数降序
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Confidence > candidates[j].Confidence
	})

	status := PathStatus{Path: PathVector, Status: "success"}
	if len(candidates) == 0 {
		status.Reason = "no vector matches above confidence threshold"
	}
	return candidates, status
}

// recallRecentSameIntent 最近相同意图召回路径：仅取最新 1 条 approved fact 用于 personal attribute boost/override。
// 仅在 detected_intent.Confidence == ConfidenceHigh 时触发。
func (m *MemoryRetriever) recallRecentSameIntent(ctx context.Context, req *RetrievalRequest) ([]RetrievalCandidate, PathStatus) {
	if req.Intent == nil || req.Intent.Confidence != ConfidenceHigh || len(req.Intent.Predicates) == 0 {
		return nil, PathStatus{Path: PathRecent, Status: "skipped", Reason: "intent confidence not High"}
	}

	facts, err := m.factRepo.FindApprovedByPredicates(ctx, req.Subject, req.Intent.Predicates, 1)
	if err != nil {
		return nil, PathStatus{Path: PathRecent, Status: "failure", Reason: fmt.Sprintf("query failed: %v", err)}
	}
	if len(facts) == 0 {
		return nil, PathStatus{Path: PathRecent, Status: "success", Reason: "no recent same-intent facts"}
	}

	now := time.Now().UTC()
	var candidates []RetrievalCandidate
	for _, f := range facts {
		content := fmt.Sprintf("%s %s %s", f.Subject, f.Predicate, f.Object)
		candidates = append(candidates, RetrievalCandidate{
			FactID:       f.FactID,
			Content:      content,
			Snippet:      truncateSnippet(content, 50),
			CreatedAt:    f.CreatedAt,
			Confidence:   f.Confidence,
			MatchedPaths: []RetrievalPath{PathRecent},
			IntentLevel:  intentConfidenceToLevel(req.Intent.Confidence),
			RecencyScore: m.decayScorer.ScoreFromCreatedAt(1.0, f.CreatedAt, now),
			Reasons:      []string{fmt.Sprintf("recent same-intent: %s", req.Intent.Intent)},
		})
	}

	return candidates, PathStatus{Path: PathRecent, Status: "success"}
}

// recallBySessionGap 会话间隙召回路径：当用户重新打开间隔超过 10 分钟的会话时，
// 主动召回该会话关联的已审批事实，避免上下文断层。
func (m *MemoryRetriever) recallBySessionGap(ctx context.Context, sessionID string) ([]RetrievalCandidate, PathStatus) {
	if sessionID == "" {
		return nil, PathStatus{Path: PathSessionGap, Status: "skipped", Reason: "empty session id"}
	}
	facts, err := m.factRepo.FindBySession(ctx, sessionID)
	if err != nil {
		return nil, PathStatus{Path: PathSessionGap, Status: "failure", Reason: fmt.Sprintf("query failed: %v", err)}
	}

	now := time.Now().UTC()
	var candidates []RetrievalCandidate
	for _, f := range facts {
		if f.Status != entity.FactStatusApproved {
			continue
		}
		content := fmt.Sprintf("%s %s %s", f.Subject, f.Predicate, f.Object)
		candidates = append(candidates, RetrievalCandidate{
			FactID:       f.FactID,
			Content:      content,
			Snippet:      truncateSnippet(content, 50),
			CreatedAt:    f.CreatedAt,
			Confidence:   f.Confidence,
			MatchedPaths: []RetrievalPath{PathSessionGap},
			RecencyScore: m.decayScorer.ScoreFromCreatedAt(1.0, f.CreatedAt, now),
			Reasons:      []string{fmt.Sprintf("session gap recall: %s", sessionID)},
		})
	}

	status := PathStatus{Path: PathSessionGap, Status: "success"}
	if len(candidates) == 0 {
		status.Reason = "no approved facts linked to session"
	}
	return candidates, status
}

// mergeCandidates 多路候选合并去重，以 fact_id 为主键。
// 同一 fact 多路命中时合并 matched_paths、reasons 和评分。
func mergeCandidates(paths ...[]RetrievalCandidate) []RetrievalCandidate {
	byID := make(map[string]*RetrievalCandidate)

	for _, candidates := range paths {
		for i := range candidates {
			c := &candidates[i]
			if existing, ok := byID[c.FactID]; ok {
				// 合并 matched_paths
				existing.MatchedPaths = append(existing.MatchedPaths, c.MatchedPaths...)
				// 合并 reasons
				existing.Reasons = append(existing.Reasons, c.Reasons...)
				// 取最高评分
				if c.IntentLevel > existing.IntentLevel {
					existing.IntentLevel = c.IntentLevel
				}
				if c.KeywordScore > existing.KeywordScore {
					existing.KeywordScore = c.KeywordScore
				}
				if c.VectorSimilarity > existing.VectorSimilarity {
					existing.VectorSimilarity = c.VectorSimilarity
				}
				if c.RecencyScore > existing.RecencyScore {
					existing.RecencyScore = c.RecencyScore
				}
				if c.Confidence > existing.Confidence {
					existing.Confidence = c.Confidence
				}
			} else {
				cp := *c
				p := new(RetrievalCandidate)
				*p = cp
				byID[c.FactID] = p
			}
		}
	}

	result := make([]RetrievalCandidate, 0, len(byID))
	for _, c := range byID {
		result = append(result, *c)
	}
	return result
}

// rerank 基础重排：按 intent_level → keyword_score → recency → vector_similarity → confidence → created_at 降序排序。
// 个人属性覆盖规则：ConfidenceHigh 且存在 recent path 命中时，该 fact 排最前。
func rerank(candidates []RetrievalCandidate, req *RetrievalRequest) []RetrievalCandidate {
	if len(candidates) <= 1 {
		return candidates
	}

	var boostPersonal bool
	if req != nil && req.Intent != nil && req.Intent.Confidence == ConfidenceHigh {
		boostPersonal = true
	}

	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]

		// recent boost: 个人属性问题优先推 recent path 候选
		if boostPersonal {
			aRecent := slices.Contains(a.MatchedPaths, PathRecent)
			bRecent := slices.Contains(b.MatchedPaths, PathRecent)
			if aRecent != bRecent {
				return aRecent
			}
		}

		// 排序键
		if a.IntentLevel != b.IntentLevel {
			return a.IntentLevel > b.IntentLevel
		}
		if a.KeywordScore != b.KeywordScore {
			return a.KeywordScore > b.KeywordScore
		}
		if a.RecencyScore != b.RecencyScore {
			return a.RecencyScore > b.RecencyScore
		}
		if a.VectorSimilarity != b.VectorSimilarity {
			return a.VectorSimilarity > b.VectorSimilarity
		}
		if a.Confidence != b.Confidence {
			return a.Confidence > b.Confidence
		}
		return a.CreatedAt.After(b.CreatedAt)
	})

	return candidates
}

// applyTokenBudgetToCandidates 对 RetrievalCandidate 列表应用 Token 预算截断。
// 返回选中的和被拒绝的候选。
func applyTokenBudgetToCandidates(candidates []RetrievalCandidate, limit int, tokenBudget int) ([]RetrievalCandidate, []RetrievalCandidate) {
	var selected, rejected []RetrievalCandidate
	var tokenCount int
	for _, c := range candidates {
		memTokens := utf8.RuneCountInString(c.Content)
		if len(selected) > 0 && tokenCount+memTokens > tokenBudget {
			r := c
			r.RejectReason = "token budget exceeded"
			rejected = append(rejected, r)
			continue
		}
		selected = append(selected, c)
		tokenCount += memTokens
		if len(selected) >= limit {
			break
		}
	}
	return selected, rejected
}

// logDiagnostics 输出检索诊断日志，默认 summary，debug 输出候选明细。
func logDiagnostics(diag *RetrievalDiagnostics) {
	if diag == nil {
		return
	}

	intentStr := "none"
	if diag.DetectedIntent != nil {
		intentStr = fmt.Sprintf("%s(conf=%d)", diag.DetectedIntent.Intent, diag.DetectedIntent.Confidence)
	}

	fmt.Printf("[MemoryRetriever] diag intent=%s expanded=%q intentC=%d keywordC=%d vectorC=%d recentC=%d sessionGapC=%d merged=%d selected=%d rejected=%d\n",
		intentStr, truncateSnippet(diag.ExpandedQuery, 80),
		len(diag.IntentCandidates), len(diag.KeywordCandidates),
		len(diag.VectorCandidates), len(diag.RecentCandidates),
		len(diag.SessionGapCandidates),
		len(diag.MergedCandidates), len(diag.SelectedMemories), len(diag.RejectedCandidates))

	// 明细仅在 vector 失败或零召回时输出（debug 级别）
	for _, ps := range diag.PathStatuses {
		if ps.Status == "failure" {
			fmt.Printf("[MemoryRetriever] diag path=%s status=%s reason=%s latency=%v\n",
				ps.Path, ps.Status, ps.Reason, ps.Latency)
		}
	}
}

// detectEntityMentions 检测 query 中是否包含已记忆实体的关键词。
// 原逻辑只匹配 subject，现扩展为匹配完整事实内容（subject + predicate + object）中的关键词。
// 命中时返回相关记忆，未命中时返回 nil 和 false。
// 兼容旧版调用方，内部委托 recallByKeyword。
func (m *MemoryRetriever) detectEntityMentions(ctx context.Context, query string) ([]*entity.HealthMemory, bool) {
	req := m.prepareRetrievalRequest(query, "", 10)
	candidates, _ := m.recallByKeyword(ctx, req)
	if len(candidates) == 0 {
		return nil, false
	}
	var memories []*entity.HealthMemory
	for _, c := range candidates {
		memories = append(memories, &entity.HealthMemory{
			ID:         models.MemoryID(c.FactID),
			Content:    c.Content,
			Confidence: c.Confidence,
			CreatedAt:  c.CreatedAt,
		})
	}
	return memories, true
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
	var scoredEmbeddings []*entity.ScoredEmbedding
	var err error
	if m.migrationState != nil && m.migrationState.IsComplete() {
		scoredEmbeddings, err = m.embeddingRepo.SearchSimilarFiltered(
			ctx, queryVector, searchLimit, m.embeddingSvc.ModelVersion())
	} else {
		scoredEmbeddings, err = m.embeddingRepo.SearchSimilar(ctx, queryVector, searchLimit)
	}
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
// 实体提及的记忆优先级更高（排在前面）。会话间隙触发时，通过 factRepo.FindBySession
// 召回该会话关联的已审批事实并插入到mentionMemories之后，减少上下文断层。
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

	// 会话间隙触发时，额外召回该会话相关的已审批事实
	if sessionGapTriggered && sessionID != "" && m.factRepo != nil {
		facts, err := m.factRepo.FindBySession(context.Background(), sessionID)
		if err == nil {
			for _, f := range facts {
				if f.Status != entity.FactStatusApproved {
					continue
				}
				mem := &entity.HealthMemory{
					ID:         models.MemoryID(f.FactID),
					Content:    fmt.Sprintf("%s %s %s", f.Subject, f.Predicate, f.Object),
					Confidence: f.Confidence,
					CreatedAt:  f.CreatedAt,
				}
				if seen[mem.ID] {
					continue
				}
				seen[mem.ID] = true
				merged = append(merged, mem)
			}
		}
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

// ArchiveConversation 将对话归档为 L2 短期记忆。
// 通过 factRepo.FindBySession 汇总本会话已审批事实，生成摘要文本后写入 memoryRepo。
// 当前为 v1.1.9 最小实现：不依赖 LLM 生成摘要，不修改现有 conversation/message 表。
func (m *MemoryRetriever) ArchiveConversation(ctx context.Context, convID models.ConversationID) error {
	if m.memoryRepo == nil {
		return fmt.Errorf("memory repository not available")
	}
	if convID == "" {
		return fmt.Errorf("empty conversation id")
	}

	facts, err := m.factRepo.FindBySession(ctx, string(convID))
	if err != nil {
		return fmt.Errorf("failed to list session facts for archive: %w", err)
	}

	var approved []string
	for _, f := range facts {
		if f.Status == entity.FactStatusApproved {
			approved = append(approved, fmt.Sprintf("%s %s %s", f.Subject, f.Predicate, f.Object))
		}
	}

	var summary string
	if len(approved) == 0 {
		summary = fmt.Sprintf("会话 %s 归档：未记录到可确认的健康信息。", convID)
	} else {
		summary = fmt.Sprintf("会话 %s 归档，共 %d 条已确认健康信息：%s", convID, len(approved), strings.Join(approved, "；"))
	}

	mem := entity.NewHealthMemory(entity.TierShortTerm, summary, convID)
	mem.Confidence = 0.8
	if err := m.memoryRepo.Save(ctx, mem); err != nil {
		return fmt.Errorf("failed to save archived memory: %w", err)
	}
	return nil
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
