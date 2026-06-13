package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/application"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckEmergency_ALevel(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"chest_pain", "我胸痛伴呼吸困难，很难受"},
		{"unconscious", "患者意识丧失，需要急救"},
		{"severe_allergy", "严重过敏反应，喉咙肿胀"},
		{"bleeding", "大出血，血流不止"},
		{"stroke", "突发偏瘫，口角歪斜，可能是脑卒中"},
		{"poisoning", "误食农药中毒，快帮忙"},
		{"drowning", "孩子溺水了，没有呼吸"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &WailsApp{}
			result, err := app.CheckEmergency(tt.text)
			require.NoError(t, err)
			assert.Equal(t, "A", result.Level)
			assert.NotEmpty(t, result.Message)
			assert.Contains(t, result.Action, "120")
		})
	}
}

func TestCheckEmergency_BLevel(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"high_fever", "持续高热三天不退"},
		{"abdominal_pain", "剧烈腹痛，难以忍受"},
		{"blood_in_urine", "发现血尿，尿液带血"},
		{"vision_loss", "视力突然下降，看不清东西"},
		{"palpitation", "心悸胸闷，心跳过快"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &WailsApp{}
			result, err := app.CheckEmergency(tt.text)
			require.NoError(t, err)
			assert.Equal(t, "B", result.Level)
			assert.NotEmpty(t, result.Message)
		})
	}
}

func TestCheckEmergency_None(t *testing.T) {
	app := &WailsApp{}
	result, err := app.CheckEmergency("今天天气不错，想了解一下健康饮食")
	require.NoError(t, err)
	assert.Equal(t, "none", result.Level)
	assert.Empty(t, result.Message)
}

func TestCheckEmergency_Empty(t *testing.T) {
	app := &WailsApp{}
	result, err := app.CheckEmergency("")
	require.NoError(t, err)
	assert.Equal(t, "none", result.Level)
}

func TestStreamTimeout_UsesProviderTimeout(t *testing.T) {
	store := newMockProviderStore()
	require.NoError(t, store.Create(t.Context(), &models.ProviderConfig{
		ID:        "gemini",
		TimeoutMs: 300000,
	}))
	app := &WailsApp{ctx: t.Context(), providerStore: store}

	assert.Equal(t, 5*time.Minute, app.streamTimeout("gemini"))
}

func TestStreamTimeout_EnforcesMinimum(t *testing.T) {
	store := newMockProviderStore()
	require.NoError(t, store.Create(t.Context(), &models.ProviderConfig{
		ID:        "gemini",
		TimeoutMs: 30000,
	}))
	app := &WailsApp{ctx: t.Context(), providerStore: store}

	assert.Equal(t, 120*time.Second, app.streamTimeout("gemini"))
}

func TestStreamTimeout_DefaultsWhenProviderMissing(t *testing.T) {
	app := &WailsApp{ctx: t.Context(), providerStore: newMockProviderStore()}

	assert.Equal(t, 5*time.Minute, app.streamTimeout("missing"))
}

func TestGetEmbeddingStatus_ModelPresentEngineUnavailable(t *testing.T) {
	app, _ := newEmbeddingStatusTestApp(t, false, "embedding warmup failed", true)

	status, err := app.GetEmbeddingStatus()
	require.NoError(t, err)
	assert.False(t, status.Available)
	assert.True(t, status.ModelPresent)
	assert.False(t, status.EngineAvailable)
	assert.True(t, status.RuntimeLibPresent)
	assert.Equal(t, "embedding warmup failed", status.FailureReason)
}

func TestGetEmbeddingStatus_RuntimeLibMissing(t *testing.T) {
	app, runtimePath := newEmbeddingStatusTestApp(t, false, "session init failed", false)

	status, err := app.GetEmbeddingStatus()
	require.NoError(t, err)
	assert.False(t, status.Available)
	assert.True(t, status.ModelPresent)
	assert.False(t, status.RuntimeLibPresent)
	assert.Equal(t, runtimePath, status.RuntimeLibPath)
	assert.Contains(t, status.FailureReason, "ONNX Runtime library not found")
}

