// Package usecase 应用用例层，编排领域对象完成完整业务流程。
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/wire"
	"github.com/hzhan516/medmemo/internal/application"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/hzhan516/medmemo/pkg/desensitizer"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/hzhan516/medmemo/pkg/resourcepath"
)

// Deidentifier 脱敏流水线接口。
type Deidentifier interface {
	Execute(ctx context.Context, raw string) (models.DeidentifyResult, error)
}

// ChatOrchestrator 对话流程编排器。
type ChatOrchestrator struct {
	llmFactory           port.LLMClientFactory
	providerStore        port.ProviderStore
	memoryRepo           port.MemoryRepository
	detector             port.SensitiveDetector
	compliance           ComplianceChecker
	deidPipeline         Deidentifier
	memoryRetriever      MemoryQuerier
	confidenceAggregator *ConfidenceAggregator
	factRepo             repository.FactRepository // 新增：用于本地 approved fact 直查
	intentResolver       *IntentResolver           // 新增：意图解析
	localAnswer          *LocalAnswerService       // 新增：本地模板回答
	// 全局事实提取限流器，跨调用共享速率限制
	factExtractMu       sync.Mutex
	factExtractLastCall time.Time
	factExtractMinGap   time.Duration // 最小调用间隔
}

// NewChatOrchestrator 构造函数，供 Wire 注入。
func NewChatOrchestrator(
	llmFactory port.LLMClientFactory,
	providerStore port.ProviderStore,
	mem port.MemoryRepository,
	det port.SensitiveDetector,
	comp ComplianceChecker,
	deid Deidentifier,
	retriever MemoryQuerier,
	confAgg *ConfidenceAggregator,
	factRepo repository.FactRepository,
	intentResolver *IntentResolver,
	localAnswer *LocalAnswerService,
) *ChatOrchestrator {
	return &ChatOrchestrator{
		llmFactory:           llmFactory,
		providerStore:        providerStore,
		memoryRepo:           mem,
		detector:             det,
		compliance:           comp,
		deidPipeline:         deid,
		memoryRetriever:      retriever,
		confidenceAggregator: confAgg,
		factRepo:             factRepo,
		intentResolver:       intentResolver,
		localAnswer:          localAnswer,
		factExtractMinGap:    15 * time.Second, // 最小 15 秒间隔，避免与主对话竞争
	}
}

// ChatRequest 对话请求 DTO。
type ChatRequest struct {
	ConversationID models.ConversationID
	Messages       []models.Message
	Model          models.ProviderType
	ProviderID     string
}

// ChatResponse 对话响应 DTO。
type ChatResponse struct {
	Reply            string
	ConfidenceResult *entity.ConfidenceResult
	Warnings         []string
}

// isLocalModel 判断是否为本地模型（跳过脱敏）。
func isLocalModel(pt models.ProviderType) bool {
	return pt == models.ProviderOllama || pt == models.ProviderLocal
}

// findLastUserMessage 定位最后一条用户消息，无则返回 -1。
func findLastUserMessage(msgs []models.Message) int {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == models.RoleUser {
			return i
		}
	}
	return -1
}

// injectMemories 将记忆片段注入为 system message 前缀。
func injectMemories(msgs []models.Message, memories []*entity.HealthMemory) []models.Message {
	if len(memories) == 0 {
		return msgs
	}
	var parts []string
	for _, m := range memories {
		if m.Content != "" {
			parts = append(parts, m.Content)
		}
	}
	if len(parts) == 0 {
		return msgs
	}

	memCtx := "以下是与当前话题相关的历史记忆，供你参考（不对外展示）：\n" + strings.Join(parts, "\n")

	result := make([]models.Message, 0, len(msgs)+1)
	// 首条已是 system 则追加，否则插入新 system message
	if len(msgs) > 0 && msgs[0].Role == models.RoleSystem {
		result = append(result, models.Message{
			Role:    models.RoleSystem,
			Content: memCtx + "\n\n" + msgs[0].Content,
		})
		result = append(result, msgs[1:]...)
	} else {
		result = append(result, models.Message{Role: models.RoleSystem, Content: memCtx})
		result = append(result, msgs...)
	}
	return result
}

