package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/config"
	"github.com/hzhan516/medmemo/pkg/models"
)

// ConversationSummary 会话摘要。
type ConversationSummary struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	UpdatedAt string `json:"updated_at"`
	DeletedAt string `json:"deleted_at"` // 空字符串表示未删除，否则为软删除时间戳（毫秒）
}

// MessageResponse 单条消息响应。
type MessageResponse struct {
	ID               string                 `json:"id"`
	Role             string                 `json:"role"`
	Content          string                 `json:"content"`
	Timestamp        string                 `json:"timestamp"`
	PromptTokens     int                    `json:"prompt_tokens"`
	CompletionTokens int                    `json:"completion_tokens"`
	TotalTokens      int                    `json:"total_tokens"`
	Confidence       map[string]interface{} `json:"confidence,omitempty"`
}

// GetConversationMessages 获取指定会话的全部消息。
func (a *WailsApp) GetConversationMessages(convID string) ([]MessageResponse, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.msgRepo == nil {
		return nil, fmt.Errorf("message repository not initialized")
	}

	msgs, _, err := a.msgRepo.ListByConversation(ctx, models.ConversationID(convID), "", 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages for conversation %s: %w", convID, err)
	}

	if len(msgs) == 0 {
		fmt.Printf("[GetConversationMessages] 会话 %s 无消息\n", convID)
	}

	// ListByConversation 返回 created_at DESC（最新的在前），需要反转为正序
	result := make([]MessageResponse, len(msgs))
	for i, m := range msgs {
		mr := MessageResponse{
			ID:               m.ID,
			Role:             string(m.Role),
			Content:          m.Content,
			Timestamp:        strconv.FormatInt(m.Timestamp.UnixMilli(), 10),
			PromptTokens:     m.PromptTokens,
			CompletionTokens: m.CompletionTokens,
			TotalTokens:      m.PromptTokens + m.CompletionTokens,
		}
		if m.ConfidenceJSON != "" {
			// 先反序列化到实体结构（兼容旧数据的大驼峰和新数据的蛇形）
			var cr entity.ConfidenceResult
			if err := json.Unmarshal([]byte(m.ConfidenceJSON), &cr); err == nil {
				mr.Confidence = confidenceResultToMap(&cr)
				// 防御性修复：若 JSON 解析后 overall_score 为 0 但 ConfidenceScore > 0，用 ConfidenceScore 覆盖
				if cr.OverallScore == 0 && m.ConfidenceScore > 0 {
					mr.Confidence["overall_score"] = m.ConfidenceScore
				}
			} else {
				// 降级：直接作为 map 透传
				var conf map[string]interface{}
				if err := json.Unmarshal([]byte(m.ConfidenceJSON), &conf); err == nil {
					mr.Confidence = conf
				}
			}
		}
		result[len(msgs)-1-i] = mr
	}
	return result, nil
}

// GetConversations 获取会话列表。
func (a *WailsApp) GetConversations() (result []ConversationSummary, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[panic] GetConversations: %v\n", r)
			err = fmt.Errorf("internal error: %v", r)
		}
	}()
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.convRepo == nil {
		return nil, fmt.Errorf("conversation repository not initialized")
	}

	convs, err := a.convRepo.ListRecent(ctx, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	result = make([]ConversationSummary, len(convs))
	for i, conv := range convs {
		summary := ConversationSummary{
			ID:        string(conv.ID),
			Title:     conv.Title,
			UpdatedAt: strconv.FormatInt(conv.UpdatedAt.UnixMilli(), 10),
		}
		if conv.DeletedAt != nil {
			summary.DeletedAt = strconv.FormatInt(conv.DeletedAt.UnixMilli(), 10)
		}
		result[i] = summary
	}
	return result, nil
}