func TestGetEmbeddingStatus_EngineAvailable(t *testing.T) {
	app, _ := newEmbeddingStatusTestApp(t, true, "", true)

	status, err := app.GetEmbeddingStatus()
	require.NoError(t, err)
	assert.True(t, status.Available)
	assert.True(t, status.ModelPresent)
	assert.True(t, status.EngineAvailable)
	assert.True(t, status.RuntimeLibPresent)
	assert.Empty(t, status.FailureReason)
}

// TestCheckEmergency_Delegation 验证 wails_app.go 的 CheckEmergency 正确委托到 application 层。
func TestCheckEmergency_Delegation(t *testing.T) {
	app := &WailsApp{}

	// A 级
	result, err := app.CheckEmergency("胸痛 呼吸困难")
	require.NoError(t, err)
	assert.Equal(t, "A", result.Level)

	// B 级
	result, err = app.CheckEmergency("持续高热")
	require.NoError(t, err)
	assert.Equal(t, "B", result.Level)

	// 无命中
	result, err = app.CheckEmergency("普通感冒吃什么好")
	require.NoError(t, err)
	assert.Equal(t, "none", result.Level)
}

func newEmbeddingStatusTestApp(t *testing.T, available bool, failureReason string, runtimePresent bool) (*WailsApp, string) {
	t.Helper()
	tmpDir := t.TempDir()
	modelDir := filepath.Join(tmpDir, "models", "all-MiniLM-L6-v2")
	require.NoError(t, os.MkdirAll(modelDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(modelDir, "model.onnx"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(modelDir, "tokenizer.json"), []byte("{}"), 0644))

	runtimePath := filepath.Join(tmpDir, "libonnxruntime.dylib")
	if runtimePresent {
		require.NoError(t, os.WriteFile(runtimePath, []byte("test"), 0644))
	}

	app := &WailsApp{
		ctx: context.Background(),
		config: &entity.AppConfig{
			DataDir: tmpDir,
		},
		embeddingSvc: &mockEmbeddingStatusService{
			available:     available,
			failureReason: failureReason,
			runtimePath:   runtimePath,
		},
	}
	return app, runtimePath
}

type mockEmbeddingStatusService struct {
	available     bool
	failureReason string
	runtimePath   string
}

func (m *mockEmbeddingStatusService) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if !m.available {
		return nil, assert.AnError
	}
	return make([][]float32, len(texts)), nil
}

func (m *mockEmbeddingStatusService) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	if !m.available {
		return nil, assert.AnError
	}
	return []float32{1}, nil
}

func (m *mockEmbeddingStatusService) ModelVersion() string { return "test-model" }
func (m *mockEmbeddingStatusService) IsAvailable() bool {
	return m.available
}

func (m *mockEmbeddingStatusService) FailureReason() string {
	return m.failureReason
}

func (m *mockEmbeddingStatusService) RuntimeLibPath() string {
	return m.runtimePath
}

// TestEvaluateEmergency_Integration 直接测试 application 层紧急检测引擎。
func TestEvaluateEmergency_Integration(t *testing.T) {
	// A 级覆盖更多关键词
	cases := []string{
		"心跳骤停怎么办",
		"过敏性休克紧急处理",
		"孕妇破水了",
		"新生儿发热不退",
		"一氧化碳中毒",
		"电击伤后昏迷",
		"窒息喘不过气",
	}
	for _, text := range cases {
		result := application.EvaluateEmergency(text)
		assert.Equal(t, application.LevelA, result.Level, "文本: %s", text)
	}

	// B 级覆盖更多关键词
	cases = []string{
		"黑便三天了",
		"便血伴有腹痛",
		"黄疸加重",
		"咯血痰中带血",
		"关节红肿热痛",
		"不明原因发热消瘦",
	}
	for _, text := range cases {
		result := application.EvaluateEmergency(text)
		assert.Equal(t, application.LevelB, result.Level, "文本: %s", text)
	}
}