// prepareMessages 执行输入脱敏与记忆检索，返回处理后的消息和脱敏结果。
func (c *ChatOrchestrator) prepareMessages(ctx context.Context, req ChatRequest) ([]models.Message, models.DeidentifyResult) {
	messages := req.Messages
	var deidResult models.DeidentifyResult

	// 输入脱敏（仅云端模型）
	if !isLocalModel(req.Model) && c.deidPipeline != nil {
		lastIdx := findLastUserMessage(req.Messages)
		if lastIdx >= 0 {
			deidStart := time.Now()
			r, err := c.deidPipeline.Execute(ctx, req.Messages[lastIdx].Content)
			fmt.Printf("[DIAG][Chat] deidPipeline.Execute took %v err=%v\n", time.Since(deidStart), err)
			if err == nil {
				deidResult = r
				messages = make([]models.Message, len(req.Messages))
				copy(messages, req.Messages)
				messages[lastIdx].Content = r.SafeText
			}
			// 脱敏失败降级，继续使用原始文本
		}
	}

	// 记忆检索（对脱敏后的内容检索，避免敏感信息进入 query）
	if c.memoryRetriever != nil {
		lastIdx := findLastUserMessage(messages)
		if lastIdx >= 0 {
			memStart := time.Now()
			memories, err := c.memoryRetriever.RetrieveForContext(ctx, messages[lastIdx].Content, string(req.ConversationID), 3)
			if err != nil {
				fmt.Printf("[DIAG][Chat] memoryRetriever.RetrieveForContext took %v err=%v\n", time.Since(memStart), err)
			} else {
				fmt.Printf("[DIAG][Chat] memoryRetriever.RetrieveForContext took %v memories=%d\n", time.Since(memStart), len(memories))
			}
			if len(memories) > 0 {
				messages = injectMemories(messages, memories)
			}
		}
	}

	return messages, deidResult
}

// calculateConfidenceWithRawScore 包装 ConfidenceAggregator.CalculateWithRawScore，带 nil 防护。
func (c *ChatOrchestrator) calculateConfidenceWithRawScore(score float64, sources []string) *entity.ConfidenceResult {
	if c.confidenceAggregator == nil {
		return &entity.ConfidenceResult{
			OverallScore: score,
			Level:        entity.ConfidenceLevelE,
			Breakdown:    map[string]float64{},
			Explanation:  "置信度引擎未初始化",
			Suggestion:   entity.ConfidenceLevelE.Suggestion(),
			MissingInfo:  []string{},
		}
	}
	return c.confidenceAggregator.CalculateWithRawScore(score, sources)
}

// tryLocalAnswer 尝试对高置信个人事实查询进行本地短路回答。
// 命中 approved fact 时返回 (answer, true, nil)；未命中时返回 ("", false, nil)；
// 数据库异常时返回错误（调用方应降级到 LLM 链路）。
func (c *ChatOrchestrator) tryLocalAnswer(ctx context.Context, query string) (string, bool, error) {
	result := c.intentResolver.Resolve(query)
	if result == nil || result.Confidence != ConfidenceHigh {
		return "", false, nil
	}
	// MVP 阶段 subject 固定为"用户"，后续可扩展为当前家庭成员
	fact, err := c.factRepo.FindLatestApprovedByPredicates(ctx, "用户", result.Predicates)
	if err != nil {
		if errors.Is(err, entity.ErrFactNotFound) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("本地事实查询失败: %w", err)
	}
	return c.localAnswer.Format(result.Intent, fact), true, nil
}

// Execute 执行单次对话用例（非流式）。
func (c *ChatOrchestrator) Execute(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 前置本地短路：高置信个人事实查询直接返回，不走 LLM
	lastIdx := findLastUserMessage(req.Messages)
	if lastIdx >= 0 {
		answer, ok, err := c.tryLocalAnswer(ctx, req.Messages[lastIdx].Content)
		if err != nil {
			fmt.Printf("[ChatOrchestrator] tryLocalAnswer error, fallback to LLM: %v\n", err)
		} else if ok {
			return &ChatResponse{
				Reply:            answer,
				ConfidenceResult: c.calculateConfidenceWithRawScore(1.0, []string{"本地已审批事实"}),
			}, nil
		}
	}

	messages, deidResult := c.prepareMessages(ctx, req)

	// 根据 ProviderID 动态创建 LLMClient
	llmClient, err := c.resolveLLMClient(ctx, req.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve llm client: %w", err)
	}

	// LLM 调用
	reply, err := llmClient.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("chat execution failed: %w", err)
	}

	// 输出还原（仅云端模型且有 P2 占位符时）
	if !isLocalModel(req.Model) && len(deidResult.Placeholder) > 0 {
		reply = desensitizer.Restore(models.DeidentifyResult{
			SafeText:    reply,
			Placeholder: deidResult.Placeholder,
		})
	}

	// 合规检测
	compResult, err := c.compliance.Check(ctx, reply)
	if err != nil {
		// 降级放行，确保对话不中断
		return &ChatResponse{
			Reply:            reply,
			ConfidenceResult: c.calculateConfidenceWithRawScore(0.0, []string{"合规检测异常"}),
			Warnings:         []string{"COMPLIANCE_CHECK_ERROR"},
		}, nil
	}

	warnings := make([]string, 0)
	if compResult.Level != application.L4Normal.String() {
		warnings = append(warnings, compResult.Level)
	}
	if compResult.Warning != "" {
		warnings = append(warnings, "WARNING:"+compResult.Warning)
	}
	if compResult.Notice != "" {
		warnings = append(warnings, "NOTICE:"+compResult.Notice)
	}
	if compResult.MatchedRule != "" {
		warnings = append(warnings, "RULE:"+compResult.MatchedRule)
	}

	// 计算回答置信度
	confidence := c.calculateConfidence(reply, messages)

	return &ChatResponse{
		Reply:            reply,
		ConfidenceResult: confidence,
		Warnings:         warnings,
	}, nil
}

