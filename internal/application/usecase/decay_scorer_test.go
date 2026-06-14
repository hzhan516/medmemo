package usecase

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecayScorer_ZeroDays(t *testing.T) {
		t.Parallel()
	s := NewDecayScorer()
	score := s.Score(1.0, 0)
	assert.InDelta(t, 1.0, score, 0.0001)
}

func TestDecayScorer_14Days(t *testing.T) {
		t.Parallel()
	s := NewDecayScorer()
	// 半衰期 = ln(2) / lambda = 0.693 / 0.05 ≈ 13.86 天
	score := s.Score(1.0, 13.86)
	assert.InDelta(t, 0.5, score, 0.01)
}

func TestDecayScorer_30Days(t *testing.T) {
		t.Parallel()
	s := NewDecayScorer()
	score := s.Score(1.0, 30)
	// exp(-0.05 * 30) = exp(-1.5) ≈ 0.223
	assert.InDelta(t, 0.223, score, 0.01)
}

func TestDecayScorer_90Days(t *testing.T) {
		t.Parallel()
	s := NewDecayScorer()
	score := s.Score(1.0, 90)
	// exp(-0.05 * 90) = exp(-4.5) ≈ 0.011
	assert.InDelta(t, 0.011, score, 0.005)
}

func TestDecayScorer_NegativeAge(t *testing.T) {
		t.Parallel()
	s := NewDecayScorer()
	// 负年龄按 0 处理，不衰减
	score := s.Score(0.8, -5)
	assert.InDelta(t, 0.8, score, 0.0001)
}

func TestDecayScorer_SimilarityClamping(t *testing.T) {
		t.Parallel()
	s := NewDecayScorer()
	// similarity > 1 截断到 1
	score := s.Score(1.5, 0)
	assert.InDelta(t, 1.0, score, 0.0001)

	// similarity < 0 截断到 0
	score = s.Score(-0.5, 0)
	assert.InDelta(t, 0.0, score, 0.0001)
}

func TestDecayScorer_CustomLambda(t *testing.T) {
		t.Parallel()
	// lambda = 0.1，半衰期 ≈ 6.93 天
	s := NewDecayScorerWithLambda(0.1)
	score := s.Score(1.0, 6.93)
	assert.InDelta(t, 0.5, score, 0.01)
}

func TestDecayScorer_Rank(t *testing.T) {
		t.Parallel()
	s := NewDecayScorer()

	items := []struct {
		id         string
		similarity float64
		ageDays    float64
	}{
		{"recent_high", 0.9, 1},  // 0.9 * exp(-0.05) ≈ 0.856
		{"old_perfect", 1.0, 30}, // 1.0 * exp(-1.5) ≈ 0.223
		{"recent_low", 0.5, 0},   // 0.5 * exp(0) = 0.5
		{"mid_high", 0.85, 7},    // 0.85 * exp(-0.35) ≈ 0.600
	}

	ranked := s.Rank(items[0].similarity, items[0].ageDays,
		items[1].similarity, items[1].ageDays,
		items[2].similarity, items[2].ageDays,
		items[3].similarity, items[3].ageDays)

	// 期望顺序：recent_high > mid_high > recent_low > old_perfect
	require.Len(t, ranked, 4)
	assert.InDelta(t, 0.856, ranked[0], 0.01)
	assert.InDelta(t, 0.600, ranked[1], 0.01)
	assert.InDelta(t, 0.500, ranked[2], 0.01)
	assert.InDelta(t, 0.223, ranked[3], 0.01)
}

func TestDecayScorer_Rank_Empty(t *testing.T) {
		t.Parallel()
	s := NewDecayScorer()
	ranked := s.Rank()
	assert.Empty(t, ranked)
}

func TestDecayScorer_ScoreFromCreatedAt(t *testing.T) {
		t.Parallel()
	s := NewDecayScorer()
	now := time.Now().UTC()

	// 1 天前的记忆，similarity = 0.9
	createdAt := now.Add(-24 * time.Hour)
	score := s.ScoreFromCreatedAt(0.9, createdAt, now)
	expected := 0.9 * math.Exp(-0.05*1.0)
	assert.InDelta(t, expected, score, 0.0001)

	// 未来时间（负 age）按 0 处理
	future := now.Add(24 * time.Hour)
	score = s.ScoreFromCreatedAt(0.8, future, now)
	assert.InDelta(t, 0.8, score, 0.0001)
}