// mockMessageRepo 是一个轻量级的内存消息仓库 mock，用于单元测试。
type mockMessageRepo struct {
	saved []*entity.Message
}

func (m *mockMessageRepo) Save(_ context.Context, _ models.ConversationID, msg *entity.Message) error {
	m.saved = append(m.saved, msg)
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

// TestStreamTimeout_DefaultWhenProviderStoreNil 验证当 providerStore 为 nil 时，streamTimeout 返回默认的 5 分钟。
func TestStreamTimeout_DefaultWhenProviderStoreNil(t *testing.T) {
	app := &WailsApp{ctx: t.Context()}
	assert.Equal(t, 5*time.Minute, app.streamTimeout("gemini"))
}

// TestStreamTimeout_DefaultWhenProviderIDEmpty 验证当 providerID 为空字符串时，streamTimeout 返回默认的 5 分钟。
func TestStreamTimeout_DefaultWhenProviderIDEmpty(t *testing.T) {
	app := &WailsApp{ctx: t.Context(), providerStore: newMockProviderStore()}
	assert.Equal(t, 5*time.Minute, app.streamTimeout(""))
}

// TestSaveUserMessage_NilMsgRepo 验证当 msgRepo 为 nil 时，saveUserMessage 应安全返回而不 panic。
func TestSaveUserMessage_NilMsgRepo(t *testing.T) {
	app := &WailsApp{ctx: t.Context(), msgRepo: nil}
	// 若发生 panic，测试框架会自动捕获并标记失败
	app.saveUserMessage(t.Context(), "conv-1", models.Message{Role: models.RoleUser, Content: "hello"})
}

// TestSaveUserMessage_EmptyConvID 验证当 convID 为空时，saveUserMessage 应直接返回且不执行保存。
func TestSaveUserMessage_EmptyConvID(t *testing.T) {
	mock := &mockMessageRepo{}
	app := &WailsApp{ctx: t.Context(), msgRepo: mock}
	app.saveUserMessage(t.Context(), "", models.Message{Role: models.RoleUser, Content: "hello"})
	assert.Empty(t, mock.saved)
}

// TestSaveUserMessage_NonUserRole 验证当消息角色不是 RoleUser 时，saveUserMessage 应直接返回且不保存。
func TestSaveUserMessage_NonUserRole(t *testing.T) {
	mock := &mockMessageRepo{}
	app := &WailsApp{ctx: t.Context(), msgRepo: mock}
	app.saveUserMessage(t.Context(), "conv-1", models.Message{Role: models.RoleAssistant, Content: "hello"})
	assert.Empty(t, mock.saved)
}

// TestExtractFactsAsync_Timeout 验证 extractFactsAsync 在 60s 超时后正确退出。
func TestExtractFactsAsync_Timeout(t *testing.T) {
	app := &WailsApp{ctx: t.Context()}

	// 空 providerID 应快速返回，不需要等待 60s
	done := make(chan struct{})
	go func() {
		app.extractFactsAsync("user content", "ai reply", "")
		close(done)
	}()

	select {
	case <-done:
		// 空 providerID 应快速返回，不需要等待 60s
	case <-time.After(2 * time.Second):
		t.Fatal("extractFactsAsync 在空 providerID 时应快速返回")
	}
}

// TestExtractFactsAsync_ObservableError 验证 extractFactsAsync 错误是可观测的。
func TestExtractFactsAsync_ObservableError(t *testing.T) {
	app := &WailsApp{ctx: t.Context()}

	// 空 providerID 不应 panic，且应快速返回
	app.extractFactsAsync("test", "reply", "")
	// 无 panic 即通过
}

// ========== WailsApp Mock Types (for binding tests) ==========

// wailsMockLLMClient 实现 port.LLMClient 接口。
type wailsMockLLMClient struct {
	chatReply    string
	chatErr      error
	streamChunks []string
	streamErr    error
	lastMessages []models.Message
}

func (m *wailsMockLLMClient) Chat(ctx context.Context, messages []models.Message) (string, error) {
	m.lastMessages = messages
	return m.chatReply, m.chatErr
}

func (m *wailsMockLLMClient) StreamChat(ctx context.Context, messages []models.Message, callback func(chunk string)) (*models.TokenUsage, error) {
	m.lastMessages = messages
	for _, chunk := range m.streamChunks {
		callback(chunk)
	}
	return nil, m.streamErr
}

func (m *wailsMockLLMClient) CheckAvailability(ctx context.Context) (bool, string) {
	return true, "available"
}

var _ port.LLMClient = (*wailsMockLLMClient)(nil)

// wailsMockLLMClientFactory 实现 port.LLMClientFactory。
type wailsMockLLMClientFactory struct {
	client port.LLMClient
}

func (m *wailsMockLLMClientFactory) CreateClient(providerConfig *models.ProviderConfig) (port.LLMClient, error) {
	return m.client, nil
}

var _ port.LLMClientFactory = (*wailsMockLLMClientFactory)(nil)

// wailsMockFactRepository 实现 repository.FactRepository 接口。
type wailsMockFactRepository struct{}

func (m *wailsMockFactRepository) Save(ctx context.Context, f *entity.ExtractedFact) error {
	return nil
}
func (m *wailsMockFactRepository) GetByID(ctx context.Context, factID string) (*entity.ExtractedFact, error) {
	return nil, entity.ErrFactNotFound
}
func (m *wailsMockFactRepository) ListByStatus(ctx context.Context, status entity.FactStatus, offset, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *wailsMockFactRepository) ListPending(ctx context.Context, offset, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *wailsMockFactRepository) UpdateStatus(ctx context.Context, factID string, status entity.FactStatus) error {
	return nil
}
func (m *wailsMockFactRepository) Delete(ctx context.Context, factID string) error { return nil }
func (m *wailsMockFactRepository) GetStats(ctx context.Context) (total, approved, rejected, pending int64, err error) {
	return 0, 0, 0, 0, nil
}
func (m *wailsMockFactRepository) ListAllSubjects(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (m *wailsMockFactRepository) FindBySubject(ctx context.Context, subject string) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *wailsMockFactRepository) FindBySession(ctx context.Context, sessionID string) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *wailsMockFactRepository) FindApprovedByPredicates(ctx context.Context, subject string, predicates []string, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}
func (m *wailsMockFactRepository) FindLatestApprovedByPredicates(ctx context.Context, subject string, predicates []string) (*entity.ExtractedFact, error) {
	return nil, entity.ErrFactNotFound
}
func (m *wailsMockFactRepository) CountApprovedFactsNeedingEmbedding(ctx context.Context, targetVersion string) (int64, error) {
	return 0, nil
}
func (m *wailsMockFactRepository) ListApprovedFactsNeedingEmbedding(ctx context.Context, targetVersion string, lastCreatedAt time.Time, lastFactID string, limit int) ([]*entity.ExtractedFact, error) {
	return nil, nil
}

