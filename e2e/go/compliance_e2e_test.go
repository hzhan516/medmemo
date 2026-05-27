//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

// mustWriteRules 将规则库 JSON 写入临时文件并返回路径。
func mustWriteRules(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "rules.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

// defaultRules 返回包含全部四级规则的测试规则库。
func defaultRules() string {
	return `{
  "version": "e2e-test-1.0.0",
  "updated_at": "2026-05-18",
  "rules": [
    {
      "id": "l1-diag-definite-001",
      "level": "L1",
      "name": "确诊性诊断结论",
      "patterns": ["你患有[一-龥]+病", "确诊为[一-龥]+"],
      "action": "block",
      "replacement": "BLOCKED_TEXT"
    },
    {
      "id": "l2-diag-implied-001",
      "level": "L2",
      "name": "暗示性诊断",
      "patterns": ["可能是[一-龥]+"],
      "action": "warn",
      "warning": "WARN_IMPLIED"
    },
    {
      "id": "l3-disease-severe-001",
      "level": "L3",
      "name": "严重疾病科普",
      "patterns": ["癌症","白血病"],
      "action": "notice",
      "notice": "NOTICE_SEVERE"
    }
  ]
}`
}

// TestE2E_Compliance_L1Block 验证 L1 阻断级内容被拦截。
func TestE2E_Compliance_L1Block(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	path := mustWriteRules(t, defaultRules())
	comp, err := application.NewComplianceInterceptor(path)
	require.NoError(t, err)

	convRepo := repository.NewConversationRepoSQLite(conn)
	checker := &mockComplianceChecker{interceptor: comp}
	chatOrch := usecase.NewChatOrchestrator(&mockLLMClientFactory{client: &mockLLMClient{chatReply: "你患有糖尿病，需要治疗。"}}, newMockProviderStore(), &mockMemoryRepository{}, &mockSensitiveDetector{}, checker, nil, &mockMemoryQuerier{}, nil)

	ctx := context.Background()
	conv := entity.NewConversation(models.ProviderKimi)
	require.NoError(t, convRepo.Save(ctx, conv))

	resp, err := chatOrch.Execute(ctx, usecase.ChatRequest{
		ConversationID: conv.ID,
		Messages:       []models.Message{{Role: models.RoleUser, Content: "测试"}},
		Model:          models.ProviderKimi,
		ProviderID:     "test-kimi",
	})
	require.NoError(t, err)

	// Execute 内部已完成合规检测，L1 内容会被替换为 BLOCKED_TEXT
	assert.Contains(t, resp.Reply, "BLOCKED_TEXT")
	assert.Contains(t, resp.Warnings, "L1_BLOCKED")
}

// TestE2E_Compliance_L2Warning 验证 L2 警告级内容触发警告。
func TestE2E_Compliance_L2Warning(t *testing.T) {
	path := mustWriteRules(t, defaultRules())
	comp, err := application.NewComplianceInterceptor(path)
	require.NoError(t, err)

	result, err := comp.Evaluate(context.Background(), "你可能是感冒了")
	require.NoError(t, err)
	assert.False(t, result.Blocked)
	assert.Equal(t, "L2_WARNING", result.Level)
	assert.Equal(t, "WARN_IMPLIED", result.Warning)
}

// TestE2E_Compliance_L3Notice 验证 L3 提示级内容追加免责声明。
func TestE2E_Compliance_L3Notice(t *testing.T) {
	path := mustWriteRules(t, defaultRules())
	comp, err := application.NewComplianceInterceptor(path)
	require.NoError(t, err)

	result, err := comp.Evaluate(context.Background(), "癌症的早期症状包括持续发热。")
	require.NoError(t, err)
	assert.False(t, result.Blocked)
	assert.Equal(t, "L3_NOTICE", result.Level)
	assert.Equal(t, "NOTICE_SEVERE", result.Notice)
}

// TestE2E_Compliance_L4Normal 验证 L4 正常级内容直接放行。
func TestE2E_Compliance_L4Normal(t *testing.T) {
	path := mustWriteRules(t, defaultRules())
	comp, err := application.NewComplianceInterceptor(path)
	require.NoError(t, err)

	result, err := comp.Evaluate(context.Background(), "保持规律作息有助于健康。")
	require.NoError(t, err)
	assert.False(t, result.Blocked)
	assert.Equal(t, "L4_NORMAL", result.Level)
	assert.Empty(t, result.MatchedRule)
}

// TestE2E_Compliance_PipelineEndToEnd 验证输入 → 脱敏 → 合规检测 → 输出的完整链路。
func TestE2E_Compliance_PipelineEndToEnd(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	path := mustWriteRules(t, defaultRules())
	comp, err := application.NewComplianceInterceptor(path)
	require.NoError(t, err)

	convRepo := repository.NewConversationRepoSQLite(conn)
	msgRepo := repository.NewMessageRepoSQLite(conn)
	checker := &mockComplianceChecker{interceptor: comp}

	// 使用含 L1 触发词的回复
	mockLLM := &mockLLMClient{chatReply: "你患有高血压病，建议服用药物。"}
	chatOrch := usecase.NewChatOrchestrator(&mockLLMClientFactory{client: mockLLM}, newMockProviderStore(), &mockMemoryRepository{}, &mockSensitiveDetector{}, checker, nil, &mockMemoryQuerier{}, nil)

	ctx := context.Background()
	conv := entity.NewConversation(models.ProviderKimi)
	require.NoError(t, convRepo.Save(ctx, conv))

	// 发送消息并获取回复（手动保存，模拟完整流程）
	userMsg := &entity.Message{ID: fmt.Sprintf("msg_%d", time.Now().UnixNano()), Role: models.RoleUser, Content: "我头晕", Timestamp: time.Now()}
	require.NoError(t, msgRepo.Save(ctx, conv.ID, userMsg))

	resp, err := chatOrch.Execute(ctx, usecase.ChatRequest{
		ConversationID: conv.ID,
		Messages:       []models.Message{{Role: models.RoleUser, Content: "我头晕"}},
		Model:          models.ProviderKimi,
		ProviderID:     "test-kimi",
	})
	require.NoError(t, err)

	aiMsg := &entity.Message{ID: fmt.Sprintf("msg_%d", time.Now().UnixNano()), Role: models.RoleAssistant, Content: resp.Reply, Timestamp: time.Now()}
	require.NoError(t, msgRepo.Save(ctx, conv.ID, aiMsg))

	// Execute 内部已完成合规检测，L1 内容被替换为 BLOCKED_TEXT
	assert.Contains(t, resp.Reply, "BLOCKED_TEXT")
	assert.Contains(t, resp.Warnings, "L1_BLOCKED")

	// 验证消息已持久化到数据库（按时间降序）
	msgs, _, err := msgRepo.ListByConversation(ctx, conv.ID, "", 10)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	assert.Equal(t, models.RoleAssistant, msgs[0].Role)
	assert.Contains(t, msgs[0].Content, "BLOCKED_TEXT")
	assert.Equal(t, models.RoleUser, msgs[1].Role)
	assert.Equal(t, "我头晕", msgs[1].Content)
}