// StreamExecute 执行流式对话用例。
// 采用逐 chunk 透传策略：LLM 每生成一个 token 立即通过 onChunk 推送到前端，
// 保持打字机效果；流结束后在后台执行输出还原与合规检测。
// 若还原或合规替换导致内容变化，返回的 finalContent 与流式过程中推送的内容不同，
// 由外层通过 chat:stream:replace 事件通知前端替换。
// 仅 L1（阻断级）返回 SafeText；L2/L3 保留原文，由外层通过
// chat:stream:compliance 事件追加标签。流式正常结束时返回 TokenUsage、置信度与最终内容。
func (c *ChatOrchestrator) StreamExecute(ctx context.Context, req ChatRequest, onChunk func(string)) (*models.TokenUsage, *entity.ConfidenceResult, string, error) {
	streamExecStart := time.Now()

	// 前置本地短路：高置信个人事实查询直接返回，不走 LLM Stream
	lastIdx := findLastUserMessage(req.Messages)
	if lastIdx >= 0 {
		answer, ok, err := c.tryLocalAnswer(ctx, req.Messages[lastIdx].Content)
		if err != nil {
			fmt.Printf("[ChatOrchestrator] tryLocalAnswer error, fallback to LLM: %v\n", err)
		} else if ok {
			onChunk(answer)
			return nil, c.calculateConfidenceWithRawScore(1.0, []string{"本地已审批事实"}), answer, nil
		}
	}

	// 预处理使用独立 context，避免 ONNX 推理消耗 stream budget
	// L1 规则引擎 <1ms，L2 NER 正常 <100ms，30s 足够覆盖异常场景
	prepCtx, prepCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer prepCancel()
	prepStart := time.Now()
	messages, deidResult := c.prepareMessages(prepCtx, req)
	fmt.Printf("[DIAG][Chat] prepareMessages took %v\n", time.Since(prepStart))

	// provider 查询使用独立 context，确保不受预处理耗时影响
	resolveCtx, resolveCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer resolveCancel()
	resolveStart := time.Now()
	llmClient, err := c.resolveLLMClient(resolveCtx, req.ProviderID)
	fmt.Printf("[DIAG][Chat] resolveLLMClient took %v err=%v\n", time.Since(resolveStart), err)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to resolve llm client: %w", err)
	}

	fmt.Printf("[DIAG][Chat] StreamExecute pre-StreamChat total=%v streamCtxErr=%v\n",
		time.Since(streamExecStart), ctx.Err())

	// 逐 chunk 透传，保持打字机流式效果
	var fullReply strings.Builder
	usage, err := llmClient.StreamChat(ctx, messages, func(chunk string) {
		fullReply.WriteString(chunk)
		onChunk(chunk)
	})
	if err != nil {
		return nil, nil, "", fmt.Errorf("stream execution failed: %w", err)
	}

	// 输出还原
	reply := fullReply.String()
	if !isLocalModel(req.Model) && len(deidResult.Placeholder) > 0 {
		reply = desensitizer.Restore(models.DeidentifyResult{
			SafeText:    reply,
			Placeholder: deidResult.Placeholder,
		})
	}

	// 合规检测（保留原始回复，不替换 SafeText；合规提示由外层通过 chat:stream:compliance 事件追加）
	compResult, compErr := c.compliance.Check(ctx, reply)
	if compErr != nil {
		// fail-closed: 合规检测异常时记录审计日志，返回安全替代文案，不阻断流
		fmt.Printf("[ChatOrchestrator] compliance check error, fail-closed: %v\n", compErr)
		reply = "内容审核服务暂时不可用，请稍后重试或咨询专业医生。"
	} else if compResult != nil && compResult.Blocked {
		// L1 阻断时替换为安全文案（流式结束后通过 replace 事件通知前端）
		reply = compResult.SafeText
	}

	// 计算回答置信度
	confidence := c.calculateConfidence(reply, messages)

	return usage, confidence, reply, nil
}

