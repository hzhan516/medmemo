// Package repository 实现数据持久化适配器。
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
)

// DisclaimerRepoSQLite 基于 SQLite 的免责声明同意记录仓库实现。
type DisclaimerRepoSQLite struct {
	db *sql.DB
}

// NewDisclaimerRepoSQLite 构造函数。
func NewDisclaimerRepoSQLite(connector database.DBConnector) *DisclaimerRepoSQLite {
	return &DisclaimerRepoSQLite{db: connector.DB()}
}

// GetAcceptance 查询用户已同意的免责声明记录。
// 若表为空（用户尚未同意），返回 nil 与 nil error。
func (r *DisclaimerRepoSQLite) GetAcceptance(ctx context.Context) (*entity.DisclaimerAcceptance, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT version, accepted_at, text_hash
		FROM disclaimer_acceptance
		ORDER BY accepted_at DESC
		LIMIT 1
	`)

	var rec entity.DisclaimerAcceptance
	var acceptedAt int64
	if err := row.Scan(&rec.Version, &acceptedAt, &rec.TextHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get disclaimer acceptance: %w", err)
	}
	rec.AcceptedAt = time.UnixMilli(acceptedAt)
	return &rec, nil
}

// SaveAcceptance 保存或更新用户的免责声明同意记录。
func (r *DisclaimerRepoSQLite) SaveAcceptance(ctx context.Context, record *entity.DisclaimerAcceptance) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO disclaimer_acceptance (version, accepted_at, text_hash)
		VALUES (?, ?, ?)
	`, record.Version, record.AcceptedAt.UnixMilli(), record.TextHash)
	if err != nil {
		return fmt.Errorf("failed to save disclaimer acceptance: %w", err)
	}
	return nil
}
