package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFactExtractor_ParseFacts(t *testing.T) {
	extractor := NewFactExtractor(&mockLLMForFactExtraction{
		response: `[{"subject":"用户","predicate":"患有","object":"头痛","confidence":0.9}]`,
	})

	facts, err := extractor.ParseFacts("用户说头疼")
	require.NoError(t, err)
	require.Len(t, facts, 1)
	assert.Equal(t, "用户", facts[0].Subject)
	assert.Equal(t, "患有", facts[0].Predicate)
	assert.Equal(t, "头痛", facts[0].Object)
	assert.Equal(t, 0.9, facts[0].Confidence)
}

func TestFactExtractor_ParseFacts_InvalidJSON(t *testing.T) {
	extractor := NewFactExtractor(&mockLLMForFactExtraction{
		response: "不是 JSON",
	})

	_, err := extractor.ParseFacts("用户说头疼")
	assert.Error(t, err)
}

func TestFactExtractor_ParseFacts_EmptyArray(t *testing.T) {
	extractor := NewFactExtractor(&mockLLMForFactExtraction{
		response: "[]",
	})

	facts, err := extractor.ParseFacts("随便聊聊")
	require.NoError(t, err)
	assert.Len(t, facts, 0)
}

func TestFactExtractor_ParseFacts_MultipleFacts(t *testing.T) {
	extractor := NewFactExtractor(&mockLLMForFactExtraction{
		response: `[{"subject":"用户","predicate":"患有","object":"头痛","confidence":0.9},{"subject":"用户","predicate":"服用","object":"阿司匹林","confidence":0.8}]`,
	})

	facts, err := extractor.ParseFacts("用户头疼，吃了阿司匹林")
	require.NoError(t, err)
	require.Len(t, facts, 2)
	assert.Equal(t, "服用", facts[1].Predicate)
}

func TestFactExtractor_ParseFacts_DeduplicatesSameBatchTriples(t *testing.T) {
	extractor := NewFactExtractor(&mockLLMForFactExtraction{
		response: `[{"subject":"用户","predicate":"患有","object":"头痛","confidence":0.9},{"subject":" 用户 ","predicate":"患有","object":"头痛","confidence":0.8}]`,
	})

	facts, err := extractor.ParseFacts("用户头疼")
	require.NoError(t, err)
	require.Len(t, facts, 1)
	assert.Equal(t, "用户", facts[0].Subject)
	assert.Equal(t, "头痛", facts[0].Object)
	assert.Equal(t, 0.9, facts[0].Confidence)
}

func TestFactExtractor_ParseFacts_MissingFields(t *testing.T) {
	// LLM 返回缺少字段的 JSON，应被过滤
	extractor := NewFactExtractor(&mockLLMForFactExtraction{
		response: `[{"subject":"用户","predicate":"","object":"头痛","confidence":0.9}]`,
	})

	facts, err := extractor.ParseFacts("用户说头疼")
	require.NoError(t, err)
	assert.Len(t, facts, 0, "facts with empty predicate should be filtered out")
}

func TestFactExtractor_RateLimiter(t *testing.T) {
	extractor := NewFactExtractor(&mockLLMForFactExtraction{
		response: `[{"subject":"用户","predicate":"患有","object":"头痛","confidence":0.9}]`,
	})
	extractor.rateLimit = 5 // 5 per minute for testing

	// 快速调用 6 次，第 6 次应被限速
	for i := 0; i < 5; i++ {
		_, err := extractor.ParseFacts("测试")
		require.NoError(t, err)
	}

	_, err := extractor.ParseFacts("测试")
	assert.ErrorIs(t, err, ErrRateLimited)
}

