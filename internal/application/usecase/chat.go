// Package usecase 实现应用用例层，编排领域对象完成完整业务流程。
package usecase

import (
	"context"
	"fmt"

	"github.com/google/wire"
	"github.com/medmemo/medmemo/internal/application"
	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/pkg/models"
)

// ChatOrchestrator 编排单次对话的完整流程：
// 输入脱敏 → 记忆检索 → 上下文组装 → LLM 调用 → 合规检测 → 输出还原。
type ChatOrchestrator struct {
	llmClient  port.LLMClient
	memoryRepo port.MemoryRepository
	detector   port.SensitiveDetector
	compliance ComplianceChecker // 本地接口，见下方
}

// NewChatOrchestrator 构造函数，供 Wire 注入。
func NewChatOrchestrator(
	llm port.LLMClient,
	mem port.MemoryRepository,
	det port.SensitiveDetector,
	comp ComplianceChecker,
) *ChatOrchestrator {
	return &ChatOrchestrator{
		llmClient:  llm,
		memoryRepo: mem,
		detector:   det,
		compliance: comp,
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

// Execute 执行单次对话用例（非流式）。
// 流程：LLM 调用 → 合规检测 → 组装响应。
func (c *ChatOrchestrator) Execute(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	reply, err := c.llmClient.Chat(ctx, req.Messages)
	if err != nil {
		return nil, fmt.Errorf("chat execution failed: %w", err)
	}

	// 合规检测
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
// MVP 阶段保持透传架构，Stream 结束后由调用方对完整内容做一次性合规检测。
func (c *ChatOrchestrator) StreamExecute(ctx context.Context, req ChatRequest, onChunk func(string)) error {
	if err := c.llmClient.StreamChat(ctx, req.Messages, onChunk); err != nil {
		return fmt.Errorf("stream execution failed: %w", err)
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
	Blocked  bool
	Level    string
	Reason   string
	SafeText string
	Warning  string // L2 警告文案
	Notice   string // L3 提示文案
}

// RuleComplianceChecker 基于规则库的合规检查器实现。
type RuleComplianceChecker struct {
	interceptor *application.ComplianceInterceptor
}

// NewRuleComplianceChecker 从默认规则库路径创建合规检查器。
func NewRuleComplianceChecker() (*RuleComplianceChecker, error) {
	ci, err := application.NewComplianceInterceptor("resources/rules/compliance_rules_v1.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create compliance interceptor: %w", err)
	}
	return &RuleComplianceChecker{interceptor: ci}, nil
}

// Check 执行合规检查，调用拦截引擎评估文本风险等级。
func (c *RuleComplianceChecker) Check(ctx context.Context, text string) (*ComplianceResult, error) {
	res, err := c.interceptor.Evaluate(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("compliance evaluation failed: %w", err)
	}
	return &ComplianceResult{
		Blocked:  res.Blocked,
		Level:    res.Level,
		SafeText: res.SafeText,
		Warning:  res.Warning,
		Notice:   res.Notice,
	}, nil
}

// ApplicationSet 供 Wire 使用的 ProviderSet。
var ApplicationSet = wire.NewSet(
	NewChatOrchestrator,
	NewRuleComplianceChecker,
	NewTitleGenerator,
)
