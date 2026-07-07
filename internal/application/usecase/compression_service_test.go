package usecase

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyDropEarliestN(t *testing.T) {
	history := []models.Message{
		{Role: models.RoleUser, Content: "1"},
		{Role: models.RoleAssistant, Content: "2"},
		{Role: models.RoleUser, Content: "3"},
		{Role: models.RoleAssistant, Content: "4"},
		{Role: models.RoleUser, Content: "5"},
	}

	t.Run("clamps N and preserves recent tail", func(t *testing.T) {
		got := applyDropEarliestN(history, 10, 2)
		if len(got) != 2 {
			t.Fatalf("len(got) = %d, want 2", len(got))
		}
		want := []string{"4", "5"}
		for i, w := range want {
			if got[i].Content != w {
				t.Errorf("got[%d].Content = %q, want %q", i, got[i].Content, w)
			}
		}
	})

	t.Run("drops exactly N when possible", func(t *testing.T) {
		got := applyDropEarliestN(history, 2, 2)
		if len(got) != 3 {
			t.Fatalf("len(got) = %d, want 3", len(got))
		}
		want := []string{"3", "4", "5"}
		for i, w := range want {
			if got[i].Content != w {
				t.Errorf("got[%d].Content = %q, want %q", i, got[i].Content, w)
			}
		}
	})

	t.Run("negative N with positive recentCount drops all but recent", func(t *testing.T) {
		got := applyDropEarliestN(history, -1, 3)
		if len(got) != 3 {
			t.Fatalf("len(got) = %d, want 3", len(got))
		}
		want := []string{"3", "4", "5"}
		for i, w := range want {
			if got[i].Content != w {
				t.Errorf("got[%d].Content = %q, want %q", i, got[i].Content, w)
			}
		}
	})
}

func TestApplySummarizeAndReplace(t *testing.T) {
	history := []models.Message{
		{Role: models.RoleUser, Content: "anchor1"},
		{Role: models.RoleAssistant, Content: "middle1"},
		{Role: models.RoleUser, Content: "middle2"},
		{Role: models.RoleAssistant, Content: "recent1"},
		{Role: models.RoleUser, Content: "recent2"},
	}

	summary := "summary"

	got := applySummarizeAndReplace(history, 1, 2, func(msgs []models.Message) string {
		if len(msgs) != 2 {
			t.Errorf("summarize called with %d messages, want 2", len(msgs))
		}
		return summary
	})

	if len(got) != 4 {
		t.Fatalf("len(got) = %d, want 4", len(got))
	}
	if got[0].Content != "anchor1" {
		t.Errorf("got[0].Content = %q, want anchor1", got[0].Content)
	}
	if got[1].Role != models.RoleSystem || got[1].Content != summary {
		t.Errorf("got[1] = {role=%s, content=%q}, want system summary", got[1].Role, got[1].Content)
	}
	wantRecent := []string{"recent1", "recent2"}
	for i, w := range wantRecent {
		if got[2+i].Content != w {
			t.Errorf("got[%d].Content = %q, want %q", 2+i, got[2+i].Content, w)
		}
	}
}

func TestBoundedRatioStillWorks(t *testing.T) {
	cases := []struct {
		used, max int
		want      float64
	}{
		{0, 100, 0},
		{50, 100, 0.5},
		{100, 100, 1},
		{200, 100, 1},
		{10, 0, 0},
	}

	for _, tt := range cases {
		got := boundedRatio(tt.used, tt.max)
		if got != tt.want {
			t.Errorf("boundedRatio(%d, %d) = %v, want %v", tt.used, tt.max, got, tt.want)
		}
	}
}

// recordingMockLLMClient 记录 Chat 入参并支持控制可用性，用于压缩摘要测试。
type recordingMockLLMClient struct {
	chatReply    string
	chatErr      error
	available    bool
	lastMessages []models.Message
}

func (m *recordingMockLLMClient) Chat(_ context.Context, messages []models.Message) (string, error) {
	m.lastMessages = messages
	return m.chatReply, m.chatErr
}

func (m *recordingMockLLMClient) StreamChat(_ context.Context, _ []models.Message, _ func(string)) (*models.TokenUsage, error) {
	return nil, nil
}

func (m *recordingMockLLMClient) CheckAvailability(_ context.Context) (bool, string) {
	return m.available, ""
}

var _ port.LLMClient = (*recordingMockLLMClient)(nil)

// cloudProviderStore 按 ID 返回固定 APIHost 的 provider。
type cloudProviderStore struct {
	apiHost string
}