func TestFactExtractorWorker_Lifecycle(t *testing.T) {
	extractor := NewFactExtractor(&mockLLMForFactExtraction{
		response: `[{"subject":"用户","predicate":"患有","object":"头痛","confidence":0.9}]`,
	})
	worker := NewFactExtractorWorker(extractor, &mockDialogueRepo{}, &mockFactRepo{})

	ctx, cancel := context.WithCancel(context.Background())
	worker.Start(ctx)

	// 等待一小段时间让 worker 启动
	time.Sleep(50 * time.Millisecond)

	// 停止 worker
	cancel()
	worker.Wait()
}

func TestFactExtractor_Extract(t *testing.T) {
	extractor := NewFactExtractor(&mockLLMForFactExtraction{
		response: `[{"subject":"用户","predicate":"患有","object":"头痛","confidence":0.9}]`,
	})

	dialogues := []*entity.RawDialogue{
		entity.NewRawDialogue("sess_1", entity.RoleUser, "我头很痛", "kimi"),
		entity.NewRawDialogue("sess_1", entity.RoleAssistant, "了解了，请告诉我更多细节", "kimi"),
	}

	facts, err := extractor.Extract(context.Background(), dialogues)
	require.NoError(t, err)
	require.Len(t, facts, 1)
	assert.Equal(t, "用户", facts[0].Subject)
	assert.Equal(t, "患有", facts[0].Predicate)
	assert.Equal(t, "头痛", facts[0].Object)
	// source_msg_ids 只关联用户消息（RoleUser），排除 AI 回复
	require.Len(t, facts[0].SourceMsgIDs, 1)
}

func TestFactExtractor_Extract_RateLimited(t *testing.T) {
	extractor := NewFactExtractor(&mockLLMForFactExtraction{
		response: `[{"subject":"用户","predicate":"患有","object":"头痛","confidence":0.9}]`,
	})
	extractor.rateLimit = 1

	// 第一次调用
	_, err := extractor.Extract(context.Background(), []*entity.RawDialogue{
		entity.NewRawDialogue("sess_1", entity.RoleUser, "测试", "kimi"),
	})
	require.NoError(t, err)

	// 第二次调用应被限速
	_, err = extractor.Extract(context.Background(), []*entity.RawDialogue{
		entity.NewRawDialogue("sess_1", entity.RoleUser, "测试2", "kimi"),
	})
	assert.ErrorIs(t, err, ErrRateLimited)
}

// mockLLMForFactExtraction 用于测试的 mock LLM
type mockLLMForFactExtraction struct {
	response string
	err      error
}

func (m *mockLLMForFactExtraction) Chat(ctx context.Context, messages []string) (string, error) {
	return m.response, m.err
}

// mockDialogueRepo 用于测试
type mockDialogueRepo struct{}

func (m *mockDialogueRepo) Insert(ctx context.Context, d *entity.RawDialogue) error { return nil }
func (m *mockDialogueRepo) InsertBatch(ctx context.Context, dialogues []*entity.RawDialogue) error {
	return nil
}
func (m *mockDialogueRepo) GetBySession(ctx context.Context, sessionID string, offset, limit int) ([]*entity.RawDialogue, error) {
	return nil, nil
}
func (m *mockDialogueRepo) GetRecent(ctx context.Context, sessionID string, minutes int) ([]*entity.RawDialogue, error) {
	return nil, nil
}
func (m *mockDialogueRepo) GetUnprocessed(ctx context.Context, limit int) ([]*entity.RawDialogue, error) {
	return nil, nil
}
func (m *mockDialogueRepo) MarkProcessing(ctx context.Context, messageID string) error { return nil }
func (m *mockDialogueRepo) MarkProcessed(ctx context.Context, messageID string) error  { return nil }
func (m *mockDialogueRepo) MarkFailed(ctx context.Context, messageID string) error     { return nil }

// mockFactRepo 用于测试
type mockFactRepo struct{}

