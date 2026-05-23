//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/adapters/repository"
	"github.com/hzhan516/medmemo/internal/application"
	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDeidentifier 实现 Deidentifier 接口，模拟 PII 脱敏。
type mockDeidentifier struct {
	result models.DeidentifyResult
	err    error
}

func (m *mockDeidentifier) Execute(ctx context.Context, raw string) (models.DeidentifyResult, error) {
	return m.result, m.err
}

// TestE2E_Privacy_DeidentifyPipeline 验证 PII 输入 → 脱敏 → LLM 调用 → 结果回填的完整链路。
func TestE2E_Privacy_DeidentifyPipeline(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	comp, err := application.NewComplianceInterceptor(mustEmptyRulesPath(t))
	require.NoError(t, err)

	convRepo := repository.NewConversationRepoSQLite(conn)
	msgRepo := repository.NewMessageRepoSQLite(conn)
	checker := &mockComplianceChecker{interceptor: comp}

	// LLM 返回包含占位符的回复
	mockLLM := &mockLLMClient{chatReply: "用户 {{NAME_1}} 的电话是 {{PHONE_1}}"}
	deid := &mockDeidentifier{
		result: models.DeidentifyResult{
			OriginalText: "我叫张三，电话 13800138000",
			SafeText:     "我叫 {{NAME_1}}，电话 {{PHONE_1}}",
			Placeholder: map[string]string{
				"{{NAME_1}}":  "张三",
				"{{PHONE_1}}": "13800138000",
			},
		},
	}
	chatOrch := usecase.NewChatOrchestrator(&mockLLMClientFactory{client: mockLLM}, newMockProviderStore(), &mockMemoryRepository{}, &mockSensitiveDetector{}, checker, deid, &mockMemoryQuerier{})

	ctx := context.Background()
	conv := entity.NewConversation(models.ProviderKimi)
	require.NoError(t, convRepo.Save(ctx, conv))

	// 发送包含 PII 的消息（手动保存）
	userMsg := &entity.Message{ID: fmt.Sprintf("msg_%d", time.Now().UnixNano()), Role: models.RoleUser, Content: "我叫张三，电话 13800138000", Timestamp: time.Now()}
	require.NoError(t, msgRepo.Save(ctx, conv.ID, userMsg))

	resp, err := chatOrch.Execute(ctx, usecase.ChatRequest{
		ConversationID: conv.ID,
		Messages:       []models.Message{{Role: models.RoleUser, Content: "我叫张三，电话 13800138000"}},
		Model:          models.ProviderKimi,
		ProviderID:     "test-kimi",
	})
	require.NoError(t, err)

	// 验证 LLM 收到的是脱敏后的文本
	require.Len(t, mockLLM.lastMessages, 1)
	assert.Equal(t, "我叫 {{NAME_1}}，电话 {{PHONE_1}}", mockLLM.lastMessages[0].Content)

	// 验证返回给用户的是回填后的原始文本
	assert.Equal(t, "用户 张三 的电话是 13800138000", resp.Reply)
}

