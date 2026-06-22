package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/application"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
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

func (m *mockMemoryQuerier) RetrieveForContext(ctx context.Context, query, sessionID string, limit int) ([]*entity.HealthMemory, error) {
	return m.memories, m.err
}

var _ MemoryQuerier = (*mockMemoryQuerier)(nil)

// mockFactRepository 实现 repository.FactRepository 接口的手动 Mock。
type mockFactRepository struct {
	facts       map[string]*entity.ExtractedFact
	byPredicate map[string][]*entity.ExtractedFact
	err         error
}

func (m *mockFactRepository) Save(ctx context.Context, f *entity.ExtractedFact) error { return nil }
func (m *mockFactRepository) GetByID(ctx context.Context, factID string) (*entity.ExtractedFact, error) {
	f, ok := m.facts[factID]
	if !ok {
		return nil, entity.ErrFactNotFound
	}
	return f, nil
}
func (m *mockFactRepository) FindByIDs(ctx context.Context, factIDs []string) (map[string]*entity.ExtractedFact, error) {
	result := make(map[string]*entity.ExtractedFact, len(factIDs))
	for _, id := range factIDs {
		if f, ok := m.facts[id]; ok {
			result[id] = f
		}
	}
	return result, nil
}
func (m *mockFactRepository) ListByStatus(ctx context.Context, status entity.FactStatus, offset, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepository) ListPending(ctx context.Context, offset, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepository) UpdateStatus(ctx context.Context, factID string, status entity.FactStatus) error {
	return nil
}
func (m *mockFactRepository) Delete(ctx context.Context, factID string) error { return nil }
func (m *mockFactRepository) GetStats(ctx context.Context) (total, approved, rejected, pending int64, err error) {
	return 0, 0, 0, 0, nil
}
func (m *mockFactRepository) ListAllSubjects(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (m *mockFactRepository) FindBySubject(ctx context.Context, subject string) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepository) FindBySession(ctx context.Context, sessionID string) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepository) FindApprovedByPredicates(ctx context.Context, subject string, predicates []string, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepository) FindLatestApprovedByPredicates(ctx context.Context, subject string, predicates []string) (*entity.ExtractedFact, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, p := range predicates {
		if facts, ok := m.byPredicate[p]; ok && len(facts) > 0 {
			return facts[0], nil
		}
	}
	return nil, entity.ErrFactNotFound
}

func (m *mockFactRepository) CountApprovedFactsNeedingEmbedding(ctx context.Context, targetVersion string) (int64, error) {
	return 0, nil
}

func (m *mockFactRepository) ListApprovedFactsNeedingEmbedding(ctx context.Context, targetVersion string, lastCreatedAt time.Time, lastFactID string, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}

var _ repository.FactRepository = (*mockFactRepository)(nil)

// newTestOrchestrator 创建带有全量 Mock 依赖的 ChatOrchestrator。
func newTestOrchestrator(mock port.LLMClient, comp *RuleComplianceChecker, deid Deidentifier, retriever MemoryQuerier, factRepo repository.FactRepository) *ChatOrchestrator {
	factory := &mockLLMClientFactory{client: mock}
	store := &mockProviderStore{}
	if factRepo == nil {
		factRepo = &mockFactRepository{}
	}
	return NewChatOrchestrator(factory, store, nil, nil, comp, deid, retriever, NewConfidenceAggregator(), factRepo, NewIntentResolver(NewQueryExpansionService()), NewLocalAnswerService())
}

// mockComplianceCheckerForTest 实现 ComplianceChecker，支持测试注入各类合规场景。
type mockComplianceCheckerForTest struct {
	result *ComplianceResult
	err    error
}

func (m *mockComplianceCheckerForTest) Check(_ context.Context, _ string) (*ComplianceResult, error) {
	return m.result, m.err
}

var _ ComplianceChecker = (*mockComplianceCheckerForTest)(nil)

// newTestOrchestratorWithChecker 创建使用指定 ComplianceChecker 的测试编排器。
func newTestOrchestratorWithChecker(mock port.LLMClient, comp ComplianceChecker) *ChatOrchestrator {
	factory := &mockLLMClientFactory{client: mock}
	store := &mockProviderStore{}
	return NewChatOrchestrator(factory, store, nil, nil, comp, nil, nil, NewConfidenceAggregator(), &mockFactRepository{}, NewIntentResolver(NewQueryExpansionService()), NewLocalAnswerService())
}

