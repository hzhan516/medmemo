// Package usecase 应用用例层，编排领域对象完成完整业务流程。
package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/wire"
	"github.com/medmemo/medmemo/internal/application"
	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/pkg/desensitizer"
	"github.com/medmemo/medmemo/pkg/models"
)

// Deidentifier 脱敏流水线接口。
type Deidentifier interface {
	Execute(ctx context.Context, raw string) (models.DeidentifyResult, error)
}

// MemoryQuerier 记忆检索接口。
type MemoryQuerier interface {
	RetrieveForContext(ctx context.Context, query string, limit int) ([]*entity.HealthMemory, error)
}

// ChatOrchestrator 对话流程编排器。
type ChatOrchestrator struct {
	llmFactory      port.LLMClientFactory
	providerStore   port.ProviderStore
	memoryRepo      port.MemoryRepository
	detector        port.SensitiveDetector
	compliance      ComplianceChecker
	deidPipeline    Deidentifier
	memoryRetriever MemoryQuerier
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
) *ChatOrchestrator {
	return &ChatOrchestrator{
		llmFactory:      llmFactory,
		providerStore:   providerStore,
		memoryRepo:      mem,
		detector:        det,
		compliance:      comp,
		deidPipeline:    deid,
		memoryRetriever: retriever,
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
	Reply      string
	Confidence float64
	Warnings   []string
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
			r, err := c.deidPipeline.Execute(ctx, req.Messages[lastIdx].Content)
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
			memories, _ := c.memoryRetriever.RetrieveForContext(ctx, messages[lastIdx].Content, 3)
			if len(memories) > 0 {
				messages = injectMemories(messages, memories)
			}
		}
	}

	return messages, deidResult
}

// Execute 执行单次对话用例（非流式）。
func (c *ChatOrchestrator) Execute(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
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
			Reply:      reply,
			Confidence: 0.0,
			Warnings:   []string{"COMPLIANCE_CHECK_ERROR"},
		}, nil
	}

	warnings := make([]string, 0)
	if compResult.Level != application.L4Normal.String() {
		warnings = append(warnings, compResult.Level)
	}

	return &ChatResponse{
		Reply:      compResult.SafeText,
		Confidence: 0.0,
		Warnings:   warnings,
	}, nil
}

// StreamExecute 执行流式对话用例。
// 先完整收集流式内容，经还原与合规检测通过后再统一推送。
// 若命中非 L4 级别则返回 error，由上层拦截替换。
// 流式正常结束时返回 TokenUsage（若 Provider 未返回 usage 则为 nil）。
func (c *ChatOrchestrator) StreamExecute(ctx context.Context, req ChatRequest, onChunk func(string)) (*models.TokenUsage, error) {
	messages, deidResult := c.prepareMessages(ctx, req)

	// 根据 ProviderID 动态创建 LLMClient
	llmClient, err := c.resolveLLMClient(ctx, req.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve llm client: %w", err)
	}

	// 完整收集流式内容（检测通过后再推送，避免用户看到不合规内容）
	var fullReply strings.Builder
	usage, err := llmClient.StreamChat(ctx, messages, func(chunk string) {
		fullReply.WriteString(chunk)
	})
	if err != nil {
		return nil, fmt.Errorf("stream execution failed: %w", err)
	}

	// 输出还原
	reply := fullReply.String()
	if !isLocalModel(req.Model) && len(deidResult.Placeholder) > 0 {
		reply = desensitizer.Restore(models.DeidentifyResult{
			SafeText:    reply,
			Placeholder: deidResult.Placeholder,
		})
	}

	// 合规检测（检测通过后才向用户展示）
	compResult, compErr := c.compliance.Check(ctx, reply)
	if compErr != nil {
		return nil, fmt.Errorf("compliance check error: %w", compErr)
	}
	if compResult.Level != application.L4Normal.String() {
		return nil, fmt.Errorf("compliance check failed: level=%s, rule=%s", compResult.Level, compResult.MatchedRule)
	}

	// 检测通过，统一推送
	onChunk(reply)
	return usage, nil
}

// resolveLLMClient 根据 ProviderID 从 store 查找配置并动态创建 LLMClient。
func (c *ChatOrchestrator) resolveLLMClient(ctx context.Context, providerID string) (port.LLMClient, error) {
	if providerID == "" {
		return nil, fmt.Errorf("provider_id is required")
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
	ci, err := application.NewComplianceInterceptor("resources/rules/compliance_rules_v1.json")
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
)
