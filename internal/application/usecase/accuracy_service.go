package usecase

import (
	"context"
	"fmt"

	"github.com/hzhan516/medmemo/internal/domain/repository"
)

// AccuracyService 提供回答准确率反馈的读写能力。
type AccuracyService struct {
	repo repository.AccuracyRepository
}

// NewAccuracyService 构造函数。
func NewAccuracyService(repo repository.AccuracyRepository) *AccuracyService {
	return &AccuracyService{repo: repo}
}

// GetAccuracy 返回指定回答类型的历史准确率；无数据时返回 0.75。
func (s *AccuracyService) GetAccuracy(ctx context.Context, answerType string) (float64, error) {
	if answerType == "" {
		return 0.75, nil
	}
	acc, err := s.repo.GetAccuracy(ctx, answerType)
	if err != nil {
		return 0, fmt.Errorf("failed to get accuracy for %s: %w", answerType, err)
	}
	return acc, nil
}

// RecordFeedback 记录用户对某条消息的评价。
// helpful=true 表示“有帮助”，否则标记为“不准确”。
func (s *AccuracyService) RecordFeedback(ctx context.Context, messageID, answerType string, helpful bool) error {
	if messageID == "" || answerType == "" {
		return fmt.Errorf("message_id and answer_type are required")
	}
	feedback := "inaccurate"
	if helpful {
		feedback = "helpful"
	}
	if err := s.repo.RecordFeedback(ctx, messageID, answerType, feedback); err != nil {
		return fmt.Errorf("failed to record feedback for %s/%s: %w", messageID, answerType, err)
	}
	return nil
}
