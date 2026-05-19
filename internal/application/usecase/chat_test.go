package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/medmemo/medmemo/internal/application"
	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustEmptyRulesPath 创建包含空规则库的临时 JSON 文件并返回路径。
func mustEmptyRulesPath(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "rules.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"version":"test","rules":[]}`), 0644))
	return path
}

// newTestComplianceChecker 从临时规则文件创建合规检查器（供测试使用）。
func newTestComplianceChecker(t *testing.T, path string) *RuleComplianceChecker {
	t.Helper()
	ci, err := application.NewComplianceInterceptor(path)
	require.NoError(t, err)
	return &RuleComplianceChecker{interceptor: ci}
}

// mockLLMClient 是 port.LLMClient 的手动 Mock 实现。
type mockLLMClient struct {
	chatReply       string
	chatErr         error
	streamChunks    []string
	streamErr       error
	streamCallbacks []string // 记录 StreamChat 收到的所有 chunk
}

func (m *mockLLMClient) Chat(ctx context.Context, messages []models.Message) (string, error) {
	return m.chatReply, m.chatErr
}

func (m *mockLLMClient) StreamChat(ctx context.Context, messages []models.Message, callback func(chunk string)) error {
	for _, chunk := range m.streamChunks {
		callback(chunk)
		m.streamCallbacks = append(m.streamCallbacks, chunk)
	}
	return m.streamErr
}

func (m *mockLLMClient) CheckAvailability(ctx context.Context) (bool, string) {
	return true, "available"
}

var _ port.LLMClient = (*mockLLMClient)(nil)

// TestRuleComplianceChecker_Check 验证空规则库下合规检查器始终放行。
func TestRuleComplianceChecker_Check(t *testing.T) {
	checker := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	ctx := context.Background()

	result, err := checker.Check(ctx, "这是一条测试内容")
	require.NoError(t, err)
	assert.False(t, result.Blocked)
	assert.Equal(t, "L4_NORMAL", result.Level)
	assert.Equal(t, "这是一条测试内容", result.SafeText)
}

// TestRuleComplianceChecker_Check_EmptyText 验证空文本不报错。
func TestRuleComplianceChecker_Check_EmptyText(t *testing.T) {
	checker := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	ctx := context.Background()

	result, err := checker.Check(ctx, "")
	require.NoError(t, err)
	assert.False(t, result.Blocked)
}

// TestChatOrchestrator_Execute_Success 验证非流式对话正常返回。
func TestChatOrchestrator_Execute_Success(t *testing.T) {
	mock := &mockLLMClient{chatReply: "你好，有什么可以帮你的？"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := NewChatOrchestrator(mock, nil, nil, comp)

	req := ChatRequest{
		ConversationID: "conv-1",
		Messages:       []models.Message{{Role: models.RoleUser, Content: "你好"}},
		Model:          models.ProviderKimi,
	}

	resp, err := orch.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "你好，有什么可以帮你的？", resp.Reply)
}

// TestChatOrchestrator_Execute_Error 验证 LLM 错误被正确包装。
func TestChatOrchestrator_Execute_Error(t *testing.T) {
	mock := &mockLLMClient{chatErr: fmt.Errorf("network timeout")}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := NewChatOrchestrator(mock, nil, nil, comp)

	req := ChatRequest{
		Messages: []models.Message{{Role: models.RoleUser, Content: "test"}},
	}

	_, err := orch.Execute(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chat execution failed")
}

// TestChatOrchestrator_StreamExecute_Success 验证流式对话 callback 被逐 chunk 调用。
func TestChatOrchestrator_StreamExecute_Success(t *testing.T) {
	mock := &mockLLMClient{streamChunks: []string{"你", "好", "！"}}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := NewChatOrchestrator(mock, nil, nil, comp)

	req := ChatRequest{
		Messages: []models.Message{{Role: models.RoleUser, Content: "打招呼"}},
	}

	var collected []string
	err := orch.StreamExecute(context.Background(), req, func(chunk string) {
		collected = append(collected, chunk)
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"你", "好", "！"}, collected)
}

// TestChatOrchestrator_StreamExecute_Error 验证流式错误被正确包装。
func TestChatOrchestrator_StreamExecute_Error(t *testing.T) {
	mock := &mockLLMClient{streamErr: fmt.Errorf("connection reset")}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := NewChatOrchestrator(mock, nil, nil, comp)

	req := ChatRequest{
		Messages: []models.Message{{Role: models.RoleUser, Content: "test"}},
	}

	err := orch.StreamExecute(context.Background(), req, func(chunk string) {})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stream execution failed")
}