// TestChatOrchestrator_Execute_Success 验证非流式对话正常返回。
func TestChatOrchestrator_Execute_Success(t *testing.T) {
	t.Parallel()
	mock := &mockLLMClient{chatReply: "你好，有什么可以帮你的？"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := newTestOrchestrator(mock, comp, nil, nil, nil)

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
	t.Parallel()
	mock := &mockLLMClient{chatErr: fmt.Errorf("network timeout")}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := newTestOrchestrator(mock, comp, nil, nil, nil)

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
	t.Parallel()
	mock := &mockLLMClient{chatReply: "收到"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	deid := &mockDeidentifier{
		result: models.DeidentifyResult{
			OriginalText: "联系我 test@example.com",
			SafeText:     "联系我 {{EMAIL_abc12345}}",
			Placeholder:  map[string]string{"{{EMAIL_abc12345}}": "test@example.com"},
		},
	}
	orch := newTestOrchestrator(mock, comp, deid, nil, nil)

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
	t.Parallel()
	mock := &mockLLMClient{chatReply: "明白"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	deid := &mockDeidentifier{err: fmt.Errorf("pipeline unavailable")}
	orch := newTestOrchestrator(mock, comp, deid, nil, nil)

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
	t.Parallel()
	mock := &mockLLMClient{chatReply: "根据您的记忆"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	retriever := &mockMemoryQuerier{
		memories: []*entity.HealthMemory{
			{Content: "用户此前提到有高血压病史"},
		},
	}
	orch := newTestOrchestrator(mock, comp, nil, retriever, nil)

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
	t.Parallel()
	mock := &mockLLMClient{chatReply: "已发送至 {{EMAIL_abc12345}}"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	deid := &mockDeidentifier{
		result: models.DeidentifyResult{
			OriginalText: "联系我 test@example.com",
			SafeText:     "联系我 {{EMAIL_abc12345}}",
			Placeholder:  map[string]string{"{{EMAIL_abc12345}}": "test@example.com"},
		},
	}
	orch := newTestOrchestrator(mock, comp, deid, nil, nil)

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
	t.Parallel()
	mock := &mockLLMClient{chatReply: "本地回复"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	deid := &mockDeidentifier{
		result: models.DeidentifyResult{
			OriginalText: "联系我 test@example.com",
			SafeText:     "联系我 {{EMAIL_abc12345}}",
			Placeholder:  map[string]string{"{{EMAIL_abc12345}}": "test@example.com"},
		},
	}
	orch := newTestOrchestrator(mock, comp, deid, nil, nil)

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

// TestChatOrchestrator_Execute_LocalAnswer_Weight 验证高置信体重查询本地短路回答。
func TestChatOrchestrator_Execute_LocalAnswer_Weight(t *testing.T) {
	t.Parallel()
	mock := &mockLLMClient{chatReply: "不应该调用我"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	factRepo := &mockFactRepository{
		byPredicate: map[string][]*entity.ExtractedFact{
			"体重是": {
				{FactID: "f1", Subject: "用户", Predicate: "体重是", Object: "110公斤", Status: entity.FactStatusApproved, Confidence: 0.95},
			},
		},
	}
	orch := newTestOrchestrator(mock, comp, nil, nil, factRepo)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "我现在多重"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	resp, err := orch.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "记录中显示，你当前体重为 110公斤。", resp.Reply)
	// LLM 不应被调用
	assert.Empty(t, mock.lastMessages)
}

// countingMockLLMClient 是一个统计调用次数的 mock LLM 客户端。
type countingMockLLMClient struct {
	mockLLMClient
	chatCount int
}

func (m *countingMockLLMClient) Chat(ctx context.Context, messages []models.Message) (string, error) {
	m.chatCount++
	m.lastMessages = messages
	return m.chatReply, m.chatErr
}

// TestChatOrchestrator_Execute_LocalAnswer_NoCloudCall 验证命中本地事实时云端调用次数为 0。
func TestChatOrchestrator_Execute_LocalAnswer_NoCloudCall(t *testing.T) {
	t.Parallel()
	mock := &countingMockLLMClient{mockLLMClient: mockLLMClient{chatReply: "不应该调用我"}}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	factRepo := &mockFactRepository{
		byPredicate: map[string][]*entity.ExtractedFact{
			"体重是": {
				{FactID: "f1", Subject: "用户", Predicate: "体重是", Object: "110公斤", Status: entity.FactStatusApproved, Confidence: 0.95},
			},
		},
	}
	orch := newTestOrchestrator(mock, comp, nil, nil, factRepo)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "我多少斤"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	_, err := orch.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 0, mock.chatCount, "命中 approved fact 时不应调用云端 LLM")
}

// TestChatOrchestrator_Execute_LocalAnswer_NotFound 验证无 approved fact 时降级到 LLM。
func TestChatOrchestrator_Execute_LocalAnswer_NotFound(t *testing.T) {
	t.Parallel()
	mock := &mockLLMClient{chatReply: "未找到记录"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := newTestOrchestrator(mock, comp, nil, nil, &mockFactRepository{})

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "我现在多重"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	resp, err := orch.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "未找到记录", resp.Reply)
	// 降级到 LLM，应被调用
	require.NotEmpty(t, mock.lastMessages)
}

// TestChatOrchestrator_Execute_LocalAnswer_DBError 验证数据库错误时降级到 LLM 且不 panic。
func TestChatOrchestrator_Execute_LocalAnswer_DBError(t *testing.T) {
	t.Parallel()
	mock := &mockLLMClient{chatReply: "服务正常"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	factRepo := &mockFactRepository{err: fmt.Errorf("db connection lost")}
	orch := newTestOrchestrator(mock, comp, nil, nil, factRepo)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "我现在多重"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	resp, err := orch.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "服务正常", resp.Reply)
}

// TestChatOrchestrator_StreamExecute_LocalAnswer 验证流式场景下本地短路直接返回完整内容。
func TestChatOrchestrator_StreamExecute_LocalAnswer(t *testing.T) {
	t.Parallel()
	mock := &mockLLMClient{streamChunks: []string{"不", "应", "调", "用"}}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	factRepo := &mockFactRepository{
		byPredicate: map[string][]*entity.ExtractedFact{
			"体重是": {
				{FactID: "f1", Subject: "用户", Predicate: "体重是", Object: "110公斤", Status: entity.FactStatusApproved, Confidence: 0.95},
			},
		},
	}
	orch := newTestOrchestrator(mock, comp, nil, nil, factRepo)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "我现在多重"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	var chunks []string
	_, _, final, err := orch.StreamExecute(context.Background(), req, func(chunk string) {
		chunks = append(chunks, chunk)
	})
	require.NoError(t, err)
	assert.Equal(t, "记录中显示，你当前体重为 110公斤。", final)
	require.Len(t, chunks, 1)
	assert.Equal(t, "记录中显示，你当前体重为 110公斤。", chunks[0])
	// StreamChat 不应被调用
	assert.Empty(t, mock.streamCallbacks)
}

// TestChatOrchestrator_StreamExecute_Success 验证流式对话内容经合规检测后统一推送。
func TestChatOrchestrator_StreamExecute_Success(t *testing.T) {
	t.Parallel()
	mock := &mockLLMClient{streamChunks: []string{"你", "好", "！"}}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := newTestOrchestrator(mock, comp, nil, nil, nil)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "打招呼"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	var collected []string
	_, _, finalContent, err := orch.StreamExecute(context.Background(), req, func(chunk string) {
		collected = append(collected, chunk)
	})
	require.NoError(t, err)
	// 逐 chunk 透传，最终内容经检测后返回
	assert.Equal(t, []string{"你", "好", "！"}, collected)
	assert.Equal(t, "你好！", finalContent)
}

// TestChatOrchestrator_StreamExecute_Error 验证流式错误被正确包装。
func TestChatOrchestrator_StreamExecute_Error(t *testing.T) {
	t.Parallel()
	mock := &mockLLMClient{streamErr: fmt.Errorf("connection reset")}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := newTestOrchestrator(mock, comp, nil, nil, nil)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "test"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	_, _, _, err := orch.StreamExecute(context.Background(), req, func(chunk string) {})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stream execution failed")
}

// TestChatOrchestrator_StreamExecute_WithDeidentify 验证流式场景下输入脱敏生效。
func TestChatOrchestrator_StreamExecute_WithDeidentify(t *testing.T) {
	t.Parallel()
	mock := &mockLLMClient{streamChunks: []string{"收", "到"}}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	deid := &mockDeidentifier{
		result: models.DeidentifyResult{
			OriginalText: "邮箱 test@example.com",
			SafeText:     "邮箱 {{EMAIL_abc12345}}",
			Placeholder:  map[string]string{"{{EMAIL_abc12345}}": "test@example.com"},
		},
	}
	orch := newTestOrchestrator(mock, comp, deid, nil, nil)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "邮箱 test@example.com"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	var full string
	_, _, finalContent, err := orch.StreamExecute(context.Background(), req, func(chunk string) {
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	assert.True(t, isLocalModel(models.ProviderOllama))
	assert.True(t, isLocalModel(models.ProviderLocal))
	assert.False(t, isLocalModel(models.ProviderKimi))
	assert.False(t, isLocalModel(models.ProviderOpenAI))
	assert.False(t, isLocalModel(models.ProviderQwen))
	assert.False(t, isLocalModel(models.ProviderSiliconFlow))
}

// mockProviderStoreCtxErr 是一个在 context 已取消时返回 context 错误的 ProviderStore Mock。
type mockProviderStoreCtxErr struct{}

func (m *mockProviderStoreCtxErr) Create(ctx context.Context, provider *models.ProviderConfig) error {
	return nil
}
func (m *mockProviderStoreCtxErr) Update(ctx context.Context, provider *models.ProviderConfig) error {
	return nil
}
func (m *mockProviderStoreCtxErr) Delete(ctx context.Context, id string) error { return nil }
func (m *mockProviderStoreCtxErr) Get(ctx context.Context, id string) (*models.ProviderConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return &models.ProviderConfig{ID: id, APIHost: "https://api.example.com", ModelID: "test-model"}, nil
}
func (m *mockProviderStoreCtxErr) List(ctx context.Context) ([]*models.ProviderConfig, error) {
	return []*models.ProviderConfig{}, nil
}

var _ port.ProviderStore = (*mockProviderStoreCtxErr)(nil)

// TestChatOrchestrator_calculateConfidence_nilAggregator 验证 confidenceAggregator 为 nil 时返回零值结果。
func TestChatOrchestrator_calculateConfidence_nilAggregator(t *testing.T) {
	t.Parallel()
	mock := &mockLLMClient{}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	factory := &mockLLMClientFactory{client: mock}
	store := &mockProviderStore{}
	// 显式将 confidenceAggregator 传为 nil
	orch := NewChatOrchestrator(factory, store, nil, nil, comp, nil, nil, nil, &mockFactRepository{}, NewIntentResolver(NewQueryExpansionService()), NewLocalAnswerService())

	result := orch.calculateConfidence("测试回复", nil)

	require.NotNil(t, result)
	assert.Equal(t, 0.0, result.OverallScore)
	assert.Equal(t, entity.ConfidenceLevelE, result.Level)
	assert.Equal(t, "置信度引擎未初始化", result.Explanation)
	assert.Equal(t, entity.ConfidenceLevelE.Suggestion(), result.Suggestion)
	assert.Empty(t, result.MissingInfo)
}

// TestChatOrchestrator_resolveLLMClient_cancelledContext 验证传入已取消的 context 时返回错误。
func TestChatOrchestrator_resolveLLMClient_cancelledContext(t *testing.T) {
	t.Parallel()
	mock := &mockLLMClient{}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	factory := &mockLLMClientFactory{client: mock}
	store := &mockProviderStoreCtxErr{}
	orch := NewChatOrchestrator(factory, store, nil, nil, comp, nil, nil, NewConfidenceAggregator(), &mockFactRepository{}, NewIntentResolver(NewQueryExpansionService()), NewLocalAnswerService())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	_, err := orch.resolveLLMClient(ctx, "test-provider")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get provider")
}

// TestChatOrchestrator_resolveLLMClient_contextDeadline 验证传入带 deadline 的 context 时正常返回。
func TestChatOrchestrator_resolveLLMClient_contextDeadline(t *testing.T) {
	t.Parallel()
	mock := &mockLLMClient{}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	factory := &mockLLMClientFactory{client: mock}
	store := &mockProviderStore{}
	orch := NewChatOrchestrator(factory, store, nil, nil, comp, nil, nil, NewConfidenceAggregator(), &mockFactRepository{}, NewIntentResolver(NewQueryExpansionService()), NewLocalAnswerService())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := orch.resolveLLMClient(ctx, "test-provider")
	require.NoError(t, err)
	assert.NotNil(t, client)
}

// TestChatOrchestrator_ExtractFactsFromReply_UsesUserContentOnly 验证事实提取仅使用用户内容。
func TestChatOrchestrator_ExtractFactsFromReply_UsesUserContentOnly(t *testing.T) {
	t.Parallel()
	mock := &mockLLMForFactExtraction{
		response: `[{"subject":"用户","predicate":"体重是","object":"110公斤","confidence":0.95}]`,
	}
	extractor := NewFactExtractor(mock)
	facts, err := extractor.ParseFacts(context.Background(), "我体重110公斤")
	require.NoError(t, err)
	require.Len(t, facts, 1)
	assert.Equal(t, "110公斤", facts[0].Object)

	// 模拟 ChatOrchestrator 调用：传入 userContent + aiReply，
	// 但底层 extractor.ParseFacts 应只接收 userContent
	mock2 := &mockLLMClient{chatReply: `[{"subject":"AI","predicate":"无法告知","object":"体重","confidence":0.8}]`}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	factory := &mockLLMClientFactory{client: mock2}
	store := &mockProviderStore{}
	orch := NewChatOrchestrator(factory, store, nil, nil, comp, nil, nil, NewConfidenceAggregator(), &mockFactRepository{}, NewIntentResolver(NewQueryExpansionService()), NewLocalAnswerService())

	facts, err = orch.ExtractFactsFromReply(context.Background(), "我体重110公斤", "AI无法知道你的体重", "test-provider")
	require.NoError(t, err)
	// AI 回复内容不应被抽成事实，质量门禁也会过滤 AI subject
	assert.Empty(t, facts, "不应从 AI 回复中提取事实")
}

// multiClientMockFactory 根据 provider ID 返回不同的 mock client。
type multiClientMockFactory struct {
	clients map[string]port.LLMClient
}

func (m *multiClientMockFactory) CreateClient(providerConfig *models.ProviderConfig) (port.LLMClient, error) {
	if client, ok := m.clients[providerConfig.ID]; ok {
		return client, nil
	}
	return nil, fmt.Errorf("no mock client for provider %s", providerConfig.ID)
}

var _ port.LLMClientFactory = (*multiClientMockFactory)(nil)

// multiProviderStore 返回不同 provider 配置的 store。
type multiProviderStore struct {
	providers map[string]*models.ProviderConfig
}

func (m *multiProviderStore) Create(ctx context.Context, provider *models.ProviderConfig) error {
	return nil
}
func (m *multiProviderStore) Update(ctx context.Context, provider *models.ProviderConfig) error {
	return nil
}
func (m *multiProviderStore) Delete(ctx context.Context, id string) error { return nil }
func (m *multiProviderStore) Get(ctx context.Context, id string) (*models.ProviderConfig, error) {
	if p, ok := m.providers[id]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("provider not found: %s", id)
}
func (m *multiProviderStore) List(ctx context.Context) ([]*models.ProviderConfig, error) {
	var result []*models.ProviderConfig
	for _, p := range m.providers {
		result = append(result, p)
	}
	return result, nil
}

var _ port.ProviderStore = (*multiProviderStore)(nil)

// TestChatOrchestrator_ProviderRouting_DifferentProviderID 验证不同 ProviderID 路由到正确客户端。
func TestChatOrchestrator_ProviderRouting_DifferentProviderID(t *testing.T) {
	t.Parallel()
	kimiClient := &mockLLMClient{chatReply: "Kimi 回复"}
	openaiClient := &mockLLMClient{chatReply: "OpenAI 回复"}

	factory := &multiClientMockFactory{
		clients: map[string]port.LLMClient{
			"kimi-provider":   kimiClient,
			"openai-provider": openaiClient,
		},
	}
	store := &multiProviderStore{
		providers: map[string]*models.ProviderConfig{
			"kimi-provider":   {ID: "kimi-provider", APIHost: "https://api.moonshot.cn", ModelID: "kimi-v1"},
			"openai-provider": {ID: "openai-provider", APIHost: "https://api.openai.com", ModelID: "gpt-4o"},
		},
	}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := NewChatOrchestrator(factory, store, nil, nil, comp, nil, nil, NewConfidenceAggregator(), &mockFactRepository{}, NewIntentResolver(NewQueryExpansionService()), NewLocalAnswerService())

	// 测试 Kimi provider
	req1 := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "你好"}},
		Model:      models.ProviderKimi,
		ProviderID: "kimi-provider",
	}
	resp1, err := orch.Execute(context.Background(), req1)
	require.NoError(t, err)
	assert.Equal(t, "Kimi 回复", resp1.Reply)

	// 测试 OpenAI provider
	req2 := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "hello"}},
		Model:      models.ProviderOpenAI,
		ProviderID: "openai-provider",
	}
	resp2, err := orch.Execute(context.Background(), req2)
	require.NoError(t, err)
	assert.Equal(t, "OpenAI 回复", resp2.Reply)
}

// TestChatOrchestrator_ProviderRouting_UnknownProviderID 验证未知 ProviderID 返回错误。
func TestChatOrchestrator_ProviderRouting_UnknownProviderID(t *testing.T) {
	t.Parallel()
	factory := &multiClientMockFactory{clients: map[string]port.LLMClient{}}
	store := &multiProviderStore{providers: map[string]*models.ProviderConfig{}}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := NewChatOrchestrator(factory, store, nil, nil, comp, nil, nil, NewConfidenceAggregator(), &mockFactRepository{}, NewIntentResolver(NewQueryExpansionService()), NewLocalAnswerService())

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "test"}},
		Model:      models.ProviderKimi,
		ProviderID: "unknown-provider",
	}
	_, err := orch.Execute(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get provider")
}

