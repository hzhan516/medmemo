//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/medmemo/medmemo/internal/adapters/repository"
	"github.com/medmemo/medmemo/internal/application"
	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/internal/application/usecase"
	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/internal/infrastructure/database"
	"github.com/medmemo/medmemo/internal/infrastructure/secret"
	"github.com/medmemo/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLLMClient 是 port.LLMClient 的手动 Mock 实现。
type mockLLMClient struct {
	chatReply    string
	streamChunks []string
	streamErr    error
	lastMessages []models.Message
}

func (m *mockLLMClient) Chat(ctx context.Context, messages []models.Message) (string, error) {
	m.lastMessages = messages
	return m.chatReply, nil
}

func (m *mockLLMClient) StreamChat(ctx context.Context, messages []models.Message, callback func(chunk string)) error {
	m.lastMessages = messages
	for _, chunk := range m.streamChunks {
		callback(chunk)
	}
	return m.streamErr
}

func (m *mockLLMClient) CheckAvailability(ctx context.Context) (bool, string) {
	return true, "available"
}

var _ port.LLMClient = (*mockLLMClient)(nil)

// mockSecretStore 是 secret.Store 的内存实现。
type mockSecretStore struct {
	data map[string][]byte
}

func newMockSecretStore() *mockSecretStore {
	return &mockSecretStore{data: make(map[string][]byte)}
}

func (m *mockSecretStore) Set(key string, value []byte) error {
	m.data[key] = value
	return nil
}

func (m *mockSecretStore) Get(key string) ([]byte, error) {
	v, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("secret not found: %s", key)
	}
	return v, nil
}

func (m *mockSecretStore) Delete(key string) error {
	delete(m.data, key)
	return nil
}

var _ secret.Store = (*mockSecretStore)(nil)

// mockMemoryRepository 实现 port.MemoryRepository 接口的内存版本。
type mockMemoryRepository struct{}

func (m *mockMemoryRepository) Save(ctx context.Context, mem *entity.HealthMemory) error { return nil }
func (m *mockMemoryRepository) GetByID(ctx context.Context, id models.MemoryID) (*entity.HealthMemory, error) {
	return nil, nil
}
func (m *mockMemoryRepository) Search(ctx context.Context, query string, limit int) ([]*entity.HealthMemory, error) {
	return nil, nil
}
func (m *mockMemoryRepository) SemanticSearch(ctx context.Context, embedding []float32, topK int) ([]*entity.HealthMemory, error) {
	return nil, nil
}
func (m *mockMemoryRepository) ListByTier(ctx context.Context, tier entity.MemoryTier, limit int) ([]*entity.HealthMemory, error) {
	return nil, nil
}
func (m *mockMemoryRepository) Delete(ctx context.Context, id models.MemoryID) error { return nil }

var _ port.MemoryRepository = (*mockMemoryRepository)(nil)

// mockSensitiveDetector 实现 port.SensitiveDetector 接口的内存版本。
type mockSensitiveDetector struct{}

func (m *mockSensitiveDetector) Detect(ctx context.Context, text string) ([]models.SensitiveEntity, error) {
	return nil, nil
}

var _ port.SensitiveDetector = (*mockSensitiveDetector)(nil)

// mockComplianceChecker 基于 application.ComplianceInterceptor 的 ComplianceChecker 实现。
type mockComplianceChecker struct {
	interceptor *application.ComplianceInterceptor
}

func (m *mockComplianceChecker) Check(ctx context.Context, text string) (*usecase.ComplianceResult, error) {
	res, err := m.interceptor.EvaluateWithInlineReplace(ctx, text)
	if err != nil {
		return nil, err
	}
	return &usecase.ComplianceResult{
		Blocked:       res.Blocked,
		Level:         res.Level,
		SafeText:      res.SafeText,
		Warning:       res.Warning,
		Notice:        res.Notice,
		MatchedRule:   res.MatchedRule,
		ReplacedTerms: res.ReplacedTerms,
	}, nil
}

var _ usecase.ComplianceChecker = (*mockComplianceChecker)(nil)

// mustEmptyRulesPath 创建包含空规则库的临时 JSON 文件并返回路径。
func mustEmptyRulesPath(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "rules.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"version":"test","rules":[]}`), 0644))
	return path
}

// setupTestDB 创建内存 SQLite 数据库连接并执行迁移。
func setupTestDB(t *testing.T) *database.SQLiteConnector {
	t.Helper()
	tmpDir := t.TempDir()
	conn, err := database.NewSQLiteConnector(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, conn)

	ctx := context.Background()
	require.NoError(t, conn.Migrate(ctx))
	return conn
}

