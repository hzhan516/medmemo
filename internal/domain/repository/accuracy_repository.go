package repository

import "context"

// AccuracyRepository 定义回答准确率反馈的持久化接口。
type AccuracyRepository interface {
	// GetAccuracy 返回指定回答类型的历史准确率；无数据时返回冷启动默认值 0.75。
	GetAccuracy(ctx context.Context, answerType string) (float64, error)
	// RecordFeedback 记录一条反馈；同一 (message_id, answer_type) 重复反馈会覆盖旧值并重新统计准确率。
	RecordFeedback(ctx context.Context, messageID, answerType, feedback string) error
}