// TestChatOrchestrator_OfflineFallback_CloudFails 验证云端失败时降级到本地模型。
func TestChatOrchestrator_OfflineFallback_CloudFails(t *testing.T) {
	t.Parallel()
	// 云端客户端返回错误
	cloudClient := &mockLLMClient{chatErr: fmt.Errorf("connection timeout")}
	// 本地客户端正常响应
	localClient := &mockLLMClient{chatReply: "本地模型回复"}

	factory := &multiClientMockFactory{
		clients: map[string]port.LLMClient{
			"cloud-provider": cloudClient,
			"local-provider": localClient,
		},
	}
	store := &multiProviderStore{
		providers: map[string]*models.ProviderConfig{
			"cloud-provider": {ID: "cloud-provider", APIHost: "https://api.moonshot.cn", ModelID: "kimi-v1"},
			"local-provider": {ID: "local-provider", APIHost: "http://localhost:11434", ModelID: "llama3"},
		},
	}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := NewChatOrchestrator(factory, store, nil, nil, comp, nil, nil, NewConfidenceAggregator(), &mockFactRepository{}, NewIntentResolver(NewQueryExpansionService()), NewLocalAnswerService())

	// 先测试云端失败
	req1 := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "test"}},
		Model:      models.ProviderKimi,
		ProviderID: "cloud-provider",
	}
	_, err := orch.Execute(context.Background(), req1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chat execution failed")

	// 再测试本地成功（应用层应主动切换 providerID 到本地）
	req2 := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "test"}},
		Model:      models.ProviderOllama,
		ProviderID: "local-provider",
	}
	resp2, err := orch.Execute(context.Background(), req2)
	require.NoError(t, err)
	assert.Equal(t, "本地模型回复", resp2.Reply)
}