// TestE2E_Privacy_DatabaseEncryption 验证消息存储到数据库（SQLCipher 加密层）。
func TestE2E_Privacy_DatabaseEncryption(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	comp, err := application.NewComplianceInterceptor(mustEmptyRulesPath(t))
	require.NoError(t, err)

	convRepo := repository.NewConversationRepoSQLite(conn)
	msgRepo := repository.NewMessageRepoSQLite(conn)
	checker := &mockComplianceChecker{interceptor: comp}
	chatOrch := usecase.NewChatOrchestrator(&mockLLMClientFactory{client: &mockLLMClient{chatReply: "回复"}}, newMockProviderStore(), &mockMemoryRepository{}, &mockSensitiveDetector{}, checker, nil, &mockMemoryQuerier{})

	ctx := context.Background()
	conv := entity.NewConversation(models.ProviderKimi)
	require.NoError(t, convRepo.Save(ctx, conv))

	userMsg := &entity.Message{ID: fmt.Sprintf("msg_%d", time.Now().UnixNano()), Role: models.RoleUser, Content: "敏感健康信息", Timestamp: time.Now()}
	require.NoError(t, msgRepo.Save(ctx, conv.ID, userMsg))

	_, err = chatOrch.Execute(ctx, usecase.ChatRequest{
		ConversationID: conv.ID,
		Messages:       []models.Message{{Role: models.RoleUser, Content: "敏感健康信息"}},
		Model:          models.ProviderKimi,
		ProviderID:     "test-kimi",
	})
	require.NoError(t, err)

	aiMsg := &entity.Message{ID: fmt.Sprintf("msg_%d", time.Now().UnixNano()), Role: models.RoleAssistant, Content: "回复", Timestamp: time.Now()}
	require.NoError(t, msgRepo.Save(ctx, conv.ID, aiMsg))

	// 验证数据库中存储了消息（按时间降序）
	msgs, _, err := msgRepo.ListByConversation(ctx, conv.ID, "", 10)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	assert.Equal(t, models.RoleAssistant, msgs[0].Role)
	assert.Equal(t, "回复", msgs[0].Content)
	assert.Equal(t, models.RoleUser, msgs[1].Role)
	assert.Equal(t, "敏感健康信息", msgs[1].Content)
}

// TestE2E_Privacy_APIKeySecretStore 验证 API Key 通过密钥环保管。
func TestE2E_Privacy_APIKeySecretStore(t *testing.T) {
	store := newMockSecretStore()

	// 保存 API Key
	err := store.Set("apikey:kimi", []byte("sk-test-secret-key"))
	require.NoError(t, err)

	// 读取 API Key
	key, err := store.Get("apikey:kimi")
	require.NoError(t, err)
	assert.Equal(t, "sk-test-secret-key", string(key))

	// 验证删除后无法读取
	err = store.Delete("apikey:kimi")
	require.NoError(t, err)
	_, err = store.Get("apikey:kimi")
	require.Error(t, err)
}

// TestE2E_Privacy_DesensitizeFallback 验证脱敏失败时降级透传原始文本。
func TestE2E_Privacy_DesensitizeFallback(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	comp, err := application.NewComplianceInterceptor(mustEmptyRulesPath(t))
	require.NoError(t, err)

	convRepo := repository.NewConversationRepoSQLite(conn)
	msgRepo := repository.NewMessageRepoSQLite(conn)
	checker := &mockComplianceChecker{interceptor: comp}

	mockLLM := &mockLLMClient{chatReply: "收到"}
	deid := &mockDeidentifier{err: assert.AnError}
	chatOrch := usecase.NewChatOrchestrator(&mockLLMClientFactory{client: mockLLM}, newMockProviderStore(), &mockMemoryRepository{}, &mockSensitiveDetector{}, checker, deid, &mockMemoryQuerier{})

	ctx := context.Background()
	conv := entity.NewConversation(models.ProviderKimi)
	require.NoError(t, convRepo.Save(ctx, conv))

	userMsg := &entity.Message{ID: fmt.Sprintf("msg_%d", time.Now().UnixNano()), Role: models.RoleUser, Content: "原始内容", Timestamp: time.Now()}
	require.NoError(t, msgRepo.Save(ctx, conv.ID, userMsg))

	resp, err := chatOrch.Execute(ctx, usecase.ChatRequest{
		ConversationID: conv.ID,
		Messages:       []models.Message{{Role: models.RoleUser, Content: "原始内容"}},
		Model:          models.ProviderKimi,
		ProviderID:     "test-kimi",
	})
	require.NoError(t, err)
	assert.Equal(t, "收到", resp.Reply)

	// 脱敏失败时 LLM 应收到原始文本
	require.Len(t, mockLLM.lastMessages, 1)
	assert.Equal(t, "原始内容", mockLLM.lastMessages[0].Content)
}
