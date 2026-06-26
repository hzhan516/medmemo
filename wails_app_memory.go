package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
)

// MemoryItem 记忆列表项 DTO，供前端管理界面展示。
type MemoryItem struct {
	FactID      string  `json:"fact_id"`
	Subject     string  `json:"subject"`
	Predicate   string  `json:"predicate"`
	Object      string  `json:"object"`
	Confidence  float64 `json:"confidence"`
	Status      string  `json:"status"`
	IsSensitive bool    `json:"is_sensitive"`
	CreatedAt   int64   `json:"created_at"`
}

// MemoryStats 记忆统计 DTO。
type MemoryStats struct {
	Total    int64 `json:"total"`
	Approved int64 `json:"approved"`
	Rejected int64 `json:"rejected"`
	Pending  int64 `json:"pending"`
}

func factToMemoryItem(f *entity.ExtractedFact) MemoryItem {
	return MemoryItem{
		FactID:      f.FactID,
		Subject:     f.Subject,
		Predicate:   f.Predicate,
		Object:      f.Object,
		Confidence:  f.Confidence,
		Status:      string(f.Status),
		IsSensitive: f.IsSensitive,
		CreatedAt:   f.CreatedAt.UnixMilli(),
	}
}

// requireAuth 检查应用是否已通过首次启动的免责声明同意流程。
// 未授权时返回 ErrUnauthorized，阻止记忆数据的访问。
func (a *WailsApp) requireAuth() error {
	if a.disclaimerRepo == nil {
		return fmt.Errorf("disclaimer repository not initialized: %w", entity.ErrUnauthorized)
	}
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()
	rec, err := a.disclaimerRepo.GetAcceptance(ctx)
	if err != nil {
		return fmt.Errorf("failed to check authorization: %w", err)
	}
	if rec == nil {
		return entity.ErrUnauthorized
	}
	return nil
}

// GetMemories 分页获取已审批的记忆列表。
func (a *WailsApp) GetMemories(limit int, offset int) ([]MemoryItem, error) {
	if err := a.requireAuth(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return nil, fmt.Errorf("fact repository not initialized")
	}

	facts, err := a.factRepo.ListByStatus(ctx, entity.FactStatusApproved, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list memories: %w", err)
	}

	items := make([]MemoryItem, 0, len(facts))
	for _, f := range facts {
		items = append(items, factToMemoryItem(f))
	}
	return items, nil
}

// GetMemoryByID 按 ID 获取单条记忆详情。
func (a *WailsApp) GetMemoryByID(factID string) (MemoryItem, error) {
	if err := a.requireAuth(); err != nil {
		return MemoryItem{}, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return MemoryItem{}, fmt.Errorf("fact repository not initialized")
	}

	f, err := a.factRepo.GetByID(ctx, factID)
	if err != nil {
		return MemoryItem{}, fmt.Errorf("failed to get memory: %w", err)
	}
	return factToMemoryItem(f), nil
}