func (m *cloudProviderStore) Create(_ context.Context, _ *models.ProviderConfig) error { return nil }
func (m *cloudProviderStore) Update(_ context.Context, _ *models.ProviderConfig) error { return nil }
func (m *cloudProviderStore) Delete(_ context.Context, _ string) error                 { return nil }
func (m *cloudProviderStore) List(_ context.Context) ([]*models.ProviderConfig, error) {
	return nil, nil
}
func (m *cloudProviderStore) Get(_ context.Context, id string) (*models.ProviderConfig, error) {
	return &models.ProviderConfig{ID: id, APIHost: m.apiHost, ModelID: "test-model"}, nil
}

var _ port.ProviderStore = (*cloudProviderStore)(nil)

// transformMockDeidentifier 将手机号替换为占位符，验证云端摘要路径的脱敏行为。
type transformMockDeidentifier struct{}

func (m *transformMockDeidentifier) Execute(_ context.Context, text string) (models.DeidentifyResult, error) {
	phone := "13800138000"
	placeholder := "{{PHONE_1}}"
	if strings.Contains(text, phone) {
		safe := strings.ReplaceAll(text, phone, placeholder)
		return models.DeidentifyResult{
			OriginalText: text,
			SafeText:     safe,
			Placeholder:  map[string]string{placeholder: phone},
		}, nil
	}
	return models.DeidentifyResult{OriginalText: text, SafeText: text}, nil
}

var _ Deidentifier = (*transformMockDeidentifier)(nil)

// newTestCompressionService 构建注入测试 mock 的 CompressionService。
func newTestCompressionService(t *testing.T, client port.LLMClient, store port.ProviderStore, deid Deidentifier) *CompressionService {
	t.Helper()
	estimator, _ := newTestContextEstimator(true)
	factory := &mockLLMClientFactory{client: client}
	return NewCompressionService(estimator, factory, store, nil, deid)
}

// TestApplyLLMSelfSummarize_Cloud_Deidentify 验证云端 provider 仅发送占位符，本地回填为原值。
func TestApplyLLMSelfSummarize_Cloud_Deidentify(t *testing.T) {
	client := &recordingMockLLMClient{
		chatReply: "摘要：用户电话 {{PHONE_1}}",
		available: true,
	}
	svc := newTestCompressionService(t, client, &cloudProviderStore{apiHost: "https://api.moonshot.cn"}, &transformMockDeidentifier{})

	history := []models.Message{
		{Role: models.RoleUser, Content: "你好"},
		{Role: models.RoleUser, Content: "我的手机号是 13800138000"},
		{Role: models.RoleAssistant, Content: "收到"},
	}

	compressed, fallback, err := svc.applyLLMSelfSummarize(context.Background(), history, "cloud", CompressionConfig{
		Strategy:    StrategyLLMSelfSummarize,
		AnchorCount: 1,
		RecentCount: 1,
	})
	require.NoError(t, err)
	require.False(t, fallback)

	// Chat 收到的拼接内容应只含占位符，不含原始手机号
	require.Len(t, client.lastMessages, 1)
	require.Contains(t, client.lastMessages[0].Content, "{{PHONE_1}}")
	require.NotContains(t, client.lastMessages[0].Content, "13800138000")

	// 回填后的摘要应还原为原手机号
	require.Len(t, compressed, 3)
	require.Equal(t, models.RoleSystem, compressed[1].Role)
	require.Contains(t, compressed[1].Content, "13800138000")
}

// TestApplyLLMSelfSummarize_Loopback_SkipDeid 验证本地回环 provider 直发不脱敏。
func TestApplyLLMSelfSummarize_Loopback_SkipDeid(t *testing.T) {
	client := &recordingMockLLMClient{
		chatReply: "本地摘要",
		available: true,
	}
	deid := &transformMockDeidentifier{}
	svc := newTestCompressionService(t, client, &cloudProviderStore{apiHost: "http://localhost:11434"}, deid)

	history := []models.Message{
		{Role: models.RoleUser, Content: "你好"},
		{Role: models.RoleUser, Content: "我的手机号是 13800138000"},
		{Role: models.RoleAssistant, Content: "收到"},
	}

	compressed, fallback, err := svc.applyLLMSelfSummarize(context.Background(), history, "local", CompressionConfig{
		Strategy:    StrategyLLMSelfSummarize,
		AnchorCount: 1,
		RecentCount: 1,
	})
	require.NoError(t, err)
	require.False(t, fallback)

	require.Len(t, client.lastMessages, 1)
	require.Contains(t, client.lastMessages[0].Content, "13800138000")
	require.NotContains(t, client.lastMessages[0].Content, "{{PHONE_1}}")

	require.Len(t, compressed, 3)
	require.Contains(t, compressed[1].Content, "本地摘要")
}

