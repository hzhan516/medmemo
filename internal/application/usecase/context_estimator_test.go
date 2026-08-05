package usecase

import (
	"context"
	"testing"

	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/pkg/models"
)

// fakeTokenCounter 是一个简单的 TokenCounter 实现，用于测试。
// 它返回文本长度作为 token 数量，并返回 ok 标志。
type fakeTokenCounter struct {
	ok bool
}

func (f *fakeTokenCounter) Count(_ context.Context, _ string, text string) (int, bool) {
	return len(text), f.ok
}

// fakeProviderStore 是一个简单的 ProviderStore 实现，用于测试。
type fakeProviderStore struct {
	provider *models.ProviderConfig
}

func (f *fakeProviderStore) Create(_ context.Context, _ *models.ProviderConfig) error {
	return nil
}

func (f *fakeProviderStore) Update(_ context.Context, _ *models.ProviderConfig) error {
	return nil
}

func (f *fakeProviderStore) Delete(_ context.Context, _ string) error {
	return nil
}

func (f *fakeProviderStore) Get(_ context.Context, _ string) (*models.ProviderConfig, error) {
	return f.provider, nil
}

func (f *fakeProviderStore) List(_ context.Context) ([]*models.ProviderConfig, error) {
	if f.provider == nil {
		return nil, nil
	}
	return []*models.ProviderConfig{f.provider}, nil
}

// newTestContextEstimator 创建一个用于测试的 ContextEstimator。
func newTestContextEstimator(ok bool) (*ContextEstimator, *fakeTokenCounter) {
	counter := &fakeTokenCounter{ok: ok}
	store := &fakeProviderStore{
		provider: &models.ProviderConfig{
			ID:   "test-provider",
			Name: "Test Provider",
			Models: []models.ProviderModel{
				{
					ID:               "test-model",
					Name:             "Test Model",
					MaxContextLength: 1024,
				},
			},
		},
	}
	resolver := NewContextLengthResolver(store)
	return NewContextEstimator(counter, resolver), counter
}

func TestBoundedRatio(t *testing.T) {
	tests := []struct {
		name string
		used int
		max  int
		want float64
	}{
		{"max zero", 10, 0, 0},
		{"max negative", 10, -5, 0},
		{"used zero", 0, 100, 0},
		{"used and max equal", 100, 100, 1},
		{"used exceeds max", 200, 100, 1},
		{"used negative", -10, 100, 0},
		{"half", 50, 100, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := boundedRatio(tt.used, tt.max)
			if got != tt.want {
				t.Errorf("boundedRatio(%d, %d) = %v, want %v", tt.used, tt.max, got, tt.want)
			}
		})
	}
}

func TestEstimateReturnsUsedTokensAndApproximate(t *testing.T) {
	ctx := context.Background()
	estimator, _ := newTestContextEstimator(true)

	in := EstimatorInput{
		ProviderID: "test-provider",
		ModelID:    "test-model",
		Messages: []models.Message{
			{Role: models.RoleUser, Content: "hello"},
			{Role: models.RoleAssistant, Content: "world"},
		},
	}

	got, err := estimator.Estimate(ctx, in)
	if err != nil {
		t.Fatalf("Estimate() returned error: %v", err)
	}

	// Each message: len(content) + perMessageOverhead
	// hello = 5, world = 5, overhead = 4 each
	wantUsed := (5 + 4) + (5 + 4)
	if got.UsedTokens != wantUsed {
		t.Errorf("UsedTokens = %d, want %d", got.UsedTokens, wantUsed)
	}
	if got.MaxTokens != 1024 {
		t.Errorf("MaxTokens = %d, want 1024", got.MaxTokens)
	}
	if got.Approximate != false {
		t.Errorf("Approximate = %v, want false", got.Approximate)
	}
	if got.Ratio != 0.017578125 {
		t.Errorf("Ratio = %v, want 0.017578125", got.Ratio)
	}
}

func TestEstimateApproximateWhenCounterNotOk(t *testing.T) {
	ctx := context.Background()
	estimator, _ := newTestContextEstimator(false)

	in := EstimatorInput{
		ProviderID: "test-provider",
		ModelID:    "test-model",
		Messages: []models.Message{
			{Role: models.RoleUser, Content: "hello"},
		},
	}

	got, err := estimator.Estimate(ctx, in)
	if err != nil {
		t.Fatalf("Estimate() returned error: %v", err)
	}

	if got.Approximate != true {
		t.Errorf("Approximate = %v, want true", got.Approximate)
	}
}

func TestEstimatePrefersAssembledPrompt(t *testing.T) {
	ctx := context.Background()
	estimator, _ := newTestContextEstimator(true)

	in := EstimatorInput{
		ProviderID: "test-provider",
		ModelID:    "test-model",
		Messages: []models.Message{
			{Role: models.RoleUser, Content: "fallback-message"},
		},
		AssembledPrompt: []models.Message{
			{Role: models.RoleSystem, Content: "assembled"},
		},
	}

	got, err := estimator.Estimate(ctx, in)
	if err != nil {
		t.Fatalf("Estimate() returned error: %v", err)
	}

	// AssembledPrompt contains one message with content "assembled" (len 9)
	wantUsed := 9 + perMessageOverhead
	if got.UsedTokens != wantUsed {
		t.Errorf("UsedTokens = %d, want %d (should use AssembledPrompt)", got.UsedTokens, wantUsed)
	}
}

func TestEstimateFallsBackToMessagesWhenAssembledPromptNil(t *testing.T) {
	ctx := context.Background()
	estimator, _ := newTestContextEstimator(true)

	in := EstimatorInput{
		ProviderID:      "test-provider",
		ModelID:         "test-model",
		AssembledPrompt: nil,
		Messages: []models.Message{
			{Role: models.RoleUser, Content: "fallback-message"},
		},
	}

	got, err := estimator.Estimate(ctx, in)
	if err != nil {
		t.Fatalf("Estimate() returned error: %v", err)
	}

	// Messages contains one message with content "fallback-message" (len 16)
	wantUsed := 16 + perMessageOverhead
	if got.UsedTokens != wantUsed {
		t.Errorf("UsedTokens = %d, want %d (should fall back to Messages)", got.UsedTokens, wantUsed)
	}
}

// 编译时检查 fakeTokenCounter 实现了 port.TokenCounter 接口。
var _ port.TokenCounter = (*fakeTokenCounter)(nil)

// 编译时检查 fakeProviderStore 实现了 port.ProviderStore 接口。
var _ port.ProviderStore = (*fakeProviderStore)(nil)
