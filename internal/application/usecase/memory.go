package usecase

import (
	"context"
	"fmt"

	"github.com/google/wire"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
)

// MemoryRetriever 编排记忆检索与归档用例。
type MemoryRetriever struct {
	repo port.MemoryRepository
}

// NewMemoryRetriever 构造函数。
func NewMemoryRetriever(repo port.MemoryRepository) *MemoryRetriever {
	return &MemoryRetriever{repo: repo}
}

// RetrieveForContext 为当前对话检索相关记忆，返回用于注入上下文的记忆片段。
func (m *MemoryRetriever) RetrieveForContext(ctx context.Context, query string, limit int) ([]*entity.HealthMemory, error) {
	memories, err := m.repo.Search(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve memories for context: %w", err)
	}
	return memories, nil
}

// ArchiveConversation 将对话归档为长期记忆（L2/L3）。
func (m *MemoryRetriever) ArchiveConversation(ctx context.Context, convID models.ConversationID) error {
	// TODO(作者): 实现对话摘要与实体提取后的记忆归档 [Issue#004]
	return fmt.Errorf("not implemented")
}

var _ = wire.NewSet // 占位，确保 wire 包被引用