// TestApplyLLMSelfSummarize_DeidFailure_Fallback 验证脱敏失败回退非模型策略且不外发。
func TestApplyLLMSelfSummarize_DeidFailure_Fallback(t *testing.T) {
	client := &recordingMockLLMClient{
		chatReply: "不应收到",
		available: true,
	}
	deid := &mockDeidentifier{err: fmt.Errorf("pipeline unavailable")}
	svc := newTestCompressionService(t, client, &cloudProviderStore{apiHost: "https://api.moonshot.cn"}, deid)

	history := []models.Message{
		{Role: models.RoleUser, Content: "你好"},
		{Role: models.RoleUser, Content: "我的手机号是 13800138000"},
		{Role: models.RoleAssistant, Content: "收到"},
	}

	compressed, fallback, err := svc.applyLLMSelfSummarize(context.Background(), history, "cloud", CompressionConfig{
		Strategy:    StrategyLLMSelfSummarize,
		AnchorCount: 1,
		RecentCount: 1,
	})
	require.NoError(t, err)
	require.True(t, fallback)
	require.Len(t, client.lastMessages, 0)
	require.Len(t, compressed, 3)
}

// TestApplyLLMSelfSummarize_Unavailable_Fallback 验证模型不可用回退非模型策略。
func TestApplyLLMSelfSummarize_Unavailable_Fallback(t *testing.T) {
	client := &recordingMockLLMClient{
		chatReply: "不应收到",
		available: false,
	}
	svc := newTestCompressionService(t, client, &cloudProviderStore{apiHost: "https://api.moonshot.cn"}, &mockDeidentifier{})

	history := []models.Message{
		{Role: models.RoleUser, Content: "你好"},
		{Role: models.RoleUser, Content: "中间内容"},
		{Role: models.RoleAssistant, Content: "收到"},
	}

	compressed, fallback, err := svc.applyLLMSelfSummarize(context.Background(), history, "cloud", CompressionConfig{
		Strategy:    StrategyLLMSelfSummarize,
		AnchorCount: 1,
		RecentCount: 1,
	})
	require.NoError(t, err)
	require.True(t, fallback)
	require.Len(t, client.lastMessages, 0)
	require.Len(t, compressed, 3)
}

// TestApplyLLMSelfSummarize_EmptySummary_Fallback 验证摘要为空时回退非模型策略。
func TestApplyLLMSelfSummarize_EmptySummary_Fallback(t *testing.T) {
	client := &recordingMockLLMClient{
		chatReply: "   ",
		available: true,
	}
	svc := newTestCompressionService(t, client, &cloudProviderStore{apiHost: "https://api.moonshot.cn"}, &mockDeidentifier{})

	history := []models.Message{
		{Role: models.RoleUser, Content: "你好"},
		{Role: models.RoleUser, Content: "中间内容"},
		{Role: models.RoleAssistant, Content: "收到"},
	}

	compressed, fallback, err := svc.applyLLMSelfSummarize(context.Background(), history, "cloud", CompressionConfig{
		Strategy:    StrategyLLMSelfSummarize,
		AnchorCount: 1,
		RecentCount: 1,
	})
	require.NoError(t, err)
	require.True(t, fallback)
	require.Len(t, compressed, 3)
}

// TestCompressMessages_SummarizeAndReplace_Reduces 验证本地摘要策略能减少 token。
func TestCompressMessages_SummarizeAndReplace_Reduces(t *testing.T) {
	svc := newTestCompressionService(t, &recordingMockLLMClient{available: true}, &cloudProviderStore{apiHost: "https://api.example.com"}, &mockDeidentifier{})

	history := make([]models.Message, 0, 10)
	for i := 0; i < 10; i++ {
		history = append(history, models.Message{Role: models.RoleUser, Content: strings.Repeat("x", 100)})
	}

	res, err := svc.CompressMessages(context.Background(), history, "test-provider", "test-model", CompressionConfig{
		Strategy:    StrategySummarizeAndReplace,
		AnchorCount: 1,
		RecentCount: 2,
		DropN:       5,
	})
	require.NoError(t, err)
	require.False(t, res.FallbackOccurred)
	require.Equal(t, StrategySummarizeAndReplace, res.Strategy)
	require.Less(t, res.UsedAfter, res.UsedBefore)
}