// TestChatOrchestrator_ModelSwitching_MidConversation 验证对话中切换模型。
func TestChatOrchestrator_ModelSwitching_MidConversation(t *testing.T) {
	t.Parallel()
	kimiClient := &mockLLMClient{chatReply: "Kimi 回复"}
	qwenClient := &mockLLMClient{chatReply: "Qwen 回复"}

	factory := &multiClientMockFactory{
		clients: map[string]port.LLMClient{
			"kimi-provider": kimiClient,
			"qwen-provider": qwenClient,
		},
	}
	store := &multiProviderStore{
		providers: map[string]*models.ProviderConfig{
			"kimi-provider": {ID: "kimi-provider", APIHost: "https://api.moonshot.cn", ModelID: "kimi-v1"},
			"qwen-provider": {ID: "qwen-provider", APIHost: "https://dashscope.aliyuncs.com", ModelID: "qwen-turbo"},
		},
	}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := NewChatOrchestrator(factory, store, nil, nil, comp, nil, nil, NewConfidenceAggregator(), &mockFactRepository{}, NewIntentResolver(NewQueryExpansionService()), NewLocalAnswerService())

	// 第一轮：使用 Kimi
	req1 := ChatRequest{
		ConversationID: "conv-1",
		Messages:       []models.Message{{Role: models.RoleUser, Content: "你好"}},
		Model:          models.ProviderKimi,
		ProviderID:     "kimi-provider",
	}
	resp1, err := orch.Execute(context.Background(), req1)
	require.NoError(t, err)
	assert.Equal(t, "Kimi 回复", resp1.Reply)

	// 第二轮：切换到 Qwen（保持同一会话）
	req2 := ChatRequest{
		ConversationID: "conv-1",
		Messages: []models.Message{
			{Role: models.RoleUser, Content: "你好"},
			{Role: models.RoleAssistant, Content: "Kimi 回复"},
			{Role: models.RoleUser, Content: "换个模型回答"},
		},
		Model:      models.ProviderQwen,
		ProviderID: "qwen-provider",
	}
	resp2, err := orch.Execute(context.Background(), req2)
	require.NoError(t, err)
	assert.Equal(t, "Qwen 回复", resp2.Reply)

	// 验证两个客户端都被调用过
	assert.NotEmpty(t, kimiClient.lastMessages, "Kimi client 应被调用")
	assert.NotEmpty(t, qwenClient.lastMessages, "Qwen client 应被调用")
}

