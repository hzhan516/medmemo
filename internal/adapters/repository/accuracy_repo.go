// Package repository 实现数据持久化适配器。
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
)

// AccuracyRepoSQLite 基于 SQLite/SQLCipher 的回答准确率反馈实现。
type AccuracyRepoSQLite struct {
	db *sql.DB
}

// NewAccuracyRepoSQLite 构造函数。
func NewAccuracyRepoSQLite(connector database.DBConnector) *AccuracyRepoSQLite {
	return &AccuracyRepoSQLite{db: connector.DB()}
}

// GetAccuracy 查询 answer_accuracy_stats；无数据时返回 0.75 冷启动默认值。
func (r *AccuracyRepoSQLite) GetAccuracy(ctx context.Context, answerType string) (float64, error) {
	var correctCount, totalCount int
	err := r.db.QueryRowContext(ctx, `
		SELECT correct_count, total_count
		FROM answer_accuracy_stats
		WHERE answer_type = ?
	`, answerType).Scan(&correctCount, &totalCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0.75, nil
		}
		return 0, fmt.Errorf("failed to get accuracy stats for %s: %w", answerType, err)
	}
	if totalCount == 0 {
		return 0.75, nil
	}
	return float64(correctCount) / float64(totalCount), nil
}

// RecordFeedback 以 (message_id, answer_type) 为唯一键记录反馈。
// 重复反馈会覆盖旧值，并基于最新的 feedback 行重新计算该 answer_type 的准确率统计。
func (r *AccuracyRepoSQLite) RecordFeedback(ctx context.Context, messageID, answerType, feedback string) error {
	if feedback != "helpful" && feedback != "inaccurate" {
		return fmt.Errorf("invalid feedback value %q: must be 'helpful' or 'inaccurate'", feedback)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin accuracy feedback transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UnixMilli()
	if _, err := tx.ExecContext(ctx, `
		INSERT OR REPLACE INTO answer_feedback (message_id, answer_type, feedback, created_at)
		VALUES (?, ?, ?, ?)
	`, messageID, answerType, feedback, now); err != nil {
		return fmt.Errorf("failed to upsert answer feedback: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT OR REPLACE INTO answer_accuracy_stats (answer_type, correct_count, total_count, updated_at)
		VALUES (
			?,
			(SELECT COUNT(*) FROM answer_feedback WHERE answer_type = ? AND feedback = 'helpful'),
			(SELECT COUNT(*) FROM answer_feedback WHERE answer_type = ?),
			?
		)
	`, answerType, answerType, answerType, now); err != nil {
		return fmt.Errorf("failed to recompute accuracy stats: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit accuracy feedback transaction: %w", err)
	}
	return nil
}

// Compile-time interface check.
var _ repository.AccuracyRepository = (*AccuracyRepoSQLite)(nil)