// TestCompressMessages_FallbackToDrop 验证摘要未减量时回退删除策略。
func TestCompressMessages_FallbackToDrop(t *testing.T) {
	svc := newTestCompressionService(t, &recordingMockLLMClient{available: true}, &cloudProviderStore{apiHost: "https://api.example.com"}, &mockDeidentifier{})

	// 中间两条极短消息被摘要为一条消息后，摘要文本反而更长，触发 drop 回退
	history := []models.Message{
		{Role: models.RoleUser, Content: "anchor"},
		{Role: models.RoleUser, Content: "a"},
		{Role: models.RoleAssistant, Content: "b"},
		{Role: models.RoleUser, Content: "recent"},
	}

	res, err := svc.CompressMessages(context.Background(), history, "test-provider", "test-model", CompressionConfig{
		Strategy:    StrategySummarizeAndReplace,
		AnchorCount: 1,
		RecentCount: 1,
		DropN:       2,
	})
	require.NoError(t, err)
	require.True(t, res.FallbackOccurred)
	require.Equal(t, StrategyDropEarliestN, res.Strategy)
	require.Less(t, res.UsedAfter, res.UsedBefore)
}

// TestCompressMessages_StillNotReduced_Error 验证回退后仍未减量时返回错误且不返回消息。
func TestCompressMessages_StillNotReduced_Error(t *testing.T) {
	svc := newTestCompressionService(t, &recordingMockLLMClient{available: true}, &cloudProviderStore{apiHost: "https://api.example.com"}, &mockDeidentifier{})

	history := []models.Message{
		{Role: models.RoleUser, Content: "a"},
	}

	res, err := svc.CompressMessages(context.Background(), history, "test-provider", "test-model", CompressionConfig{
		Strategy:    StrategySummarizeAndReplace,
		AnchorCount: 0,
		RecentCount: 0,
		DropN:       0,
	})
	require.Error(t, err)
	require.Empty(t, res.Messages)
	require.Contains(t, err.Error(), "compression did not reduce")
}

// TestCompressMessages_SmallHistory_NoOp 验证历史较短时直接返回原历史。
func TestCompressMessages_SmallHistory_NoOp(t *testing.T) {
	svc := newTestCompressionService(t, &recordingMockLLMClient{available: true}, &cloudProviderStore{apiHost: "https://api.example.com"}, &mockDeidentifier{})

	history := []models.Message{
		{Role: models.RoleUser, Content: "a"},
		{Role: models.RoleAssistant, Content: "b"},
	}

	res, err := svc.CompressMessages(context.Background(), history, "test-provider", "test-model", CompressionConfig{
		Strategy:    StrategySummarizeAndReplace,
		AnchorCount: 1,
		RecentCount: 1,
		DropN:       5,
	})
	require.NoError(t, err)
	require.Equal(t, history, res.Messages)
}

// mockMessageRepo 模拟 MessageRepository，支持事务内失败注入与提交记录。
type mockMessageRepo struct {
	committedSaved   []*entity.Message
	committedDeleted []string
	failAfterN       int
	calls            int
}

func (m *mockMessageRepo) Save(_ context.Context, _ models.ConversationID, msg *entity.Message) error {
	return nil
}

func (m *mockMessageRepo) ListByConversation(_ context.Context, _ models.ConversationID, _ string, _ int) ([]*entity.Message, string, error) {
	return nil, "", nil
}

func (m *mockMessageRepo) SoftDelete(_ context.Context, _ string) error {
	return nil
}

func (m *mockMessageRepo) Restore(_ context.Context, _ string) error {
	return nil
}

func (m *mockMessageRepo) WithTx(_ context.Context, fn func(tx port.MessageRepository) error) error {
	tx := &mockMessageRepoTx{
		parent:     m,
		failAfterN: m.failAfterN,
		calls:      &m.calls,
	}
	if err := fn(tx); err != nil {
		return err
	}
	m.committedSaved = append(m.committedSaved, tx.saved...)
	m.committedDeleted = append(m.committedDeleted, tx.deleted...)
	return nil
}

var _ port.MessageRepository = (*mockMessageRepo)(nil)

// mockMessageRepoTx 是 mockMessageRepo 的事务内视图，失败时修改不提交到 parent。
type mockMessageRepoTx struct {
	parent     *mockMessageRepo
	saved      []*entity.Message
	deleted    []string
	failAfterN int
	calls      *int
}

func (m *mockMessageRepoTx) Save(_ context.Context, _ models.ConversationID, msg *entity.Message) error {
	m.saved = append(m.saved, msg)
	return nil
}

func (m *mockMessageRepoTx) ListByConversation(_ context.Context, _ models.ConversationID, _ string, _ int) ([]*entity.Message, string, error) {
	return nil, "", fmt.Errorf("not supported inside transaction")
}

