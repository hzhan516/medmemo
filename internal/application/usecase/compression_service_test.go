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

func (m *cloudProviderStore) Create(_ context.Context, _ *models.ProviderConfig) error  { return nil }
func (m *cloudProviderStore) Update(_ context.Context, _ *models.ProviderConfig) error  { return nil }
func (m *cloudProviderStore) Delete(_ context.Context, _ string) error                  { return nil }
func (m *cloudProviderStore) List(_ context.Context) ([]*models.ProviderConfig, error)   { return nil, nil }
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


