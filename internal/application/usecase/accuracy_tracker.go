package usecase

import (
	"sync"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
)

// AccuracyTracker 历史准确率统计服务，滚动窗口追踪各类型回答的准确率（TASK-068）。
// 所有窗口共享同一组底层记录，查询时按时间范围过滤。
type AccuracyTracker struct {
	mu      sync.RWMutex
	records map[string][]accuracyRecord // key: answerType（不再区分窗口）
}

// accuracyRecord 单条反馈记录。
type accuracyRecord struct {
	Correct   bool
	Timestamp time.Time
}

// windowDurationMap 窗口名称到持续时间的映射。
var windowDurationMap = map[string]time.Duration{
	"30d": 30 * 24 * time.Hour,
	"90d": 90 * 24 * time.Hour,
	"all": 100 * 365 * 24 * time.Hour, // 约100年，等效于全部
}

// NewAccuracyTracker 创建新的历史准确率统计服务。
func NewAccuracyTracker() *AccuracyTracker {
	return &AccuracyTracker{
		records: make(map[string][]accuracyRecord),
	}
}

// RecordFeedback 记录显式用户反馈（thumbs up/down）。
// window 参数保留以兼容接口，实际存储不区分窗口。
func (at *AccuracyTracker) RecordFeedback(answerType entity.AnswerType, isCorrect bool, _ string) {
	at.mu.Lock()
	defer at.mu.Unlock()

	key := at.key(answerType)
	at.records[key] = append(at.records[key], accuracyRecord{
		Correct:   isCorrect,
		Timestamp: time.Now().UTC(),
	})
}

// RecordImplicitSignal 记录隐式用户行为信号。
func (at *AccuracyTracker) RecordImplicitSignal(answerType entity.AnswerType, isPositive bool, _ string) {
	at.mu.Lock()
	defer at.mu.Unlock()

	key := at.key(answerType)
	at.records[key] = append(at.records[key], accuracyRecord{
		Correct:   isPositive,
		Timestamp: time.Now().UTC(),
	})
}

// recordWithTime 带指定时间的内部记录方法（用于测试窗口过期）。
func (at *AccuracyTracker) recordWithTime(answerType entity.AnswerType, isCorrect bool, _ string, ts time.Time) {
	at.mu.Lock()
	defer at.mu.Unlock()

	key := at.key(answerType)
	at.records[key] = append(at.records[key], accuracyRecord{
		Correct:   isCorrect,
		Timestamp: ts,
	})
}

// GetAccuracy 获取某类型回答在指定窗口的准确率，冷启动返回默认值 0.75。
func (at *AccuracyTracker) GetAccuracy(answerType entity.AnswerType, window string) float64 {
	at.mu.RLock()
	defer at.mu.RUnlock()

	key := at.key(answerType)
	records := at.filterByWindow(at.records[key], window)

	if len(records) == 0 {
		return 0.75 // 冷启动默认基准值
	}

	correct := 0
	for _, r := range records {
		if r.Correct {
			correct++
		}
	}
	return float64(correct) / float64(len(records))
}

// GetStats 获取某类型回答的统计详情。
func (at *AccuracyTracker) GetStats(answerType entity.AnswerType, window string) entity.AccuracyStats {
	at.mu.RLock()
	defer at.mu.RUnlock()

	key := at.key(answerType)
	records := at.filterByWindow(at.records[key], window)

	correct := 0
	for _, r := range records {
		if r.Correct {
			correct++
		}
	}

	return entity.AccuracyStats{
		AnswerType:   answerType,
		TotalCount:   len(records),
		CorrectCount: correct,
		UpdatedAt:    time.Now().UTC(),
	}
}

// key 生成记录键（仅按 answerType）。
func (at *AccuracyTracker) key(answerType entity.AnswerType) string {
	return string(answerType)
}

// filterByWindow 按窗口过滤记录，移除过期数据。
func (at *AccuracyTracker) filterByWindow(records []accuracyRecord, window string) []accuracyRecord {
	duration, ok := windowDurationMap[window]
	if !ok {
		duration = windowDurationMap["30d"] // 无效窗口默认 30d
	}
	if duration == windowDurationMap["all"] {
		return records
	}

	cutoff := time.Now().UTC().Add(-duration)
	var filtered []accuracyRecord
	for _, r := range records {
		if r.Timestamp.After(cutoff) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