// calculateConfidence 为 AI 回复计算置信度结果。
// 当前 MVP 实现：默认来源为 llm_internal，从回复内容提取推理链，
// 上下文分数基于用户消息长度，历史准确率使用冷启动默认值 0.75。
func (c *ChatOrchestrator) calculateConfidence(reply string, messages []models.Message) *entity.ConfidenceResult {
	if c.confidenceAggregator == nil {
		// 置信度引擎未注入时返回零值结果，避免 panic
		return &entity.ConfidenceResult{
			OverallScore: 0.0,
			Level:        entity.ConfidenceLevelE,
			Breakdown:    map[string]float64{},
			Explanation:  "置信度引擎未初始化",
			Suggestion:   entity.ConfidenceLevelE.Suggestion(),
			MissingInfo:  []string{},
		}
	}

	// 默认知识来源为 llm_internal（MVP 阶段无 RAG 实际集成）
	sources := []entity.KnowledgeSource{
		c.confidenceAggregator.tagger.Tag(entity.SourceLLMInternal, "LLM 内部推理"),
	}

	// 从回复内容提取推理链
	reasoning := c.confidenceAggregator.evaluator.ExtractReasoningChain(reply)

	// 上下文分数：基于用户消息长度简单评估（0-100）
	contextScore := 50.0
	for _, m := range messages {
		if m.Role == models.RoleUser {
			length := len(m.Content)
			if length > 100 {
				contextScore = 80.0
			} else if length > 30 {
				contextScore = 60.0
			}
		}
	}

	// 历史准确率：冷启动默认值
	historyAccuracy := 0.75

	// 不确定性分数：检测回复中是否包含不确定性表达
	uncertaintyScore := 50.0
	lowerReply := strings.ToLower(reply)
	if strings.Contains(lowerReply, "不确定") || strings.Contains(lowerReply, "可能") ||
		strings.Contains(lowerReply, "建议") || strings.Contains(lowerReply, "也许") ||
		strings.Contains(lowerReply, "仅供参考") {
		uncertaintyScore = 85.0
	}

	return c.confidenceAggregator.Calculate(
		sources,
		reasoning,
		contextScore,
		historyAccuracy,
		uncertaintyScore,
	)
}

// resolveLLMClient 根据 ProviderID 从 store 查找配置并动态创建 LLMClient。
func (c *ChatOrchestrator) resolveLLMClient(ctx context.Context, providerID string) (port.LLMClient, error) {
	if providerID == "" {
		return nil, fmt.Errorf("provider_id is required")
	}

	// 诊断日志：检查传入 context 的剩余时间
	if deadline, ok := ctx.Deadline(); ok {
		fmt.Printf("[resolveLLMClient] context deadline in %v, providerID=%s\n",
			time.Until(deadline), providerID)
	} else {
		fmt.Printf("[resolveLLMClient] context has no deadline, providerID=%s\n", providerID)
	}

	provider, err := c.providerStore.Get(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider %s: %w", providerID, err)
	}

	client, err := c.llmFactory.CreateClient(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create llm client for provider %s: %w", providerID, err)
	}

	return client, nil
}

// llmClientAdapter 将 port.LLMClient 适配为 FactLLMClient。
// 用于复用已配置的 Provider 进行事实提取，避免单独维护 LLM 配置。
type llmClientAdapter struct {
	client port.LLMClient
}

func (a *llmClientAdapter) Chat(ctx context.Context, messages []string) (string, error) {
	msgs := make([]models.Message, len(messages))
	for i, m := range messages {
		msgs[i] = models.Message{Role: models.RoleUser, Content: m}
	}
	return a.client.Chat(ctx, msgs)
}