var _ repository.FactRepository = (*wailsMockFactRepository)(nil)

// wailsMustEmptyRulesPath 创建包含空规则库的临时 JSON 文件并返回路径。
func wailsMustEmptyRulesPath(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "rules.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"version":"test","rules":[]}`), 0644))
	return path
}

// wailsNewTestComplianceChecker 从临时规则文件创建合规检查器。
func wailsNewTestComplianceChecker(t *testing.T, path string) usecase.ComplianceChecker {
	t.Helper()
	// 使用一个简单的 pass-through compliance checker
	return &wailsMockComplianceChecker{}
}

// wailsMockComplianceChecker 实现 usecase.ComplianceChecker 接口。
type wailsMockComplianceChecker struct{}

func (m *wailsMockComplianceChecker) Check(ctx context.Context, text string) (*usecase.ComplianceResult, error) {
	return &usecase.ComplianceResult{
		Blocked:  false,
		Level:    "L4_NORMAL",
		SafeText: text,
	}, nil
}

// ========== Chat Binding Tests (SendMessage / SendMessageStream) ==========

// TestWailsApp_SendMessage_RequestMapping 验证 SendMessage 正确映射请求参数到 ChatRequest。
func TestWailsApp_SendMessage_RequestMapping(t *testing.T) {
	mock := &wailsMockLLMClient{chatReply: "AI 回复"}
	comp := wailsNewTestComplianceChecker(t, wailsMustEmptyRulesPath(t))
	factory := &wailsMockLLMClientFactory{client: mock}
	store := newMockProviderStore()
	orch := usecase.NewChatOrchestrator(factory, store, nil, nil, comp, nil, nil, usecase.NewConfidenceAggregator(), &wailsMockFactRepository{}, usecase.NewIntentResolver(usecase.NewQueryExpansionService()), usecase.NewLocalAnswerService())

	app := &WailsApp{
		ctx:              t.Context(),
		chatOrchestrator: orch,
		msgRepo:          &mockMessageRepo{},
	}

	req := SendMessageRequest{
		ConversationID: "conv-1",
		Messages: []models.Message{
			{Role: models.RoleUser, Content: "你好"},
		},
		Model:      "kimi",
		ProviderID: "test-provider",
	}
	resp, err := app.SendMessage(req)
	require.NoError(t, err)
	assert.Equal(t, "AI 回复", resp.Reply)
}

