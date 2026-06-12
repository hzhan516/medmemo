package usecase

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
)

// MigrationState 追踪 embedding 迁移完成状态。
// 采用内存 atomic bool，配合启动时快速 COUNT 检查，避免持久化额外状态表。
type MigrationState struct {
	complete atomic.Bool
}

// NewMigrationState 创建迁移状态追踪器。
func NewMigrationState() *MigrationState {
	return &MigrationState{}
}

// IsComplete 返回迁移是否已完成（所有 approved fact 的 embedding 均已升级到当前版本）。
func (s *MigrationState) IsComplete() bool {
	return s.complete.Load()
}

// SetComplete 将迁移状态置为完成。
func (s *MigrationState) SetComplete(v bool) {
	s.complete.Store(v)
}

// EmbeddingMigrator 负责在版本升级后重建过期/缺失的 embedding。
type EmbeddingMigrator struct {
	factRepo      repository.FactRepository
	embeddingRepo repository.EmbeddingRepository
	embeddingSvc  port.EmbeddingService
	currentVer    string
	state         *MigrationState
	batchPause    time.Duration
	pageSize      int
}

// NewEmbeddingMigrator 创建 embedding 迁移器。
func NewEmbeddingMigrator(
	factRepo repository.FactRepository,
	embeddingRepo repository.EmbeddingRepository,
	embeddingSvc port.EmbeddingService,
	state *MigrationState,
) *EmbeddingMigrator {
	return &EmbeddingMigrator{
		factRepo:      factRepo,
		embeddingRepo: embeddingRepo,
		embeddingSvc:  embeddingSvc,
		currentVer:    embeddingSvc.ModelVersion(),
		state:         state,
		batchPause:    50 * time.Millisecond,
		pageSize:      500,
	}
}

// NeedsMigration 快速检测是否需要迁移（启动时调用）。
// 使用 LEFT JOIN 查询，同时覆盖旧版本 embedding 和缺失 embedding 两种情况。
func (m *EmbeddingMigrator) NeedsMigration(ctx context.Context) (bool, int64, error) {
	count, err := m.factRepo.CountApprovedFactsNeedingEmbedding(ctx, m.currentVer)
	if err != nil {
		return false, 0, fmt.Errorf("failed to count facts needing embedding: %w", err)
	}
	return count > 0, count, nil
}

// RunMigration 扫描并重建过期/缺失的 embedding。
//
// progressFn 在每条 fact 处理后回调，报告 (已处理, 总数)。
// 返回 (已处理数, 失败数, error)。error 仅在系统级故障时返回，单条 fact 失败仅记日志。
func (m *EmbeddingMigrator) RunMigration(
	ctx context.Context,
	progressFn func(processed, total int),
) (int, int, error) {
	total, err := m.factRepo.CountApprovedFactsNeedingEmbedding(ctx, m.currentVer)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count migration candidates: %w", err)
	}
	if total == 0 {
		m.state.complete.Store(true)
		return 0, 0, nil
	}

	var processed, failed int
	var lastCreatedAt time.Time
	var lastFactID string

	for {
		if ctx.Err() != nil {
			return processed, failed, fmt.Errorf("migration interrupted: %w", ctx.Err())
		}

		facts, err := m.factRepo.ListApprovedFactsNeedingEmbedding(ctx, m.currentVer, lastCreatedAt, lastFactID, m.pageSize)
		if err != nil {
			return processed, failed, fmt.Errorf("failed to list migration candidates: %w", err)
		}
		if len(facts) == 0 {
			break
		}

		for _, fact := range facts {
			if err := m.processFact(ctx, fact); err != nil {
				fmt.Printf("[EmbeddingMigration] 处理 fact %s 失败: %v\n", fact.FactID, err)
				failed++
			} else {
				processed++
			}
			if progressFn != nil {
				progressFn(processed+failed, int(total))
			}
		}

		last := facts[len(facts)-1]
		lastCreatedAt = last.CreatedAt
		lastFactID = last.FactID

		if m.batchPause > 0 {
			select {
			case <-time.After(m.batchPause):
			case <-ctx.Done():
				return processed, failed, fmt.Errorf("migration interrupted during batch pause: %w", ctx.Err())
			}
		}
	}

	// 严格置位条件：失败数为 0 直接完成；有失败时二次 COUNT 确认待迁移数为 0 才置位。
	if failed == 0 {
		m.state.complete.Store(true)
	} else {
		recheckCount, err := m.factRepo.CountApprovedFactsNeedingEmbedding(ctx, m.currentVer)
		if err == nil && recheckCount == 0 {
			m.state.complete.Store(true)
		}
		// 否则保持 false，下次启动重试失败项；同时 MemoryRetriever 继续使用 SearchSimilar（所有版本）
	}

	return processed, failed, nil
}

// processFact 单条 fact 的 re-embedding 处理。
func (m *EmbeddingMigrator) processFact(ctx context.Context, fact *entity.ExtractedFact) error {
	text := BuildFactRetrievalText(fact)
	vector, err := m.embeddingSvc.EmbedSingle(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to embed fact %s: %w", fact.FactID, err)
	}

	existing, getErr := m.embeddingRepo.GetByFactID(ctx, fact.FactID)
	if getErr == nil && existing != nil {
		// 保留原 embedding_id，只更新向量/版本/时间戳
		updated := entity.NewSemanticEmbedding(fact.FactID, vector, m.currentVer)
		updated.EmbeddingID = existing.EmbeddingID
		if err := m.embeddingRepo.UpdateEmbedding(ctx, updated); err != nil {
			return fmt.Errorf("failed to update embedding for fact %s: %w", fact.FactID, err)
		}
		return nil
	}

	// 无旧 embedding则新建
	newEmb := entity.NewSemanticEmbedding(fact.FactID, vector, m.currentVer)
	if err := m.embeddingRepo.Save(ctx, newEmb); err != nil {
		return fmt.Errorf("failed to save embedding for fact %s: %w", fact.FactID, err)
	}
	return nil
}