// TestChatOrchestrator_ModelSwitching_StreamMode 验证流式模式下切换模型。
func TestChatOrchestrator_ModelSwitching_StreamMode(t *testing.T) {
	t.Parallel()
	kimiClient := &mockLLMClient{streamChunks: []string{"Kimi", "流式"}}
	ollamaClient := &mockLLMClient{streamChunks: []string{"Ollama", "本地"}}

	factory := &multiClientMockFactory{
		clients: map[string]port.LLMClient{
			"kimi-provider":   kimiClient,
			"ollama-provider": ollamaClient,
		},
	}
	store := &multiProviderStore{
		providers: map[string]*models.ProviderConfig{
			"kimi-provider":   {ID: "kimi-provider", APIHost: "https://api.moonshot.cn", ModelID: "kimi-v1"},
			"ollama-provider": {ID: "ollama-provider", APIHost: "http://localhost:11434", ModelID: "llama3"},
		},
	}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	orch := NewChatOrchestrator(factory, store, nil, nil, comp, nil, nil, NewConfidenceAggregator(), &mockFactRepository{}, NewIntentResolver(NewQueryExpansionService()), NewLocalAnswerService())

	// 第一轮：Kimi 流式
	req1 := ChatRequest{
		ConversationID: "conv-stream-1",
		Messages:       []models.Message{{Role: models.RoleUser, Content: "你好"}},
		Model:          models.ProviderKimi,
		ProviderID:     "kimi-provider",
	}
	var chunks1 []string
	_, _, final1, err := orch.StreamExecute(context.Background(), req1, func(chunk string) {
		chunks1 = append(chunks1, chunk)
	})
	require.NoError(t, err)
	assert.Equal(t, "Kimi流式", final1)

	// 第二轮：切换到 Ollama 流式
	req2 := ChatRequest{
		ConversationID: "conv-stream-1",
		Messages: []models.Message{
			{Role: models.RoleUser, Content: "你好"},
			{Role: models.RoleAssistant, Content: "Kimi流式"},
			{Role: models.RoleUser, Content: "用本地模型"},
		},
		Model:      models.ProviderOllama,
		ProviderID: "ollama-provider",
	}
	var chunks2 []string
	_, _, final2, err := orch.StreamExecute(context.Background(), req2, func(chunk string) {
		chunks2 = append(chunks2, chunk)
	})
	require.NoError(t, err)
	assert.Equal(t, "Ollama本地", final2)
}

