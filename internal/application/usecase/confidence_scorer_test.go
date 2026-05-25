package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/hzhan516/medmemo/internal/domain/entity"
)

func TestConfidenceScorer_PerfectFact(t *testing.T) {
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "患有", "偏头痛", 0.9, []string{"msg_001", "msg_002"})

	score := scorer.Score(f)
	// 0.2 + 0.2 + 0.2 + 0.9*0.3 + 0.1 = 0.97
	assert.InDelta(t, 0.97, score, 0.001)
}

func TestConfidenceScorer_MissingSubject(t *testing.T) {
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("", "患有", "偏头痛", 0.9, []string{"msg_001"})

	score := scorer.Score(f)
	// 0 + 0.2 + 0.2 + 0.9*0.3 + 0 = 0.67
	assert.InDelta(t, 0.67, score, 0.001)
}

func TestConfidenceScorer_MissingPredicate(t *testing.T) {
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "", "偏头痛", 0.9, []string{"msg_001"})

	score := scorer.Score(f)
	// 0.2 + 0 + 0.2 + 0.9*0.3 + 0 = 0.67
	assert.InDelta(t, 0.67, score, 0.001)
}

func TestConfidenceScorer_MissingObject(t *testing.T) {
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "患有", "", 0.9, []string{"msg_001"})

	score := scorer.Score(f)
	// 0.2 + 0.2 + 0 + 0.9*0.3 + 0 = 0.67
	assert.InDelta(t, 0.67, score, 0.001)
}

func TestConfidenceScorer_LowConfidence(t *testing.T) {
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "患有", "头痛", 0.3, []string{"msg_001"})

	score := scorer.Score(f)
	// 0.2 + 0.2 + 0.2 + 0.3*0.3 + 0 = 0.69
	assert.InDelta(t, 0.69, score, 0.001)
}

func TestConfidenceScorer_AutoApproveThreshold(t *testing.T) {
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "患有", "头痛", 0.9, []string{"msg_001"})
	// score = 0.2+0.2+0.2+0.9*0.3+0 = 0.87 >= 0.8

	status := scorer.EvaluateStatus(f)
	assert.Equal(t, entity.FactStatusApproved, status)
}

func TestConfidenceScorer_AutoRejectThreshold(t *testing.T) {
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "", "", 0.1, []string{"msg_001"})
	// score = 0.2+0+0+0.1*0.3+0 = 0.23 < 0.5

	status := scorer.EvaluateStatus(f)
	assert.Equal(t, entity.FactStatusRejected, status)
}

func TestConfidenceScorer_PendingThreshold(t *testing.T) {
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "患有", "头痛", 0.6, []string{"msg_001"})
	// score = 0.2+0.2+0.2+0.6*0.3+0 = 0.78

	status := scorer.EvaluateStatus(f)
	assert.Equal(t, entity.FactStatusPending, status)
}

func TestConfidenceScorer_ZeroConfidence(t *testing.T) {
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "患有", "头痛", 0.0, []string{"msg_001"})

	score := scorer.Score(f)
	// 0.2 + 0.2 + 0.2 + 0*0.3 + 0 = 0.6
	assert.InDelta(t, 0.6, score, 0.001)
}

func TestConfidenceScorer_SingleSource(t *testing.T) {
	scorer := NewConfidenceScorer()
	f := entity.NewExtractedFact("用户", "患有", "头痛", 0.8, []string{"msg_001"})

	score := scorer.Score(f)
	// 0.2+0.2+0.2+0.8*0.3+0 = 0.84
	assert.InDelta(t, 0.84, score, 0.001)
}