func (m *mockMessageRepoTx) SoftDelete(_ context.Context, id string) error {
	*m.calls++
	if m.failAfterN > 0 && *m.calls >= m.failAfterN {
		return fmt.Errorf("injected soft delete failure")
	}
	m.deleted = append(m.deleted, id)
	return nil
}

func (m *mockMessageRepoTx) Restore(_ context.Context, _ string) error {
	return nil
}

func (m *mockMessageRepoTx) WithTx(_ context.Context, fn func(tx port.MessageRepository) error) error {
	return fn(m)
}

var _ port.MessageRepository = (*mockMessageRepoTx)(nil)

// TestPersist_SummarizeAndReplace_Atomic 验证摘要保存与软删在一次事务内完成。
func TestPersist_SummarizeAndReplace_Atomic(t *testing.T) {
	repo := &mockMessageRepo{}
	svc := NewCompressionService(nil, nil, nil, repo, nil)

	entities := []*entity.Message{
		{ID: "m1", Role: models.RoleUser, Content: "a"},
		{ID: "m2", Role: models.RoleUser, Content: "b"},
		{ID: "m3", Role: models.RoleAssistant, Content: "c"},
	}
	before := []models.Message{
		{Role: models.RoleUser, Content: "a"},
		{Role: models.RoleUser, Content: "b"},
		{Role: models.RoleAssistant, Content: "c"},
	}
	after := []models.Message{
		{Role: models.RoleUser, Content: "a"},
		{Role: models.RoleSystem, Content: "summary"},
		{Role: models.RoleAssistant, Content: "c"},
	}

	err := svc.persist(context.Background(), "conv-1", entities, before, after, StrategySummarizeAndReplace)
	require.NoError(t, err)

	require.Len(t, repo.committedSaved, 1)
	require.Equal(t, models.RoleSystem, repo.committedSaved[0].Role)
	require.Equal(t, "summary", repo.committedSaved[0].Content)
	require.Equal(t, []string{"m2"}, repo.committedDeleted)
}

// TestPersist_DropEarliestN_Atomic 验证删除最早 N 条与消息数量一致。
func TestPersist_DropEarliestN_Atomic(t *testing.T) {
	repo := &mockMessageRepo{}
	svc := NewCompressionService(nil, nil, nil, repo, nil)

	entities := []*entity.Message{
		{ID: "m1", Role: models.RoleUser, Content: "a"},
		{ID: "m2", Role: models.RoleUser, Content: "b"},
		{ID: "m3", Role: models.RoleAssistant, Content: "c"},
		{ID: "m4", Role: models.RoleUser, Content: "d"},
	}
	before := []models.Message{
		{Role: models.RoleUser, Content: "a"},
		{Role: models.RoleUser, Content: "b"},
		{Role: models.RoleAssistant, Content: "c"},
		{Role: models.RoleUser, Content: "d"},
	}
	after := []models.Message{
		{Role: models.RoleAssistant, Content: "c"},
		{Role: models.RoleUser, Content: "d"},
	}

	err := svc.persist(context.Background(), "conv-1", entities, before, after, StrategyDropEarliestN)
	require.NoError(t, err)

	require.Empty(t, repo.committedSaved)
	require.Equal(t, []string{"m1", "m2"}, repo.committedDeleted)
}

// TestPersist_Failure_Rollback 验证软删中途失败时整体回滚，无净变更。
func TestPersist_Failure_Rollback(t *testing.T) {
	repo := &mockMessageRepo{failAfterN: 1}
	svc := NewCompressionService(nil, nil, nil, repo, nil)

	entities := []*entity.Message{
		{ID: "m1", Role: models.RoleUser, Content: "a"},
		{ID: "m2", Role: models.RoleUser, Content: "b"},
		{ID: "m3", Role: models.RoleAssistant, Content: "c"},
	}
	before := []models.Message{
		{Role: models.RoleUser, Content: "a"},
		{Role: models.RoleUser, Content: "b"},
		{Role: models.RoleAssistant, Content: "c"},
	}
	after := []models.Message{
		{Role: models.RoleUser, Content: "a"},
		{Role: models.RoleSystem, Content: "summary"},
		{Role: models.RoleAssistant, Content: "c"},
	}

	err := svc.persist(context.Background(), "conv-1", entities, before, after, StrategySummarizeAndReplace)
	require.Error(t, err)

	require.Empty(t, repo.committedSaved)
	require.Empty(t, repo.committedDeleted)
}

