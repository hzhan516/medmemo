package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAccuracyRepository 用于测试 AccuracyService 的内存仓库。
type mockAccuracyRepository struct {
	accuracies map[string]float64
	feedbacks  []accuracyFeedback
}

type accuracyFeedback struct {
	messageID  string
	answerType string
	feedback   string
}

func (m *mockAccuracyRepository) GetAccuracy(_ context.Context, answerType string) (float64, error) {
	if v, ok := m.accuracies[answerType]; ok {
		return v, nil
	}
	return 0.75, nil
}

func (m *mockAccuracyRepository) RecordFeedback(_ context.Context, messageID, answerType, feedback string) error {
	m.feedbacks = append(m.feedbacks, accuracyFeedback{messageID: messageID, answerType: answerType, feedback: feedback})
	return nil
}

func TestAccuracyService_GetAccuracy(t *testing.T) {
	ctx := context.Background()
	svc := NewAccuracyService(&mockAccuracyRepository{accuracies: map[string]float64{
		"health_info": 0.8,
	}})

	acc, err := svc.GetAccuracy(ctx, "health_info")
	require.NoError(t, err)
	assert.InDelta(t, 0.8, acc, 0.0001)

	acc, err = svc.GetAccuracy(ctx, "unknown_type")
	require.NoError(t, err)
	assert.InDelta(t, 0.75, acc, 0.0001)
}

func TestAccuracyService_RecordFeedback(t *testing.T) {
	ctx := context.Background()
	repo := &mockAccuracyRepository{accuracies: make(map[string]float64)}
	svc := NewAccuracyService(repo)

	err := svc.RecordFeedback(ctx, "msg-1", "symptom_analysis", true)
	require.NoError(t, err)
	require.Len(t, repo.feedbacks, 1)
	assert.Equal(t, "helpful", repo.feedbacks[0].feedback)

	err = svc.RecordFeedback(ctx, "msg-2", "symptom_analysis", false)
	require.NoError(t, err)
	assert.Equal(t, "inaccurate", repo.feedbacks[1].feedback)
}

func TestAccuracyService_RecordFeedback_Validation(t *testing.T) {
	ctx := context.Background()
	svc := NewAccuracyService(&mockAccuracyRepository{})

	err := svc.RecordFeedback(ctx, "", "symptom_analysis", true)
	assert.Error(t, err)

	err = svc.RecordFeedback(ctx, "msg-1", "", true)
	assert.Error(t, err)
}