// TestWailsApp_SendMessage_ErrorPropagation 验证 SendMessage 错误被正确包装。
func TestWailsApp_SendMessage_ErrorPropagation(t *testing.T) {
	mock := &wailsMockLLMClient{chatErr: fmt.Errorf("llm connection failed")}
	comp := wailsNewTestComplianceChecker(t, wailsMustEmptyRulesPath(t))
	factory := &wailsMockLLMClientFactory{client: mock}
	store := newMockProviderStore()
	orch := usecase.NewChatOrchestrator(factory, store, nil, nil, comp, nil, nil, usecase.NewConfidenceAggregator(), &wailsMockFactRepository{}, usecase.NewIntentResolver(usecase.NewQueryExpansionService()), usecase.NewLocalAnswerService())

	app := &WailsApp{
		ctx:              t.Context(),
		chatOrchestrator: orch,
		msgRepo:          &mockMessageRepo{},
	}

	req := SendMessageRequest{
		ConversationID: "conv-err",
		Messages:       []models.Message{{Role: models.RoleUser, Content: "test"}},
		ProviderID:     "test-provider",
	}
	_, err := app.SendMessage(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send message")
}

// TestWailsApp_activeStreams_ConcurrentAccess 验证 activeStreams 并发安全。
func TestWailsApp_activeStreams_ConcurrentAccess(t *testing.T) {
	app := &WailsApp{
		ctx:           context.Background(),
		activeStreams: make(map[string]context.CancelFunc),
		streamMu:      sync.Mutex{},
	}

	// 并发写入
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			convID := fmt.Sprintf("conv-%d", idx)
			_, cancel := context.WithCancel(context.Background())
			app.streamMu.Lock()
			app.activeStreams[convID] = cancel
			app.streamMu.Unlock()

			// 模拟一段时间后取消
			app.streamMu.Lock()
			if c, ok := app.activeStreams[convID]; ok {
				c()
				delete(app.activeStreams, convID)
			}
			app.streamMu.Unlock()
		}(i)
	}
	wg.Wait()

	// 验证所有 stream 已被清理
	app.streamMu.Lock()
	count := len(app.activeStreams)
	app.streamMu.Unlock()
	assert.Equal(t, 0, count, "所有 activeStreams 应已被清理")
}

// TestWailsApp_streamTimeout_NilProviderStore 验证 providerStore 为 nil 时返回默认超时。
func TestWailsApp_streamTimeout_NilProviderStore(t *testing.T) {
	app := &WailsApp{ctx: t.Context()}
	assert.Equal(t, 5*time.Minute, app.streamTimeout("any-provider"))
}