// TestSummarizeDeterministic_RuneBoundary 验证中文长句按 rune 截断，不截出半个字符。
func TestSummarizeDeterministic_RuneBoundary(t *testing.T) {
	long := strings.Repeat("中", 80)
	msgs := []models.Message{{Role: models.RoleUser, Content: long}}
	summary := summarizeDeterministic(msgs)
	require.True(t, utf8.ValidString(summary))
	require.True(t, strings.HasSuffix(summary, "…"))
	require.Contains(t, summary, "· 用户:")
}

// TestSummarizeDeterministic_SentenceDelimiter 验证按句子分隔符截断。
func TestSummarizeDeterministic_SentenceDelimiter(t *testing.T) {
	msgs := []models.Message{{Role: models.RoleAssistant, Content: "第一句。第二句很长很长很长"}}
	summary := summarizeDeterministic(msgs)
	require.Contains(t, summary, "第一句。")
	require.NotContains(t, summary, "第二句")
}

// TestSummarizeDeterministic_EmptyAndWhitespace 验证空/纯空白内容被跳过。
func TestSummarizeDeterministic_EmptyAndWhitespace(t *testing.T) {
	msgs := []models.Message{
		{Role: models.RoleUser, Content: ""},
		{Role: models.RoleUser, Content: "   "},
		{Role: models.RoleAssistant, Content: "有效"},
	}
	summary := summarizeDeterministic(msgs)
	require.Contains(t, summary, "· 助手: 有效")
	require.NotContains(t, summary, "用户:")
}

// TestFirstSentence_MixedRunes 验证 CJK/ASCII/emoji 混合不 panic 且按 rune 截断。
func TestFirstSentence_MixedRunes(t *testing.T) {
	text := "hello世界🌍这是第二句"
	got := firstSentence([]rune(text), 8)
	require.True(t, utf8.ValidString(got))
	require.Equal(t, "hello世界🌍…", got)
}

// TestFirstSentence_NoDelimiterTruncatesWithEllipsis 验证无分隔符超长文本以省略号收尾。
func TestFirstSentence_NoDelimiterTruncatesWithEllipsis(t *testing.T) {
	text := strings.Repeat("a", 100)
	got := firstSentence([]rune(text), 10)
	require.Equal(t, strings.Repeat("a", 10)+"…", got)
}

// TestIsLoopbackProvider 验证本地回环 host 判断。
func TestIsLoopbackProvider(t *testing.T) {
	cases := []struct {
		apiHost string
		want    bool
	}{
		{"http://localhost:11434", true},
		{"http://127.0.0.1:11434", true},
		{"http://[::1]:11434", true},
		{"https://api.moonshot.cn", false},
		{"", false},
		{"not-a-url", false},
	}

	for _, tt := range cases {
		p := &models.ProviderConfig{APIHost: tt.apiHost}
		assert.Equal(t, tt.want, isLoopbackProvider(p), "apiHost=%q", tt.apiHost)
	}

	require.False(t, isLoopbackProvider(nil))
}

// modelProviderStore 返回携带指定模型列表的 provider。
type modelProviderStore struct {
	models []models.ProviderModel
}

func (m *modelProviderStore) Create(_ context.Context, _ *models.ProviderConfig) error { return nil }
func (m *modelProviderStore) Update(_ context.Context, _ *models.ProviderConfig) error { return nil }
func (m *modelProviderStore) Delete(_ context.Context, _ string) error                 { return nil }
func (m *modelProviderStore) List(_ context.Context) ([]*models.ProviderConfig, error) {
	return nil, nil
}
func (m *modelProviderStore) Get(_ context.Context, id string) (*models.ProviderConfig, error) {
	return &models.ProviderConfig{
		ID:      id,
		APIHost: "https://api.example.com",
		ModelID: "default-model",
		Models:  m.models,
	}, nil
}

var _ port.ProviderStore = (*modelProviderStore)(nil)

// TestTestModelAvailability_ModelNotFound 验证指定模型不存在时返回失败。
func TestTestModelAvailability_ModelNotFound(t *testing.T) {
	client := &recordingMockLLMClient{available: true}
	svc := newTestCompressionService(t, client, &modelProviderStore{
		models: []models.ProviderModel{{ID: "other-model"}},
	}, &mockDeidentifier{})

	available, err := svc.TestModelAvailability(context.Background(), "test-provider", "missing-model")
	require.Error(t, err)
	require.False(t, available)
	require.Contains(t, err.Error(), "not found")
}

// TestTestModelAvailability_Available 验证模型存在且可用时返回成功。
func TestTestModelAvailability_Available(t *testing.T) {
	client := &recordingMockLLMClient{available: true}
	svc := newTestCompressionService(t, client, &modelProviderStore{
		models: []models.ProviderModel{{ID: "target-model"}},
	}, &mockDeidentifier{})

	available, err := svc.TestModelAvailability(context.Background(), "test-provider", "target-model")
	require.NoError(t, err)
	require.True(t, available)
}

