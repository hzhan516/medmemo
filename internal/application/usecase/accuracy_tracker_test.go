package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/hzhan516/medmemo/internal/domain/entity"
)

func TestAccuracyTracker_GetAccuracy_ColdStart(t *testing.T) {
	tracker := NewAccuracyTracker()
	acc := tracker.GetAccuracy(entity.AnswerTypeSymptomAnalysis, "30d")
	// 冷启动默认基准值 0.75
	assert.InDelta(t, 0.75, acc, 0.001)
}

func TestAccuracyTracker_RecordFeedback_ThumbsUp(t *testing.T) {
	tracker := NewAccuracyTracker()
	tracker.RecordFeedback(entity.AnswerTypeSymptomAnalysis, true, "30d")

	acc := tracker.GetAccuracy(entity.AnswerTypeSymptomAnalysis, "30d")
	assert.InDelta(t, 1.0, acc, 0.001)
}

func TestAccuracyTracker_RecordFeedback_ThumbsDown(t *testing.T) {
	tracker := NewAccuracyTracker()
	tracker.RecordFeedback(entity.AnswerTypeSymptomAnalysis, false, "30d")

	acc := tracker.GetAccuracy(entity.AnswerTypeSymptomAnalysis, "30d")
	assert.InDelta(t, 0.0, acc, 0.001)
}

func TestAccuracyTracker_GetAccuracy_MixedFeedback(t *testing.T) {
	tracker := NewAccuracyTracker()
	// 7 正确 / 10 总计 = 0.70
	for i := 0; i < 7; i++ {
		tracker.RecordFeedback(entity.AnswerTypeRecommendation, true, "30d")
	}
	for i := 0; i < 3; i++ {
		tracker.RecordFeedback(entity.AnswerTypeRecommendation, false, "30d")
	}

	acc := tracker.GetAccuracy(entity.AnswerTypeRecommendation, "30d")
	assert.InDelta(t, 0.70, acc, 0.001)
}

func TestAccuracyTracker_GetAccuracy_DifferentWindows(t *testing.T) {
	tracker := NewAccuracyTracker()
	// 30d 窗口：1 正确 / 1 = 1.0
	tracker.RecordFeedback(entity.AnswerTypeHealthInfo, true, "30d")

	// 90d 窗口：包含 30d 的数据
	acc30d := tracker.GetAccuracy(entity.AnswerTypeHealthInfo, "30d")
	acc90d := tracker.GetAccuracy(entity.AnswerTypeHealthInfo, "90d")
	accAll := tracker.GetAccuracy(entity.AnswerTypeHealthInfo, "all")

	assert.InDelta(t, 1.0, acc30d, 0.001)
	assert.InDelta(t, 1.0, acc90d, 0.001)
	assert.InDelta(t, 1.0, accAll, 0.001)
}

func TestAccuracyTracker_GetAccuracy_DifferentAnswerTypes(t *testing.T) {
	tracker := NewAccuracyTracker()
	tracker.RecordFeedback(entity.AnswerTypeSymptomAnalysis, true, "30d")
	tracker.RecordFeedback(entity.AnswerTypeRecommendation, false, "30d")
	tracker.RecordFeedback(entity.AnswerTypeHealthInfo, true, "30d")
	tracker.RecordFeedback(entity.AnswerTypeEmergency, true, "30d")

	assert.InDelta(t, 1.0, tracker.GetAccuracy(entity.AnswerTypeSymptomAnalysis, "30d"), 0.001)
	assert.InDelta(t, 0.0, tracker.GetAccuracy(entity.AnswerTypeRecommendation, "30d"), 0.001)
	assert.InDelta(t, 1.0, tracker.GetAccuracy(entity.AnswerTypeHealthInfo, "30d"), 0.001)
	assert.InDelta(t, 1.0, tracker.GetAccuracy(entity.AnswerTypeEmergency, "30d"), 0.001)
}

func TestAccuracyTracker_GetStats(t *testing.T) {
	tracker := NewAccuracyTracker()
	tracker.RecordFeedback(entity.AnswerTypeSymptomAnalysis, true, "30d")
	tracker.RecordFeedback(entity.AnswerTypeSymptomAnalysis, true, "30d")
	tracker.RecordFeedback(entity.AnswerTypeSymptomAnalysis, false, "30d")

	stats := tracker.GetStats(entity.AnswerTypeSymptomAnalysis, "30d")
	assert.Equal(t, 3, stats.TotalCount)
	assert.Equal(t, 2, stats.CorrectCount)
	assert.InDelta(t, 0.667, stats.Accuracy(), 0.01)
}

func TestAccuracyTracker_RecordImplicitSignal_Positive(t *testing.T) {
	tracker := NewAccuracyTracker()
	// 用户追问 = 积极信号
	tracker.RecordImplicitSignal(entity.AnswerTypeSymptomAnalysis, true, "30d")

	acc := tracker.GetAccuracy(entity.AnswerTypeSymptomAnalysis, "30d")
	assert.InDelta(t, 1.0, acc, 0.001)
}

func TestAccuracyTracker_RecordImplicitSignal_Negative(t *testing.T) {
	tracker := NewAccuracyTracker()
	// 用户未追问 = 消极信号
	tracker.RecordImplicitSignal(entity.AnswerTypeSymptomAnalysis, false, "30d")

	acc := tracker.GetAccuracy(entity.AnswerTypeSymptomAnalysis, "30d")
	assert.InDelta(t, 0.0, acc, 0.001)
}

func TestAccuracyTracker_ConcurrentSafety(t *testing.T) {
	tracker := NewAccuracyTracker()
	done := make(chan bool)

	// 多个 goroutine 并发写入
	for i := 0; i < 100; i++ {
		go func() {
			tracker.RecordFeedback(entity.AnswerTypeSymptomAnalysis, true, "30d")
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	acc := tracker.GetAccuracy(entity.AnswerTypeSymptomAnalysis, "30d")
	assert.InDelta(t, 1.0, acc, 0.001)
}

func TestAccuracyTracker_GetAccuracy_InvalidWindow(t *testing.T) {
	tracker := NewAccuracyTracker()
	// 无效窗口默认使用 30d
	acc := tracker.GetAccuracy(entity.AnswerTypeSymptomAnalysis, "invalid")
	assert.InDelta(t, 0.75, acc, 0.001)
}

func TestAccuracyTracker_WindowExpiration(t *testing.T) {
	tracker := NewAccuracyTracker()
	// 模拟过期数据：手动插入一个 40 天前的记录
	oldTime := time.Now().UTC().Add(-40 * 24 * time.Hour)
	tracker.recordWithTime(entity.AnswerTypeSymptomAnalysis, true, "30d", oldTime)

	// 30d 窗口不应包含过期数据
	acc30d := tracker.GetAccuracy(entity.AnswerTypeSymptomAnalysis, "30d")
	// 冷启动默认值
	assert.InDelta(t, 0.75, acc30d, 0.001)
}