// GetDeletedConversations 获取已软删除的会话列表（回收站）。
func (a *WailsApp) GetDeletedConversations() (result []ConversationSummary, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[panic] GetDeletedConversations: %v\n", r)
			err = fmt.Errorf("internal error: %v", r)
		}
	}()
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.convRepo == nil {
		return nil, fmt.Errorf("conversation repository not initialized")
	}

	convs, err := a.convRepo.ListDeleted(ctx, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to list deleted conversations: %w", err)
	}

	result = make([]ConversationSummary, len(convs))
	for i, conv := range convs {
		summary := ConversationSummary{
			ID:        string(conv.ID),
			Title:     conv.Title,
			UpdatedAt: strconv.FormatInt(conv.UpdatedAt.UnixMilli(), 10),
		}
		if conv.DeletedAt != nil {
			summary.DeletedAt = strconv.FormatInt(conv.DeletedAt.UnixMilli(), 10)
		}
		result[i] = summary
	}
	return result, nil
}

// SetDataRetentionDays 设置数据留存期限并持久化到配置文件。
func (a *WailsApp) SetDataRetentionDays(days int) error {
	if days < 0 {
		return fmt.Errorf("data retention days must be non-negative")
	}
	a.config.DataRetentionDays = days
	if err := config.SaveDataRetentionDays(days); err != nil {
		return fmt.Errorf("failed to save data retention days: %w", err)
	}
	return nil
}

// SetDesensitizationLevel 设置脱敏级别并持久化到配置文件。
// 仅接受标准小写值 standard / strict / off；非法值将返回错误，不会写入配置。
func (a *WailsApp) SetDesensitizationLevel(level string) error {
	lvl, ok := models.CanonicalizeDesensitizationLevel(level)
	if !ok {
		return fmt.Errorf("invalid desensitization level: %s (must be standard, strict, or off)", level)
	}
	a.config.DesensitizationLevel = lvl
	if err := config.SaveDesensitizationLevel(string(lvl)); err != nil {
		return fmt.Errorf("failed to save desensitization level: %w", err)
	}
	return nil
}

// DeleteConversation 软删除指定会话（移入回收站）。
func (a *WailsApp) DeleteConversation(convID string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.convRepo == nil {
		return fmt.Errorf("conversation repository not initialized")
	}
	if err := a.convRepo.Delete(ctx, models.ConversationID(convID)); err != nil {
		return fmt.Errorf("failed to delete conversation %s: %w", convID, err)
	}
	return nil
}

// RestoreConversation 恢复软删除的会话。
func (a *WailsApp) RestoreConversation(convID string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.convRepo == nil {
		return fmt.Errorf("conversation repository not initialized")
	}
	if err := a.convRepo.Restore(ctx, models.ConversationID(convID)); err != nil {
		return fmt.Errorf("failed to restore conversation %s: %w", convID, err)
	}
	return nil
}

// HardDeleteConversation 永久删除指定会话及其消息。
func (a *WailsApp) HardDeleteConversation(convID string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	if a.convRepo == nil {
		return fmt.Errorf("conversation repository not initialized")
	}
	if err := a.convRepo.HardDelete(ctx, models.ConversationID(convID)); err != nil {
		return fmt.Errorf("failed to hard delete conversation %s: %w", convID, err)
	}
	return nil
}

// runRetentionCleanup 执行数据留存自动归档与清理。
func (a *WailsApp) runRetentionCleanup() {
	retentionDays := a.config.DataRetentionDays
	if retentionDays <= 0 {
		return // 永久保留，不执行清理
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	if a.convRepo != nil {
		// 自动归档：主列表中超过期限的会话 → 移入回收站
		if err := a.convRepo.ArchiveOlderThan(ctx, cutoff); err != nil {
			fmt.Printf("[retention] 自动归档失败: %v\n", err)
		}
		// 自动清理：回收站中超过期限的会话 → 物理删除
		if err := a.convRepo.PermanentlyDeleteOlderThan(ctx, cutoff); err != nil {
			fmt.Printf("[retention] 自动清理失败: %v\n", err)
		}
	}
}

// CreateConversation 创建新会话，返回会话 ID。
func (a *WailsApp) CreateConversation() (string, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	conv := entity.NewConversation(models.ProviderType(a.config.DefaultModel))
	if err := a.convRepo.Save(ctx, conv); err != nil {
		return "", fmt.Errorf("failed to create conversation: %w", err)
	}
	return string(conv.ID), nil
}