// TestTestModelAvailability_Unavailable 验证模型存在但服务不可用时返回失败。
func TestTestModelAvailability_Unavailable(t *testing.T) {
	client := &recordingMockLLMClient{available: false}
	svc := newTestCompressionService(t, client, &modelProviderStore{
		models: []models.ProviderModel{{ID: "target-model"}},
	}, &mockDeidentifier{})

	available, err := svc.TestModelAvailability(context.Background(), "test-provider", "target-model")
	require.NoError(t, err)
	require.False(t, available)
}

// TestModelExists 验证模型存在性判断覆盖 provider.ModelID 与 Models 列表。
func TestModelExists(t *testing.T) {
	assert.True(t, modelExists(&models.ProviderConfig{ModelID: "m1"}, "m1"))
	assert.True(t, modelExists(&models.ProviderConfig{Models: []models.ProviderModel{{ID: "m2"}}}, "m2"))
	assert.False(t, modelExists(&models.ProviderConfig{ModelID: "m1"}, "m2"))
	assert.False(t, modelExists(nil, "m1"))
	assert.False(t, modelExists(&models.ProviderConfig{ModelID: "m1"}, ""))
}

// seededMessageRepo 在 mockMessageRepo 基础上让 ListByConversation 返回预置实体，
// 用于覆盖 Compress 的完整 DB 加载 + 压缩 + 持久化路径。
type seededMessageRepo struct {
	*mockMessageRepo
	entities []*entity.Message
	listErr  error
}

func (r *seededMessageRepo) ListByConversation(_ context.Context, _ models.ConversationID, _ string, _ int) ([]*entity.Message, string, error) {
	if r.listErr != nil {
		return nil, "", r.listErr
	}
	// 返回副本，避免 reverseEntities 原地反转污染测试预置数据。
	out := make([]*entity.Message, len(r.entities))
	copy(out, r.entities)
	return out, "", nil
}

var _ port.MessageRepository = (*seededMessageRepo)(nil)

// newSeededCompressionService 构建带预置消息仓库的压缩服务。
func newSeededCompressionService(t *testing.T, repo port.MessageRepository) *CompressionService {
	t.Helper()
	estimator, _ := newTestContextEstimator(true)
	factory := &mockLLMClientFactory{client: &recordingMockLLMClient{available: true}}
	return NewCompressionService(estimator, factory, &cloudProviderStore{apiHost: "https://api.example.com"}, repo, &mockDeidentifier{})
}

// TestCompress_FullPath_Persists 验证 Compress 从 DB 加载、压缩并落库摘要与软删。
func TestCompress_FullPath_Persists(t *testing.T) {
	// ListByConversation 返回最新在前（倒序），Compress 内部会 reverse 为时间正序。
	entities := make([]*entity.Message, 0, 10)
	for i := 9; i >= 0; i-- {
		entities = append(entities, &entity.Message{
			ID:      fmt.Sprintf("m%d", i),
			Role:    models.RoleUser,
			Content: strings.Repeat("x", 100),
		})
	}
	repo := &seededMessageRepo{mockMessageRepo: &mockMessageRepo{}, entities: entities}
	svc := newSeededCompressionService(t, repo)

	res, err := svc.Compress(context.Background(), "conv-1", "test-provider", "test-model", CompressionConfig{
		Strategy:    StrategySummarizeAndReplace,
		AnchorCount: 1,
		RecentCount: 2,
		DropN:       5,
	})
	require.NoError(t, err)
	require.Equal(t, StrategySummarizeAndReplace, res.Strategy)
	require.Less(t, res.UsedAfter, res.UsedBefore)

	// 应保存一条摘要并软删中间消息（10 条中除去 1 anchor + 2 recent = 7 条被删）。
	require.Len(t, repo.committedSaved, 1)
	require.Equal(t, models.RoleSystem, repo.committedSaved[0].Role)
	require.Len(t, repo.committedDeleted, 7)
}

