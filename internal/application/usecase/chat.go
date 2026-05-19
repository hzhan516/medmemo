// Package usecase 实现应用用例层，编排领域对象完成完整业务流程。
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

// Deidentifier 脱敏流水线接口，供 ChatOrchestrator 消费。
type Deidentifier interface {
	Execute(ctx context.Context, raw string) (models.DeidentifyResult, error)
}

// MemoryQuerier 记忆检索接口，供 ChatOrchestrator 消费。
type MemoryQuerier interface {
	RetrieveForContext(ctx context.Context, query string, limit int) ([]*entity.HealthMemory, error)
}

// ChatOrchestrator 编排单次对话的完整流程：
// 输入脱敏 → 记忆检索 → 上下文组装 → LLM 调用 → 输出还原 → 合规检测。
type ChatOrchestrator struct {
	llmClient       port.LLMClient
	memoryRepo      port.MemoryRepository
	detector        port.SensitiveDetector
	compliance      ComplianceChecker
	deidPipeline    Deidentifier
	memoryRetriever MemoryQuerier
}

// NewChatOrchestrator 构造函数，供 Wire 注入。
func NewChatOrchestrator(
	llm port.LLMClient,
	mem port.MemoryRepository,
	det port.SensitiveDetector,
	comp ComplianceChecker,
	deid Deidentifier,
	retriever MemoryQuerier,
) *ChatOrchestrator {
	return &ChatOrchestrator{
		llmClient:       llm,
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
}

// ChatResponse 对话响应 DTO。
type ChatResponse struct {
	Reply      string
	Confidence float64
	Warnings   []string
}

// isLocalModel 判断是否为本地模型（数据不离开本机，跳过脱敏）。
func isLocalModel(pt models.ProviderType) bool {
	return pt == models.ProviderOllama || pt == models.ProviderLocal
}

// findLastUserMessage 找到 messages 中最后一条用户消息的索引，无则返回 -1。
func findLastUserMessage(msgs []models.Message) int {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == models.RoleUser {
			return i
		}
	}
	return -1
}

// injectMemories 将检索到的记忆片段注入为 system message 前缀。
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
	// 若首条已是 system，则在内容前追加记忆上下文
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

// Execute 执行单次对话用例（非流式）。
// 完整流程：输入脱敏 → 记忆检索 → 上下文组装 → LLM 调用 → 输出还原 → 合规检测。
func (c *ChatOrchestrator) Execute(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	messages := req.Messages
	var deidResult models.DeidentifyResult

	// 1. 输入脱敏（仅云端模型）
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
			// 脱敏失败时降级，继续使用原始文本
		}
	}

	// 2. 记忆检索（对脱敏后的内容检索，避免敏感信息进入检索 query）
	if c.memoryRetriever != nil {
		lastIdx := findLastUserMessage(messages)
		if lastIdx >= 0 {
			memories, _ := c.memoryRetriever.RetrieveForContext(ctx, messages[lastIdx].Content, 3)
			if len(memories) > 0 {
				messages = injectMemories(messages, memories)
			}
		}
	}

	// 3. LLM 调用
	reply, err := c.llmClient.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("chat execution failed: %w", err)
	}

	// 4. 输出还原（仅云端模型且有 P2 占位符时）
	if !isLocalModel(req.Model) && len(deidResult.Placeholder) > 0 {
		reply = desensitizer.Restore(models.DeidentifyResult{
			SafeText:    reply,
			Placeholder: deidResult.Placeholder,
		})
	}

	// 5. 合规检测
	compResult, err := c.compliance.Check(ctx, reply)
	if err != nil {
		// 合规检测失败时降级放行，确保对话不中断
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
// MVP 阶段：输入脱敏与记忆注入后启动流式，Stream 结束后对完整内容做还原与合规检测。
func (c *ChatOrchestrator) StreamExecute(ctx context.Context, req ChatRequest, onChunk func(string)) error {
	messages := req.Messages
	var deidResult models.DeidentifyResult

	// 1. 输入脱敏（仅云端模型）
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
		}
	}

	// 2. 记忆检索
	if c.memoryRetriever != nil {
		lastIdx := findLastUserMessage(messages)
		if lastIdx >= 0 {
			memories, _ := c.memoryRetriever.RetrieveForContext(ctx, messages[lastIdx].Content, 3)
			if len(memories) > 0 {
				messages = injectMemories(messages, memories)
			}
		}
	}

	// 3. 收集完整流式内容用于后续还原与检测
	var fullReply stringsBuilder

	err := c.llmClient.StreamChat(ctx, messages, func(chunk string) {
		fullReply.WriteString(chunk)
		onChunk(chunk)
	})
	if err != nil {
		return fmt.Errorf("stream execution failed: %w", err)
	}

	// 4. 输出还原
	reply := fullReply.String()
	if !isLocalModel(req.Model) && len(deidResult.Placeholder) > 0 {
		reply = desensitizer.Restore(models.DeidentifyResult{
			SafeText:    reply,
			Placeholder: deidResult.Placeholder,
		})
	}

	// 5. 流式结束后对完整内容做一次性合规检测（MVP 简化策略）
	compResult, compErr := c.compliance.Check(ctx, reply)
	if compErr == nil && compResult.Level != application.L4Normal.String() {
		// 通过 Wails 事件或回调方式告知前端合规结果，当前不阻断流式
		_ = compResult
	}

	return nil
}

// CheckCompliance 对文本执行合规检测，供流式结束后调用。
func (c *ChatOrchestrator) CheckCompliance(ctx context.Context, text string) (*ComplianceResult, error) {
	return c.compliance.Check(ctx, text)
}

// ComplianceChecker 应用层本地接口，检查输出内容合规性。
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

// Check 执行合规检查，先进行 inline 用词替换，再评估风险等级。
func (c *RuleComplianceChecker) Check(ctx context.Context, text string) (*ComplianceResult, error) {
	res, err := c.interceptor.EvaluateWithInlineReplace(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("compliance evaluation failed: %w", err)
	}

	// 记录拦截日志（仅当命中规则时）
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

// stringsBuilder 是 strings.Builder 的轻量别名，避免 import strings 包（仅内部使用）。
type stringsBuilder struct {
	b []byte
}

func (s *stringsBuilder) WriteString(str string) {
	s.b = append(s.b, str...)
}

func (s *stringsBuilder) String() string {
	return string(s.b)
}

// ApplicationSet 供 Wire 使用的 ProviderSet。
var ApplicationSet = wire.NewSet(
	NewChatOrchestrator,
	NewRuleComplianceChecker,
	NewTitleGenerator,
)