// TestE2E_Conversation_FullFlow 验证完整对话生命周期：创建会话 → 发送消息 → 流式响应 → 获取会话列表。
func TestE2E_Conversation_FullFlow(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	comp, err := application.NewComplianceInterceptor(mustEmptyRulesPath(t))
	require.NoError(t, err)

	// 创建 repository
	convRepo := repository.NewConversationRepoSQLite(conn)
	msgRepo := repository.NewMessageRepoSQLite(conn)

	// 创建 usecase
	mockLLM := &mockLLMClient{
		chatReply:    "这是非流式回复",
		streamChunks: []string{"流", "式", "回", "复"},
	}
	checker := &mockComplianceChecker{interceptor: comp}
	chatOrch := usecase.NewChatOrchestrator(mockLLM, &mockMemoryRepository{}, &mockSensitiveDetector{}, checker, nil, nil)

	ctx := context.Background()

	// 1. 创建会话
	conv := entity.NewConversation(models.ProviderKimi)
	err = convRepo.Save(ctx, conv)
	require.NoError(t, err)
	assert.NotEmpty(t, conv.ID)

	// 2. 发送非流式消息（手动保存到数据库，模拟 WailsApp 的完整流程）
	userMsg := &entity.Message{ID: fmt.Sprintf("msg_%d", time.Now().UnixNano()), Role: models.RoleUser, Content: "你好", Timestamp: time.Now()}
	require.NoError(t, msgRepo.Save(ctx, conv.ID, userMsg))

	req := usecase.ChatRequest{
		ConversationID: conv.ID,
		Messages:       []models.Message{{Role: models.RoleUser, Content: "你好"}},
		Model:          models.ProviderKimi,
	}
	resp, err := chatOrch.Execute(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "这是非流式回复", resp.Reply)

	aiMsg := &entity.Message{ID: fmt.Sprintf("msg_%d", time.Now().UnixNano()), Role: models.RoleAssistant, Content: resp.Reply, Timestamp: time.Now()}
	require.NoError(t, msgRepo.Save(ctx, conv.ID, aiMsg))

	// 3. 验证消息已持久化（按时间降序）
	msgs, _, err := msgRepo.ListByConversation(ctx, conv.ID, "", 10)
	require.NoError(t, err)
	require.Len(t, msgs, 2) // 用户消息 + AI 回复
	assert.Equal(t, models.RoleAssistant, msgs[0].Role)
	assert.Equal(t, "这是非流式回复", msgs[0].Content)
	assert.Equal(t, models.RoleUser, msgs[1].Role)
	assert.Equal(t, "你好", msgs[1].Content)

	// 4. 流式发送消息
	userMsg2 := &entity.Message{ID: fmt.Sprintf("msg_%d", time.Now().UnixNano()), Role: models.RoleUser, Content: "流式测试", Timestamp: time.Now()}
	require.NoError(t, msgRepo.Save(ctx, conv.ID, userMsg2))

	var streamResult string
	err = chatOrch.StreamExecute(ctx, usecase.ChatRequest{
		ConversationID: conv.ID,
		Messages:       []models.Message{{Role: models.RoleUser, Content: "流式测试"}},
		Model:          models.ProviderKimi,
	}, func(chunk string) {
		streamResult += chunk
	})
	require.NoError(t, err)
	assert.Equal(t, "流式回复", streamResult)

	aiMsg2 := &entity.Message{ID: fmt.Sprintf("msg_%d", time.Now().UnixNano()), Role: models.RoleAssistant, Content: streamResult, Timestamp: time.Now()}
	require.NoError(t, msgRepo.Save(ctx, conv.ID, aiMsg2))

	// 5. 获取会话列表
	convs, err := convRepo.ListRecent(ctx, 100)
	require.NoError(t, err)
	require.Len(t, convs, 1)
	assert.Equal(t, conv.ID, convs[0].ID)
}

// TestE2E_Conversation_MultipleSessions 验证多会话隔离。
func TestE2E_Conversation_MultipleSessions(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	comp, err := application.NewComplianceInterceptor(mustEmptyRulesPath(t))
	require.NoError(t, err)

	convRepo := repository.NewConversationRepoSQLite(conn)
	msgRepo := repository.NewMessageRepoSQLite(conn)
	checker := &mockComplianceChecker{interceptor: comp}
	chatOrch := usecase.NewChatOrchestrator(&mockLLMClient{chatReply: "回复"}, &mockMemoryRepository{}, &mockSensitiveDetector{}, checker, nil, nil)

	ctx := context.Background()

	// 创建两个会话
	conv1 := entity.NewConversation(models.ProviderKimi)
	conv2 := entity.NewConversation(models.ProviderOpenAI)
	require.NoError(t, convRepo.Save(ctx, conv1))
	require.NoError(t, convRepo.Save(ctx, conv2))

	// 在 conv1 发送消息（手动保存）
	userMsg := &entity.Message{ID: fmt.Sprintf("msg_%d", time.Now().UnixNano()), Role: models.RoleUser, Content: "会话1的消息", Timestamp: time.Now()}
	require.NoError(t, msgRepo.Save(ctx, conv1.ID, userMsg))

	_, err = chatOrch.Execute(ctx, usecase.ChatRequest{
		ConversationID: conv1.ID,
		Messages:       []models.Message{{Role: models.RoleUser, Content: "会话1的消息"}},
		Model:          models.ProviderKimi,
	})
	require.NoError(t, err)

	aiMsg := &entity.Message{ID: fmt.Sprintf("msg_%d", time.Now().UnixNano()), Role: models.RoleAssistant, Content: "回复", Timestamp: time.Now()}
	require.NoError(t, msgRepo.Save(ctx, conv1.ID, aiMsg))

	// 验证 conv2 没有消息
	msgs2, _, err := msgRepo.ListByConversation(ctx, conv2.ID, "", 10)
	require.NoError(t, err)
	assert.Len(t, msgs2, 0)

	// 验证 conv1 有消息（按时间降序：AI 回复在前，用户消息在后）
	msgs1, _, err := msgRepo.ListByConversation(ctx, conv1.ID, "", 10)
	require.NoError(t, err)
	assert.Len(t, msgs1, 2)
	assert.Equal(t, models.RoleAssistant, msgs1[0].Role)
	assert.Equal(t, models.RoleUser, msgs1[1].Role)
}