// TestChatOrchestrator_NilConfidenceAggregator 验证 confidenceAggregator 为 nil 时不 panic。
func TestChatOrchestrator_NilConfidenceAggregator(t *testing.T) {
	t.Parallel()
	mock := &mockLLMClient{chatReply: "你好"}
	comp := newTestComplianceChecker(t, mustEmptyRulesPath(t))
	factory := &mockLLMClientFactory{client: mock}
	store := &mockProviderStore{}
	// 故意不传 ConfidenceAggregator（nil）
	orch := NewChatOrchestrator(factory, store, nil, nil, comp, nil, nil, nil, &mockFactRepository{}, NewIntentResolver(NewQueryExpansionService()), NewLocalAnswerService())

	req := ChatRequest{
		ConversationID: "test-conv",
		Messages: []models.Message{
			{Role: models.RoleUser, Content: "你好"},
		},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	// Execute 路径不应 panic
	resp, err := orch.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp.ConfidenceResult)
	assert.Equal(t, entity.ConfidenceLevelE, resp.ConfidenceResult.Level)

	// StreamExecute 路径不应 panic
	var chunks []string
	_, confResult, _, err := orch.StreamExecute(context.Background(), req, func(chunk string) {
		chunks = append(chunks, chunk)
	})
	require.NoError(t, err)
	assert.NotNil(t, confResult)
	assert.Equal(t, entity.ConfidenceLevelE, confResult.Level)
}