func (m *mockFactRepo) Save(ctx context.Context, f *entity.ExtractedFact) error { return nil }
func (m *mockFactRepo) GetByID(ctx context.Context, factID string) (*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepo) ListByStatus(ctx context.Context, status entity.FactStatus, offset, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepo) ListPending(ctx context.Context, offset, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepo) UpdateStatus(ctx context.Context, factID string, status entity.FactStatus) error {
	return nil
}
func (m *mockFactRepo) Delete(ctx context.Context, factID string) error { return nil }
func (m *mockFactRepo) GetStats(ctx context.Context) (total, approved, rejected, pending int64, err error) {
	return 0, 0, 0, 0, nil
}
func (m *mockFactRepo) ListAllSubjects(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (m *mockFactRepo) FindBySubject(ctx context.Context, subject string) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepo) FindBySession(ctx context.Context, sessionID string) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepo) FindApprovedByPredicates(ctx context.Context, subject string, predicates []string, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepo) FindLatestApprovedByPredicates(ctx context.Context, subject string, predicates []string) (*entity.ExtractedFact, error) {
	return nil, entity.ErrFactNotFound
}

func (m *mockFactRepo) CountApprovedFactsNeedingEmbedding(ctx context.Context, targetVersion string) (int64, error) {
	return 0, nil
}

func (m *mockFactRepo) ListApprovedFactsNeedingEmbedding(ctx context.Context, targetVersion string, lastCreatedAt time.Time, lastFactID string, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}

func TestFactExtractor_ParseFacts_rateLimitExceeded(t *testing.T) {
	// 直接构造 FactExtractor，设置较低的速率限制以便快速触发限流
	extractor := &FactExtractor{
		llm:       &mockLLMForFactExtraction{response: `[]`},
		rateLimit: 2,
		callTimes: []time.Time{},
	}

	// 前两次调用应在速率限制内
	_, err := extractor.ParseFacts("第一次")
	require.NoError(t, err)
	_, err = extractor.ParseFacts("第二次")
	require.NoError(t, err)

	// 第三次调用超出限制，应返回包含 rate limit 的错误
	_, err = extractor.ParseFacts("第三次")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit")
}

func TestFactExtractor_ParseFacts_llmError(t *testing.T) {
	// 模拟 LLM 返回错误，验证 ParseFacts 是否正确包装并向上返回
	extractor := NewFactExtractor(&mockLLMForFactExtraction{
		err: fmt.Errorf("llm failed"),
	})

	_, err := extractor.ParseFacts("用户说头疼")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "llm chat failed")
}

func TestFactExtractor_Extract_llmError(t *testing.T) {
	// 模拟 LLM 返回错误，验证 Extract 是否正确包装并向上返回
	extractor := NewFactExtractor(&mockLLMForFactExtraction{
		err: fmt.Errorf("llm failed"),
	})

	dialogues := []*entity.RawDialogue{
		entity.NewRawDialogue("sess_1", entity.RoleUser, "我头很痛", "kimi"),
	}

	_, err := extractor.Extract(context.Background(), dialogues)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "llm extraction failed")
}

func TestFactExtractor_checkRateLimit(t *testing.T) {
	extractor := &FactExtractor{
		rateLimit: 2,
		callTimes: []time.Time{},
	}

	// 首次调用应成功
	err := extractor.checkRateLimit()
	require.NoError(t, err, "首次调用应在速率限制内")

	// 第二次调用应成功（当前仅 1 条记录，小于限制 2）
	err = extractor.checkRateLimit()
	require.NoError(t, err, "第二次调用仍应在速率限制内")

	// 第三次调用应失败（已达限制 2）
	err = extractor.checkRateLimit()
	require.ErrorIs(t, err, ErrRateLimited, "第三次调用应触发速率限制")

	// 将历史调用时间设置为超过 1 分钟前，模拟时间窗口过期
	oldTime := time.Now().Add(-2 * time.Minute)
	extractor.callTimes = []time.Time{oldTime}

	// 过期时间窗口被清除后，调用应再次成功
	err = extractor.checkRateLimit()
	require.NoError(t, err, "过期时间窗口清除后应恢复可用")
}
