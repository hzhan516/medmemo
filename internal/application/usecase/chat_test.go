package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hzhan516/medmemo/internal/application"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
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
	lastMessages    []models.Message
}

func (m *mockLLMClient) Chat(ctx context.Context, messages []models.Message) (string, error) {
	m.lastMessages = messages
	return m.chatReply, m.chatErr
}

func (m *mockLLMClient) StreamChat(ctx context.Context, messages []models.Message, callback func(chunk string)) (*models.TokenUsage, error) {
	m.lastMessages = messages
	for _, chunk := range m.streamChunks {
		callback(chunk)
		m.streamCallbacks = append(m.streamCallbacks, chunk)
	}
	return nil, m.streamErr
}

func (m *mockLLMClient) CheckAvailability(ctx context.Context) (bool, string) {
	return true, "available"
}

var _ port.LLMClient = (*mockLLMClient)(nil)

// mockLLMClientFactory 实现 port.LLMClientFactory。
type mockLLMClientFactory struct {
	client port.LLMClient
}

func (m *mockLLMClientFactory) CreateClient(providerConfig *models.ProviderConfig) (port.LLMClient, error) {
	return m.client, nil
}

var _ port.LLMClientFactory = (*mockLLMClientFactory)(nil)

// mockProviderStore 实现 port.ProviderStore。
type mockProviderStore struct {
	provider *models.ProviderConfig
}

func (m *mockProviderStore) Create(ctx context.Context, provider *models.ProviderConfig) error {
	return nil
}
func (m *mockProviderStore) Update(ctx context.Context, provider *models.ProviderConfig) error {
	return nil
}
func (m *mockProviderStore) Delete(ctx context.Context, id string) error {
	return nil
}
func (m *mockProviderStore) Get(ctx context.Context, id string) (*models.ProviderConfig, error) {
	if m.provider == nil {
		return &models.ProviderConfig{ID: id, APIHost: "https://api.example.com", ModelID: "test-model"}, nil
	}
	return m.provider, nil
}
func (m *mockProviderStore) List(ctx context.Context) ([]*models.ProviderConfig, error) {
	return []*models.ProviderConfig{}, nil
}

var _ port.ProviderStore = (*mockProviderStore)(nil)

// mockDeidentifier 实现 Deidentifier 接口。
type mockDeidentifier struct {
	result models.DeidentifyResult
	err    error
}

func (m *mockDeidentifier) Execute(ctx context.Context, raw string) (models.DeidentifyResult, error) {
	return m.result, m.err
}

var _ Deidentifier = (*mockDeidentifier)(nil)

// mockMemoryQuerier 实现 MemoryQuerier 接口。
type mockMemoryQuerier struct {
	memories []*entity.HealthMemory
	err      error
}

func (m *mockMemoryQuerier) RetrieveForContext(ctx context.Context, query string, limit int) ([]*entity.HealthMemory, error) {
	return m.memories, m.err
}

var _ MemoryQuerier = (*mockMemoryQuerier)(nil)

// newTestOrchestrator 创建带有全量 Mock 依赖的 ChatOrchestrator。
func newTestOrchestrator(mock *mockLLMClient, comp *RuleComplianceChecker, deid Deidentifier, retriever MemoryQuerier) *ChatOrchestrator {
	factory := &mockLLMClientFactory{client: mock}
	store := &mockProviderStore{}
	return NewChatOrchestrator(factory, store, nil, nil, comp, deid, retriever)
}

// TestChatOrchestrator_Execute_Success 验证非流式对话正常返回。
func TestChatOrchestrator_Execute_Success(t *testing.T) {
	mock := &mockLLMClient{chatReply: "你好，有什么可以帮你的？"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := newTestOrchestrator(mock, comp, nil, nil)

	req := ChatRequest{
		ConversationID: "conv-1",
		Messages:       []models.Message{{Role: models.RoleUser, Content: "你好"}},
		Model:          models.ProviderKimi,
		ProviderID:     "test-provider",
	}

	resp, err := orch.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "你好，有什么可以帮你的？", resp.Reply)
}

