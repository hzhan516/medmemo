package usecase

import (
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestConfidenceScorer_PerfectFact(t *testing.T) {
	t.Parallel()
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "患有", "偏头痛", 0.9, []string{"msg_001", "msg_002"})

	score := scorer.Score(f)
	// 0.2 + 0.2 + 0.2 + 0.9*0.3 + 0.1 = 0.97
	assert.InDelta(t, 0.97, score, 0.001)
}

func TestConfidenceScorer_MissingSubject(t *testing.T) {
	t.Parallel()
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("", "患有", "偏头痛", 0.9, []string{"msg_001"})

	score := scorer.Score(f)
	// 0 + 0.2 + 0.2 + 0.9*0.3 + 0 = 0.67
	assert.InDelta(t, 0.67, score, 0.001)
}

func TestConfidenceScorer_MissingPredicate(t *testing.T) {
	t.Parallel()
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "", "偏头痛", 0.9, []string{"msg_001"})

	score := scorer.Score(f)
	// 0.2 + 0 + 0.2 + 0.9*0.3 + 0 = 0.67
	assert.InDelta(t, 0.67, score, 0.001)
}

func TestConfidenceScorer_MissingObject(t *testing.T) {
	t.Parallel()
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "患有", "", 0.9, []string{"msg_001"})

	score := scorer.Score(f)
	// 0.2 + 0.2 + 0 + 0.9*0.3 + 0 = 0.67
	assert.InDelta(t, 0.67, score, 0.001)
}

func TestConfidenceScorer_LowConfidence(t *testing.T) {
	t.Parallel()
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "患有", "头痛", 0.3, []string{"msg_001"})

	score := scorer.Score(f)
	// 0.2 + 0.2 + 0.2 + 0.3*0.3 + 0 = 0.69
	assert.InDelta(t, 0.69, score, 0.001)
}

func TestConfidenceScorer_AutoApproveThreshold(t *testing.T) {
	t.Parallel()
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "患有", "头痛", 0.9, []string{"msg_001"})
	// score = 0.2+0.2+0.2+0.9*0.3+0 = 0.87 >= 0.8

	status := scorer.EvaluateStatus(f)
	assert.Equal(t, entity.FactStatusApproved, status)
}

func TestConfidenceScorer_AutoRejectThreshold(t *testing.T) {
	t.Parallel()
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "", "", 0.1, []string{"msg_001"})
	// score = 0.2+0+0+0.1*0.3+0 = 0.23 < 0.5

	status := scorer.EvaluateStatus(f)
	assert.Equal(t, entity.FactStatusRejected, status)
}

func TestConfidenceScorer_PendingThreshold(t *testing.T) {
	t.Parallel()
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "患有", "头痛", 0.6, []string{"msg_001"})
	// score = 0.2+0.2+0.2+0.6*0.3+0 = 0.78

	status := scorer.EvaluateStatus(f)
	assert.Equal(t, entity.FactStatusPending, status)
}

func TestConfidenceScorer_ZeroConfidence(t *testing.T) {
	t.Parallel()
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "患有", "头痛", 0.0, []string{"msg_001"})

	score := scorer.Score(f)
	// 0.2 + 0.2 + 0.2 + 0*0.3 + 0 = 0.6
	assert.InDelta(t, 0.6, score, 0.001)
}

func TestConfidenceScorer_SingleSource(t *testing.T) {
	t.Parallel()
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "患有", "头痛", 0.8, []string{"msg_001"})

	score := scorer.Score(f)
	// 0.2+0.2+0.2+0.8*0.3+0 = 0.84
	assert.InDelta(t, 0.84, score, 0.001)
}

func TestConfidenceScorer_SensitiveFact_Medical_NotSensitive(t *testing.T) {
	t.Parallel()
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "患有", "高血压", 0.9, []string{"msg_001"})

	scorer.Score(f)
	assert.False(t, f.IsSensitive, "医学敏感词不再驱动 IsSensitive")
}

func TestConfidenceScorer_SensitiveFact_PII(t *testing.T) {
	t.Parallel()
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "身份证", "110101199001011234", 0.9, []string{"msg_001"})

	scorer.Score(f)
	assert.True(t, f.IsSensitive, "PII 应被标记为敏感")
}

func TestConfidenceScorer_NonSensitiveFact(t *testing.T) {
	t.Parallel()
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "喜欢", "跑步", 0.9, []string{"msg_001"})

	scorer.Score(f)
	assert.False(t, f.IsSensitive, "非敏感事实不应被标记")
}