// ---- 合规范式测试 ----

// TestExecute_ComplianceCheckError_FailClosed 验证非流式路径合规检查异常时返回安全文案。
func TestExecute_ComplianceCheckError_FailClosed(t *testing.T) {
	t.Parallel()
	mock := &mockLLMClient{chatReply: "你患有糖尿病，需要服用二甲双胍每天两次。"}
	comp := &mockComplianceCheckerForTest{err: fmt.Errorf("compliance timeout")}
	orch := newTestOrchestratorWithChecker(mock, comp)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "我得了什么病？"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	resp, err := orch.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "内容审核服务暂时不可用，请稍后重试或咨询专业医生。", resp.Reply)
	assert.Contains(t, resp.Warnings, "COMPLIANCE_CHECK_ERROR")
}

// TestExecute_ComplianceCheck_L1Block 验证 L1 阻断时 reply 替换为 SafeText。
func TestExecute_ComplianceCheck_L1Block(t *testing.T) {
	t.Parallel()
	mock := &mockLLMClient{chatReply: "你患有糖尿病，需要治疗。"}
	comp := &mockComplianceCheckerForTest{
		result: &ComplianceResult{
			Blocked:  true,
			Level:    application.L1Blocked.String(),
			SafeText: "BLOCKED_SAFE_TEXT",
		},
	}
	orch := newTestOrchestratorWithChecker(mock, comp)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "我得了什么病？"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	resp, err := orch.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "BLOCKED_SAFE_TEXT", resp.Reply)
	assert.Contains(t, resp.Warnings, application.L1Blocked.String())
}