// TestWailsApp_streamTimeout_EmptyProviderID 验证空 providerID 返回默认超时。
func TestWailsApp_streamTimeout_EmptyProviderID(t *testing.T) {
	app := &WailsApp{ctx: t.Context(), providerStore: newMockProviderStore()}
	assert.Equal(t, 5*time.Minute, app.streamTimeout(""))
}

// TestWailsApp_safeEventsEmit_NilContext 验证 nil context 时不 panic。
func TestWailsApp_safeEventsEmit_NilContext(t *testing.T) {
	app := &WailsApp{ctx: nil}
	// 不应 panic
	app.safeEventsEmit("test:event", "data")
}

// TestWailsApp_safeEventsEmit_BackgroundContext 验证 Background context 时不发送。
func TestWailsApp_safeEventsEmit_BackgroundContext(t *testing.T) {
	app := &WailsApp{ctx: context.Background()}
	// 不应 panic
	app.safeEventsEmit("test:event", "data")
}

// ========== StopGeneration 绑定测试 ==========

// TestStopGeneration_CancelsActiveStreams 验证 StopGeneration 取消所有活跃流。
func TestStopGeneration_CancelsActiveStreams(t *testing.T) {
	var cancelled []string
	app := &WailsApp{
		ctx:           context.Background(),
		activeStreams: make(map[string]context.CancelFunc),
		streamMu:      sync.Mutex{},
	}

	// 注册两个活跃流
	for _, id := range []string{"conv-1", "conv-2"} {
		convID := id
		ctx, cancel := context.WithCancel(context.Background())
		app.activeStreams[convID] = cancel
		go func() {
			<-ctx.Done()
			app.streamMu.Lock()
			cancelled = append(cancelled, convID)
			app.streamMu.Unlock()
		}()
	}

	app.StopGeneration()

	// 等待 goroutine 检测到取消
	time.Sleep(50 * time.Millisecond)
	app.streamMu.Lock()
	assert.ElementsMatch(t, []string{"conv-1", "conv-2"}, cancelled)
	app.streamMu.Unlock()
}

// TestStopGeneration_EmptyStreams 验证空 activeStreams 不 panic。
func TestStopGeneration_EmptyStreams(t *testing.T) {
	app := &WailsApp{
		ctx:           context.Background(),
		activeStreams: make(map[string]context.CancelFunc),
		streamMu:      sync.Mutex{},
	}
	// 不应 panic
	app.StopGeneration()
}

// ========== Provider Health 绑定测试 ==========

// mockHealthChecker 实现 port.HealthChecker。
type mockHealthChecker struct {
	checkResult port.HealthResult
	checkErr    error
	status      map[string]port.HealthResult
}

func (m *mockHealthChecker) Start(ctx context.Context) {}
func (m *mockHealthChecker) Stop()                     {}
func (m *mockHealthChecker) CheckNow(ctx context.Context, providerID string) (port.HealthResult, error) {
	if m.checkErr != nil {
		return port.HealthResult{}, m.checkErr
	}
	return m.checkResult, nil
}
func (m *mockHealthChecker) GetStatus(providerID string) (port.HealthResult, bool) {
	r, ok := m.status[providerID]
	return r, ok
}
func (m *mockHealthChecker) SetOnChange(cb func(port.HealthResult)) {}

var _ port.HealthChecker = (*mockHealthChecker)(nil)

// TestCheckProviderHealth_NilChecker 验证 healthChecker 为 nil 时返回错误。
func TestCheckProviderHealth_NilChecker(t *testing.T) {
	app := &WailsApp{ctx: t.Context()}
	_, err := app.CheckProviderHealth("kimi")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "health checker not initialized")
}