// ExtractFactsFromReply 从完整对话轮次（用户消息 + AI 回复）中提取结构化事实三元组。
// 使用当前会话的 Provider 创建 LLM client，异步调用时不阻塞主流程。
// 全局限流：确保事实提取与主对话、其他事实提取之间有足够间隔，避免触发 429。
func (c *ChatOrchestrator) ExtractFactsFromReply(ctx context.Context, userContent, aiReply, providerID string) ([]*entity.ExtractedFact, error) {
	if providerID == "" {
		return nil, nil
	}

	// 全局限流：确保事实提取与主对话、其他事实提取之间有足够间隔
	c.factExtractMu.Lock()
	elapsed := time.Since(c.factExtractLastCall)
	if elapsed < c.factExtractMinGap {
		wait := c.factExtractMinGap - elapsed
		c.factExtractMu.Unlock()
		fmt.Printf("[ExtractFacts] rate limited, waiting %v\n", wait)
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		c.factExtractMu.Lock()
	}
	c.factExtractLastCall = time.Now()
	c.factExtractMu.Unlock()

	// 只从用户消息中提取事实，避免把 AI 回复中的建议、能力限制等抽成记忆。
	if userContent == "" {
		return nil, nil
	}
	client, err := c.resolveLLMClient(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve llm client for fact extraction: %w", err)
	}
	adapter := &llmClientAdapter{client: client}
	extractor := NewFactExtractor(adapter)
	facts, err := extractor.ParseFacts(ctx, userContent)
	if err != nil {
		return nil, err
	}
	return ApplyFactQualityGate(facts), nil
}

// CheckCompliance 对文本执行合规检测。
func (c *ChatOrchestrator) CheckCompliance(ctx context.Context, text string) (*ComplianceResult, error) {
	return c.compliance.Check(ctx, text)
}

// ComplianceChecker 合规检查接口。
type ComplianceChecker interface {
	Check(ctx context.Context, text string) (*ComplianceResult, error)
}

// ComplianceResult 合规检查结果。
type ComplianceResult struct {
	Blocked       bool
	Level         string
	Reason        string
	SafeText      string
	Warning       string   // L2 警告文案
	Notice        string   // L3 提示文案
	MatchedRule   string   // 命中的规则 ID
	ReplacedTerms []string // inline 替换中被替换的用词规则 ID 列表
}

// RuleComplianceChecker 基于规则库的合规检查器实现。
type RuleComplianceChecker struct {
	interceptor *application.ComplianceInterceptor
	logger      *application.ComplianceLogger
}

// NewRuleComplianceChecker 从默认规则库路径创建合规检查器。
func NewRuleComplianceChecker() (*RuleComplianceChecker, error) {
	ci, err := application.NewComplianceInterceptor(resourcepath.Path("rules", "compliance_rules_v1.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to create compliance interceptor: %w", err)
	}
	logger := application.NewComplianceLogger("data")
	return &RuleComplianceChecker{interceptor: ci, logger: logger}, nil
}

// Check 执行合规检查，先 inline 替换再评估风险等级。
func (c *RuleComplianceChecker) Check(ctx context.Context, text string) (*ComplianceResult, error) {
	res, err := c.interceptor.EvaluateWithInlineReplace(ctx, text)
	if err != nil {
		// 检测异常时记录审计日志，避免静默吞掉错误
		if c.logger != nil {
			_ = c.logger.Log(ctx, "EVALUATE_ERROR", text, "", "ERROR")
		}
		return nil, fmt.Errorf("compliance evaluation failed: %w", err)
	}

	// 超时降级记录审计日志
	if res.TimeoutDowngrade && c.logger != nil {
		_ = c.logger.Log(ctx, "TIMEOUT_DOWNGRADE", text, res.SafeText, application.L4Normal.String())
	}

	// 命中规则时记录拦截日志
	if res.Level != application.L4Normal.String() && c.logger != nil {
		_ = c.logger.Log(ctx, res.MatchedRule, text, res.SafeText, res.Level)
	}

	return &ComplianceResult{
		Blocked:       res.Blocked,
		Level:         res.Level,
		SafeText:      res.SafeText,
		Warning:       res.Warning,
		Notice:        res.Notice,
		MatchedRule:   res.MatchedRule,
		ReplacedTerms: res.ReplacedTerms,
	}, nil
}

// ApplicationSet 供 Wire 使用的 ProviderSet。
var ApplicationSet = wire.NewSet(
	NewChatOrchestrator,
	NewRuleComplianceChecker,
	NewTitleGenerator,
	NewConfidenceAggregator,
	NewQueryExpansionService,
	NewIntentResolver,
	NewLocalAnswerService,
)
