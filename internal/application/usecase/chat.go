// Package usecase 实现应用用例层，编排领域对象完成完整业务流程。
package usecase

import (
	"context"
	"fmt"

	"github.com/google/wire"
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

// Execute 执行单次对话用例。
func (c *ChatOrchestrator) Execute(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// TODO(作者): 实现完整流水线 [Issue#003]
	// 1. 脱敏检测 → 2. 记忆检索 → 3. LLM 调用 → 4. 合规检测 → 5. 输出还原
	return nil, fmt.Errorf("not implemented")
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
}

// DefaultComplianceChecker 是 ComplianceChecker 的默认占位实现，
// 返回放行结果，待合规引擎完善后替换。
type DefaultComplianceChecker struct{}

// NewDefaultComplianceChecker 创建默认合规检查器。
func NewDefaultComplianceChecker() *DefaultComplianceChecker {
	return &DefaultComplianceChecker{}
}

// Check 执行合规检查，当前始终放行。
func (c *DefaultComplianceChecker) Check(ctx context.Context, text string) (*ComplianceResult, error) {
	return &ComplianceResult{Blocked: false, Level: "L4", SafeText: text}, nil
}

// ApplicationSet 供 Wire 使用的 ProviderSet。
var ApplicationSet = wire.NewSet(
	NewChatOrchestrator,
	NewDefaultComplianceChecker,
)