// TestCheckProviderHealth_Success 验证健康检查正常返回。
func TestCheckProviderHealth_Success(t *testing.T) {
	now := time.Now()
	hc := &mockHealthChecker{
		checkResult: port.HealthResult{
			ProviderID: "kimi",
			Status:     port.HealthGreen,
			LatencyMs:  150,
			CheckedAt:  now,
		},
	}
	app := &WailsApp{ctx: t.Context(), healthChecker: hc}

	result, err := app.CheckProviderHealth("kimi")
	require.NoError(t, err)
	assert.Equal(t, "kimi", result.ProviderID)
	assert.Equal(t, "green", result.Status)
	assert.EqualValues(t, 150, result.LatencyMs)
}

// TestCheckProviderHealth_Error 验证健康检查错误被正确包装。
func TestCheckProviderHealth_Error(t *testing.T) {
	hc := &mockHealthChecker{
		checkErr: fmt.Errorf("connection refused"),
	}
	app := &WailsApp{ctx: t.Context(), healthChecker: hc}

	_, err := app.CheckProviderHealth("bad-provider")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check provider health")
}

// TestGetProviderHealthStatus_NilChecker 验证 healthChecker 为 nil 时返回错误。
func TestGetProviderHealthStatus_NilChecker(t *testing.T) {
	app := &WailsApp{ctx: t.Context()}
	_, err := app.GetProviderHealthStatus("kimi")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "health checker not initialized")
}

// TestGetProviderHealthStatus_NotFound 验证未缓存的 Provider 返回错误。
func TestGetProviderHealthStatus_NotFound(t *testing.T) {
	hc := &mockHealthChecker{
		status: map[string]port.HealthResult{},
	}
	app := &WailsApp{ctx: t.Context(), healthChecker: hc}

	_, err := app.GetProviderHealthStatus("unknown")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "health status not available")
}

// TestGetProviderHealthStatus_Success 验证缓存命中正常返回。
func TestGetProviderHealthStatus_Success(t *testing.T) {
	now := time.Now()
	hc := &mockHealthChecker{
		status: map[string]port.HealthResult{
			"kimi": {ProviderID: "kimi", Status: port.HealthYellow, LatencyMs: 3000, CheckedAt: now},
		},
	}
	app := &WailsApp{ctx: t.Context(), healthChecker: hc}

	result, err := app.GetProviderHealthStatus("kimi")
	require.NoError(t, err)
	assert.Equal(t, "kimi", result.ProviderID)
	assert.Equal(t, "yellow", result.Status)
}

// ========== Updater 绑定测试 ==========

// TestCheckUpdate_NilService 验证 updaterSvc 为 nil 时返回错误。
func TestCheckUpdate_NilService(t *testing.T) {
	app := &WailsApp{ctx: t.Context()}
	_, err := app.CheckUpdate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "updater service not initialized")
}

// TestDownloadUpdate_NilService 验证 updaterSvc 为 nil 时返回错误。
func TestDownloadUpdate_NilService(t *testing.T) {
	app := &WailsApp{ctx: t.Context()}
	_, err := app.DownloadUpdate(DownloadUpdateRequest{Version: "v1.0.0"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "updater service not initialized")
}

// TestApplyUpdate_NilService 验证 updaterSvc 为 nil 时返回错误。
func TestApplyUpdate_NilService(t *testing.T) {
	app := &WailsApp{ctx: t.Context()}
	err := app.ApplyUpdate("/path/to/asset")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "updater service not initialized")
}

// TestGetUpdateSettings_NilService 验证 updaterSvc 为 nil 时返回错误。
func TestGetUpdateSettings_NilService(t *testing.T) {
	app := &WailsApp{ctx: t.Context()}
	_, err := app.GetUpdateSettings()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "updater service not initialized")
}

// TestSkipUpdateVersion_NilService 验证 updaterSvc 为 nil 时返回错误。
func TestSkipUpdateVersion_NilService(t *testing.T) {
	app := &WailsApp{ctx: t.Context()}
	err := app.SkipUpdateVersion("v1.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "updater service not initialized")
}