// TestCompress_ListError 验证 DB 加载失败时返回包装错误。
func TestCompress_ListError(t *testing.T) {
	repo := &seededMessageRepo{mockMessageRepo: &mockMessageRepo{}, listErr: fmt.Errorf("db down")}
	svc := newSeededCompressionService(t, repo)

	_, err := svc.Compress(context.Background(), "conv-1", "p", "m", CompressionConfig{
		Strategy:    StrategySummarizeAndReplace,
		AnchorCount: 1,
		RecentCount: 2,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to list messages")
}

// TestApplyStrategy 覆盖策略分发的各分支与非法策略错误。
func TestApplyStrategy(t *testing.T) {
	svc := newTestCompressionService(t, &recordingMockLLMClient{available: true}, &cloudProviderStore{apiHost: "https://api.example.com"}, &mockDeidentifier{})
	history := []models.Message{
		{Role: models.RoleUser, Content: "a"},
		{Role: models.RoleAssistant, Content: "b"},
		{Role: models.RoleUser, Content: "c"},
		{Role: models.RoleAssistant, Content: "d"},
	}

	t.Run("drop earliest n", func(t *testing.T) {
		got, kind, fallback, err := svc.applyStrategy(context.Background(), history, "p", CompressionConfig{
			Strategy: StrategyDropEarliestN, DropN: 2, RecentCount: 1,
		})
		require.NoError(t, err)
		assert.Equal(t, StrategyDropEarliestN, kind)
		assert.False(t, fallback)
		assert.Len(t, got, 2)
	})

	t.Run("summarize and replace", func(t *testing.T) {
		got, kind, fallback, err := svc.applyStrategy(context.Background(), history, "p", CompressionConfig{
			Strategy: StrategySummarizeAndReplace, AnchorCount: 1, RecentCount: 1,
		})
		require.NoError(t, err)
		assert.Equal(t, StrategySummarizeAndReplace, kind)
		assert.False(t, fallback)
		assert.Equal(t, models.RoleSystem, got[1].Role)
	})

	t.Run("llm self summarize dispatches", func(t *testing.T) {
		client := &recordingMockLLMClient{available: true, chatReply: "摘要内容"}
		llmSvc := newTestCompressionService(t, client, &cloudProviderStore{apiHost: "https://api.example.com"}, &mockDeidentifier{})
		got, kind, fallback, err := llmSvc.applyStrategy(context.Background(), history, "p", CompressionConfig{
			Strategy: StrategyLLMSelfSummarize, AnchorCount: 1, RecentCount: 1,
		})
		require.NoError(t, err)
		assert.Equal(t, StrategyLLMSelfSummarize, kind)
		assert.False(t, fallback)
		assert.Equal(t, models.RoleSystem, got[1].Role)
		assert.Contains(t, got[1].Content, "摘要内容")
	})

	t.Run("unsupported strategy errors", func(t *testing.T) {
		_, _, _, err := svc.applyStrategy(context.Background(), history, "p", CompressionConfig{
			Strategy: CompressionStrategyKind("bogus"),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported compression strategy")
	})
}

// TestSliceMiddle 覆盖中间块切片的边界钳制逻辑。
func TestSliceMiddle(t *testing.T) {
	history := []models.Message{
		{Content: "0"}, {Content: "1"}, {Content: "2"}, {Content: "3"}, {Content: "4"},
	}

	t.Run("normal middle", func(t *testing.T) {
		mid := sliceMiddle(history, 1, 2)
		require.Len(t, mid, 2)
		assert.Equal(t, "1", mid[0].Content)
		assert.Equal(t, "2", mid[1].Content)
	})

	t.Run("negative counts clamp to zero", func(t *testing.T) {
		mid := sliceMiddle(history, -1, -1)
		assert.Len(t, mid, len(history))
	})

	t.Run("anchor exceeds length yields empty", func(t *testing.T) {
		mid := sliceMiddle(history, 100, 0)
		assert.Empty(t, mid)
	})

	t.Run("recent exceeds length yields empty", func(t *testing.T) {
		mid := sliceMiddle(history, 0, 100)
		assert.Empty(t, mid)
	})
}

// TestApplyLLMSelfSummarize_EmptyMiddle 验证中间块为空时原样返回、不调用模型。
func TestApplyLLMSelfSummarize_EmptyMiddle(t *testing.T) {
	client := &recordingMockLLMClient{available: true, chatReply: "unused"}
	svc := newTestCompressionService(t, client, &cloudProviderStore{apiHost: "https://api.example.com"}, &mockDeidentifier{})

	history := []models.Message{
		{Role: models.RoleUser, Content: "a"},
		{Role: models.RoleAssistant, Content: "b"},
	}
	// anchor=1, recent=1 => 中间块为空
	compressed, fallback, err := svc.applyLLMSelfSummarize(context.Background(), history, "cloud", CompressionConfig{
		Strategy: StrategyLLMSelfSummarize, AnchorCount: 1, RecentCount: 1,
	})
	require.NoError(t, err)
	assert.False(t, fallback)
	assert.Equal(t, history, compressed)
	assert.Empty(t, client.lastMessages, "空中间块不应调用模型")
}