// DeleteMemory 删除指定记忆（级联删除关联嵌入）。
func (a *WailsApp) DeleteMemory(factID string) error {
	if err := a.requireAuth(); err != nil {
		return fmt.Errorf("requireAuth failed: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return fmt.Errorf("fact repository not initialized")
	}

	if err := a.factRepo.Delete(ctx, factID); err != nil {
		return fmt.Errorf("failed to delete memory: %w", err)
	}

	// 记录审计日志（失败不影响主流程）
	if a.auditLogRepo != nil {
		entry := entity.NewAuditLogEntry(entity.AuditActionDelete, "fact", factID, "user")
		_ = a.auditLogRepo.Save(ctx, entry)
	}
	return nil
}

// SearchMemories 关键词搜索已审批的记忆。
// 使用数据库层 LIKE 过滤，避免一次性加载全部 approved 事实到内存。
func (a *WailsApp) SearchMemories(query string) ([]MemoryItem, error) {
	if err := a.requireAuth(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return nil, fmt.Errorf("fact repository not initialized")
	}

	facts, err := a.factRepo.SearchApproved(ctx, strings.TrimSpace(query), 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}

	items := make([]MemoryItem, 0, len(facts))
	for _, f := range facts {
		items = append(items, factToMemoryItem(f))
	}
	return items, nil
}

// GetPendingReviews 获取待审核事实列表。
func (a *WailsApp) GetPendingReviews(limit int, offset int) ([]MemoryItem, error) {
	if err := a.requireAuth(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return nil, fmt.Errorf("fact repository not initialized")
	}

	facts, err := a.factRepo.ListPending(ctx, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending reviews: %w", err)
	}

	items := make([]MemoryItem, 0, len(facts))
	for _, f := range facts {
		items = append(items, factToMemoryItem(f))
	}
	return items, nil
}

// ApproveFact 审核通过指定事实。
func (a *WailsApp) ApproveFact(factID string) error {
	if err := a.requireAuth(); err != nil {
		return fmt.Errorf("requireAuth failed: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return fmt.Errorf("fact repository not initialized")
	}

	if err := a.factRepo.UpdateStatus(ctx, factID, entity.FactStatusApproved); err != nil {
		return fmt.Errorf("failed to approve fact: %w", err)
	}

	// 审批通过后生成语义嵌入（向量索引），供记忆召回使用
	if a.embeddingSvc != nil && a.embeddingRepo != nil {
		fact, err := a.factRepo.GetByID(ctx, factID)
		if err == nil && fact != nil {
			content := usecase.BuildFactRetrievalText(fact)
			vector, embErr := a.embeddingSvc.EmbedSingle(ctx, content)
			if embErr == nil {
				embedding := entity.NewSemanticEmbedding(factID, vector, models.CurrentEmbeddingVersion)
				if saveErr := a.embeddingRepo.Save(ctx, embedding); saveErr != nil {
					fmt.Printf("[ApproveFact] 保存嵌入向量失败 %s: %v\n", factID, saveErr)
				}
			} else {
				fmt.Printf("[ApproveFact] 生成嵌入向量失败 %s: %v\n", factID, embErr)
			}
		}
	}

	// 记录审计日志（失败不影响主流程）
	if a.auditLogRepo != nil {
		entry := entity.NewAuditLogEntry(entity.AuditActionApprove, "fact", factID, "user")
		_ = a.auditLogRepo.Save(ctx, entry)
	}
	return nil
}

// RejectFact 审核拒绝指定事实。
func (a *WailsApp) RejectFact(factID string) error {
	if err := a.requireAuth(); err != nil {
		return fmt.Errorf("requireAuth failed: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return fmt.Errorf("fact repository not initialized")
	}

	if err := a.factRepo.UpdateStatus(ctx, factID, entity.FactStatusRejected); err != nil {
		return fmt.Errorf("failed to reject fact: %w", err)
	}

	// 拒绝时清理可能已存在的 embedding，避免历史版本或异常写入留下的 stale 向量
	if a.embeddingRepo != nil {
		if delErr := a.embeddingRepo.DeleteByFactID(ctx, factID); delErr != nil {
			fmt.Printf("[RejectFact] 清理 embedding 失败 %s: %v\n", factID, delErr)
		}
	}

	// 记录审计日志（失败不影响主流程）
	if a.auditLogRepo != nil {
		entry := entity.NewAuditLogEntry(entity.AuditActionReject, "fact", factID, "user")
		_ = a.auditLogRepo.Save(ctx, entry)
	}
	return nil
}

// GetMemoryStats 获取记忆审核统计。
func (a *WailsApp) GetMemoryStats() (MemoryStats, error) {
	if err := a.requireAuth(); err != nil {
		return MemoryStats{}, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return MemoryStats{}, fmt.Errorf("fact repository not initialized")
	}

	total, approved, rejected, pending, err := a.factRepo.GetStats(ctx)
	if err != nil {
		return MemoryStats{}, fmt.Errorf("failed to get memory stats: %w", err)
	}
	return MemoryStats{
		Total:    total,
		Approved: approved,
		Rejected: rejected,
		Pending:  pending,
	}, nil
}

// GetMemoriesBySession 按会话 ID 获取关联的已审批记忆。
func (a *WailsApp) GetMemoriesBySession(sessionID string) ([]MemoryItem, error) {
	if err := a.requireAuth(); err != nil {
		return nil, err
	}
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.factRepo == nil {
		return nil, fmt.Errorf("fact repository not initialized")
	}

	facts, err := a.factRepo.FindBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get memories by session: %w", err)
	}

	items := make([]MemoryItem, 0, len(facts))
	for _, f := range facts {
		items = append(items, factToMemoryItem(f))
	}
	return items, nil
}

// SetMemoryInjectionEnabled 设置记忆注入全局开关。
func (a *WailsApp) SetMemoryInjectionEnabled(enabled bool) error {
	if err := a.requireAuth(); err != nil {
		return fmt.Errorf("requireAuth failed: %w", err)
	}
	if a.memoryRetriever == nil {
		return fmt.Errorf("memory retriever not initialized")
	}
	a.memoryRetriever.SetEnabled(enabled)
	return nil
}

// SetSessionMemoryInjection 设置指定会话的记忆注入开关。
func (a *WailsApp) SetSessionMemoryInjection(sessionID string, enabled bool) error {
	if err := a.requireAuth(); err != nil {
		return fmt.Errorf("requireAuth failed: %w", err)
	}
	if a.memoryRetriever == nil {
		return fmt.Errorf("memory retriever not initialized")
	}
	a.memoryRetriever.SetSessionEnabled(sessionID, enabled)
	return nil
}
