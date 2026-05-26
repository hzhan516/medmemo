package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/hzhan516/medmemo/internal/domain/entity"
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
	// source_msg_ids 应被正确关联
	require.Len(t, facts[0].SourceMsgIDs, 2)
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
}

func (m *mockLLMForFactExtraction) Chat(ctx context.Context, messages []string) (string, error) {
	return m.response, nil
}

// mockDialogueRepo 用于测试
type mockDialogueRepo struct{}

func (m *mockDialogueRepo) Insert(ctx context.Context, d *entity.RawDialogue) error         { return nil }
func (m *mockDialogueRepo) InsertBatch(ctx context.Context, dialogues []*entity.RawDialogue) error { return nil }
func (m *mockDialogueRepo) GetBySession(ctx context.Context, sessionID string, offset, limit int) ([]*entity.RawDialogue, error) {
	return nil, nil
}
func (m *mockDialogueRepo) GetRecent(ctx context.Context, sessionID string, minutes int) ([]*entity.RawDialogue, error) {
	return nil, nil
}
func (m *mockDialogueRepo) GetUnprocessed(ctx context.Context, limit int) ([]*entity.RawDialogue, error) {
	return nil, nil
}
func (m *mockDialogueRepo) MarkProcessing(ctx context.Context, messageID string) error  { return nil }
func (m *mockDialogueRepo) MarkProcessed(ctx context.Context, messageID string) error    { return nil }
func (m *mockDialogueRepo) MarkFailed(ctx context.Context, messageID string) error       { return nil }

// mockFactRepo 用于测试
type mockFactRepo struct{}

func (m *mockFactRepo) Save(ctx context.Context, f *entity.ExtractedFact) error                    { return nil }
func (m *mockFactRepo) GetByID(ctx context.Context, factID string) (*entity.ExtractedFact, error)  { return nil, nil }
func (m *mockFactRepo) ListByStatus(ctx context.Context, status entity.FactStatus, offset, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepo) ListPending(ctx context.Context, offset, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *mockFactRepo) UpdateStatus(ctx context.Context, factID string, status entity.FactStatus) error { return nil }
func (m *mockFactRepo) Delete(ctx context.Context, factID string) error                             { return nil }
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