// TestChatOrchestrator_Execute_Error 验证 LLM 错误被正确包装。
func TestChatOrchestrator_Execute_Error(t *testing.T) {
	mock := &mockLLMClient{chatErr: fmt.Errorf("network timeout")}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := newTestOrchestrator(mock, comp, nil, nil)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "test"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	_, err := orch.Execute(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chat execution failed")
}

// TestChatOrchestrator_Execute_WithDeidentify 验证输入脱敏后被注入 LLM 调用。
func TestChatOrchestrator_Execute_WithDeidentify(t *testing.T) {
	mock := &mockLLMClient{chatReply: "收到"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	deid := &mockDeidentifier{
		result: models.DeidentifyResult{
			OriginalText: "联系我 test@example.com",
			SafeText:     "联系我 {{EMAIL_abc12345}}",
			Placeholder:  map[string]string{"{{EMAIL_abc12345}}": "test@example.com"},
		},
	}
	orch := newTestOrchestrator(mock, comp, deid, nil)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "联系我 test@example.com"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	resp, err := orch.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "收到", resp.Reply)
	// 验证 LLM 收到的是脱敏后的文本
	require.Len(t, mock.lastMessages, 1)
	assert.Equal(t, "联系我 {{EMAIL_abc12345}}", mock.lastMessages[0].Content)
}

// TestChatOrchestrator_Execute_DeidentifyFallback 验证脱敏失败时降级透传原始文本。
func TestChatOrchestrator_Execute_DeidentifyFallback(t *testing.T) {
	mock := &mockLLMClient{chatReply: "明白"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	deid := &mockDeidentifier{err: fmt.Errorf("pipeline unavailable")}
	orch := newTestOrchestrator(mock, comp, deid, nil)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "原始内容"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	resp, err := orch.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "明白", resp.Reply)
	// 验证降级后 LLM 收到的是原始文本
	require.Len(t, mock.lastMessages, 1)
	assert.Equal(t, "原始内容", mock.lastMessages[0].Content)
}

// TestChatOrchestrator_Execute_WithMemory 验证检索到的记忆被注入上下文。
func TestChatOrchestrator_Execute_WithMemory(t *testing.T) {
	mock := &mockLLMClient{chatReply: "根据您的记忆"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	retriever := &mockMemoryQuerier{
		memories: []*entity.HealthMemory{
			{Content: "用户此前提到有高血压病史"},
		},
	}
	orch := newTestOrchestrator(mock, comp, nil, retriever)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "最近头晕"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	resp, err := orch.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "根据您的记忆", resp.Reply)
	// 验证 system message 被注入
	require.Len(t, mock.lastMessages, 2)
	assert.Equal(t, models.RoleSystem, mock.lastMessages[0].Role)
	assert.Contains(t, mock.lastMessages[0].Content, "高血压病史")
	assert.Equal(t, models.RoleUser, mock.lastMessages[1].Role)
}

// TestChatOrchestrator_Execute_Restore 验证云端响应中 P2 占位符被还原。
func TestChatOrchestrator_Execute_Restore(t *testing.T) {
	mock := &mockLLMClient{chatReply: "已发送至 {{EMAIL_abc12345}}"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	deid := &mockDeidentifier{
		result: models.DeidentifyResult{
			OriginalText: "联系我 test@example.com",
			SafeText:     "联系我 {{EMAIL_abc12345}}",
			Placeholder:  map[string]string{"{{EMAIL_abc12345}}": "test@example.com"},
		},
	}
	orch := newTestOrchestrator(mock, comp, deid, nil)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "联系我 test@example.com"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	resp, err := orch.Execute(context.Background(), req)
	require.NoError(t, err)
	// 占位符应被还原为原始值
	assert.Equal(t, "已发送至 test@example.com", resp.Reply)
}

// TestChatOrchestrator_Execute_LocalModelSkipDeid 验证本地模型跳过脱敏。
func TestChatOrchestrator_Execute_LocalModelSkipDeid(t *testing.T) {
	mock := &mockLLMClient{chatReply: "本地回复"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	deid := &mockDeidentifier{
		result: models.DeidentifyResult{
			OriginalText: "联系我 test@example.com",
			SafeText:     "联系我 {{EMAIL_abc12345}}",
			Placeholder:  map[string]string{"{{EMAIL_abc12345}}": "test@example.com"},
		},
	}
	orch := newTestOrchestrator(mock, comp, deid, nil)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "联系我 test@example.com"}},
		Model:      models.ProviderOllama,
		ProviderID: "test-provider",
	}

	resp, err := orch.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "本地回复", resp.Reply)
	// 本地模型应收到原始文本，不经过脱敏
	require.Len(t, mock.lastMessages, 1)
	assert.Equal(t, "联系我 test@example.com", mock.lastMessages[0].Content)
}