// TestExecute_ComplianceCheck_L2Warning 验证 L2 警告时 reply 不变但有警告信息。
func TestExecute_ComplianceCheck_L2Warning(t *testing.T) {
	t.Parallel()
	mock := &mockLLMClient{chatReply: "建议你考虑使用布洛芬缓解疼痛。"}
	comp := &mockComplianceCheckerForTest{
		result: &ComplianceResult{
			Level:   application.L2Warning.String(),
			Warning: "药物建议需谨慎",
		},
	}
	orch := newTestOrchestratorWithChecker(mock, comp)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "我头痛怎么办？"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	resp, err := orch.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, mock.chatReply, resp.Reply)
	assert.Contains(t, resp.Warnings, application.L2Warning.String())
	assert.Contains(t, resp.Warnings, "WARNING:药物建议需谨慎")
}

// TestExecute_ComplianceCheck_L3Notice 验证 L3 提示时 reply 不变但有通知信息。
func TestExecute_ComplianceCheck_L3Notice(t *testing.T) {
	t.Parallel()
	mock := &mockLLMClient{chatReply: "高血压是一种常见慢性病，需要注意饮食和运动。"}
	comp := &mockComplianceCheckerForTest{
		result: &ComplianceResult{
			Level:  application.L3Notice.String(),
			Notice: "以上内容仅供参考",
		},
	}
	orch := newTestOrchestratorWithChecker(mock, comp)

	req := ChatRequest{
		Messages:   []models.Message{{Role: models.RoleUser, Content: "什么是高血压？"}},
		Model:      models.ProviderKimi,
		ProviderID: "test-provider",
	}

	resp, err := orch.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, mock.chatReply, resp.Reply)
	assert.Contains(t, resp.Warnings, application.L3Notice.String())
	assert.Contains(t, resp.Warnings, "NOTICE:以上内容仅供参考")
}