// TestChatOrchestrator_StreamExecute_Success 验证流式对话内容经合规检测后统一推送。
func TestChatOrchestrator_StreamExecute_Success(t *testing.T) {
	mock := &mockLLMClient{streamChunks: []string{"你", "好", "！"}}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := newTestOrchestrator(mock, comp, nil, nil)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "打招呼"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	var collected []string
	_, finalContent, err := orch.StreamExecute(context.Background(), req, func(chunk string) {
		collected = append(collected, chunk)
	})
	require.NoError(t, err)
	// 逐 chunk 透传，最终内容经检测后返回
	assert.Equal(t, []string{"你", "好", "！"}, collected)
	assert.Equal(t, "你好！", finalContent)
}

// TestChatOrchestrator_StreamExecute_Error 验证流式错误被正确包装。
func TestChatOrchestrator_StreamExecute_Error(t *testing.T) {
	mock := &mockLLMClient{streamErr: fmt.Errorf("connection reset")}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := newTestOrchestrator(mock, comp, nil, nil)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "test"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	_, _, err := orch.StreamExecute(context.Background(), req, func(chunk string) {})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stream execution failed")
}

// TestChatOrchestrator_StreamExecute_WithDeidentify 验证流式场景下输入脱敏生效。
func TestChatOrchestrator_StreamExecute_WithDeidentify(t *testing.T) {
	mock := &mockLLMClient{streamChunks: []string{"收", "到"}}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	deid := &mockDeidentifier{
		result: models.DeidentifyResult{
			OriginalText: "邮箱 test@example.com",
			SafeText:     "邮箱 {{EMAIL_abc12345}}",
			Placeholder:  map[string]string{"{{EMAIL_abc12345}}": "test@example.com"},
		},
	}
	orch := newTestOrchestrator(mock, comp, deid, nil)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "邮箱 test@example.com"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	var full string
	_, finalContent, err := orch.StreamExecute(context.Background(), req, func(chunk string) {
		full += chunk
	})
	require.NoError(t, err)
	// LLM 应收到脱敏后的文本
	assert.Equal(t, "收到", finalContent)
	require.Len(t, mock.lastMessages, 1)
	assert.Equal(t, "邮箱 {{EMAIL_abc12345}}", mock.lastMessages[0].Content)
}

// TestFindLastUserMessage 验证查找最后一条用户消息索引。
func TestFindLastUserMessage(t *testing.T) {
	assert.Equal(t, 0, findLastUserMessage([]models.Message{
		{Role: models.RoleUser, Content: "a"},
	}))
	assert.Equal(t, 2, findLastUserMessage([]models.Message{
		{Role: models.RoleSystem, Content: "sys"},
		{Role: models.RoleAssistant, Content: "hi"},
		{Role: models.RoleUser, Content: "a"},
	}))
	assert.Equal(t, -1, findLastUserMessage([]models.Message{
		{Role: models.RoleSystem, Content: "sys"},
	}))
	assert.Equal(t, -1, findLastUserMessage([]models.Message{}))
}

// TestInjectMemories 验证记忆注入逻辑。
func TestInjectMemories(t *testing.T) {
	// 无 system message 时插入 system
	msgs := []models.Message{
		{Role: models.RoleUser, Content: "hello"},
	}
	memories := []*entity.HealthMemory{{Content: "记忆1"}}
	result := injectMemories(msgs, memories)
	require.Len(t, result, 2)
	assert.Equal(t, models.RoleSystem, result[0].Role)
	assert.Contains(t, result[0].Content, "记忆1")
	assert.Equal(t, models.RoleUser, result[1].Role)

	// 有 system message 时追加到现有 system
	msgs = []models.Message{
		{Role: models.RoleSystem, Content: "你是医生"},
		{Role: models.RoleUser, Content: "hello"},
	}
	result = injectMemories(msgs, memories)
	require.Len(t, result, 2)
	assert.Equal(t, models.RoleSystem, result[0].Role)
	assert.Contains(t, result[0].Content, "记忆1")
	assert.Contains(t, result[0].Content, "你是医生")

	// 空记忆时不改变
	result = injectMemories(msgs, nil)
	require.Len(t, result, 2)
	assert.Equal(t, "你是医生", result[0].Content)
}

// TestIsLocalModel 验证本地模型判断。
func TestIsLocalModel(t *testing.T) {
	assert.True(t, isLocalModel(models.ProviderOllama))
	assert.True(t, isLocalModel(models.ProviderLocal))
	assert.False(t, isLocalModel(models.ProviderKimi))
	assert.False(t, isLocalModel(models.ProviderOpenAI))
	assert.False(t, isLocalModel(models.ProviderQwen))
	assert.False(t, isLocalModel(models.ProviderSiliconFlow))
}
