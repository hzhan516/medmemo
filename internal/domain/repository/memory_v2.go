package repository

import (
	"context"

	"github.com/hzhan516/medmemo/internal/domain/entity"
)

// RawDialogueRepository 定义原始对话层的持久化接口。
type RawDialogueRepository interface {
	// Insert 保存单条原始对话记录。
	Insert(ctx context.Context, d *entity.RawDialogue) error
	// InsertBatch 批量保存原始对话记录（事务内）。
	InsertBatch(ctx context.Context, dialogues []*entity.RawDialogue) error
	// GetBySession 按会话分页查询。
	GetBySession(ctx context.Context, sessionID string, offset, limit int) ([]*entity.RawDialogue, error)
	// GetRecent 获取会话最近 N 分钟内的消息。
	GetRecent(ctx context.Context, sessionID string, minutes int) ([]*entity.RawDialogue, error)
	// GetUnprocessed 获取尚未进行事实提取的消息。
	GetUnprocessed(ctx context.Context, limit int) ([]*entity.RawDialogue, error)
	// MarkProcessing 标记为处理中。
	MarkProcessing(ctx context.Context, messageID string) error
	// MarkProcessed 标记为已处理。
	MarkProcessed(ctx context.Context, messageID string) error
	// MarkFailed 标记为处理失败。
	MarkFailed(ctx context.Context, messageID string) error
}

// FactRepository 定义提取事实层的持久化接口。
type FactRepository interface {
	// Save 保存事实记录。
	Save(ctx context.Context, f *entity.ExtractedFact) error
	// GetByID 按 ID 查询事实。
	GetByID(ctx context.Context, factID string) (*entity.ExtractedFact, error)
	// ListByStatus 按审核状态分页查询。
	ListByStatus(ctx context.Context, status entity.FactStatus, offset, limit int) ([]*entity.ExtractedFact, error)
	// ListPending 获取待审核列表。
	ListPending(ctx context.Context, offset, limit int) ([]*entity.ExtractedFact, error)
	// UpdateStatus 更新审核状态。
	UpdateStatus(ctx context.Context, factID string, status entity.FactStatus) error
	// Delete 删除事实（级联删除关联嵌入）。
	Delete(ctx context.Context, factID string) error
	// GetStats 获取审核统计。
	GetStats(ctx context.Context) (total, approved, rejected, pending int64, err error)
	// ListAllSubjects 获取所有已审批事实的不重复 subject 列表。
	ListAllSubjects(ctx context.Context) ([]string, error)
	// FindBySubject 按 subject 查找已审批事实。
	FindBySubject(ctx context.Context, subject string) ([]*entity.ExtractedFact, error)
	// FindBySession 按原始对话会话 ID 查找关联的已审批事实。
	FindBySession(ctx context.Context, sessionID string) ([]*entity.ExtractedFact, error)
}

// EmbeddingRepository 定义语义嵌入层的持久化接口。
type EmbeddingRepository interface {
	// Save 保存嵌入向量。
	Save(ctx context.Context, e *entity.SemanticEmbedding) error
	// GetByFactID 按事实 ID 查询嵌入。
	GetByFactID(ctx context.Context, factID string) (*entity.SemanticEmbedding, error)
	// DeleteByFactID 按事实 ID 删除嵌入。
	DeleteByFactID(ctx context.Context, factID string) error
	// SearchSimilar 执行向量相似度搜索，返回与查询向量最相似的 topK 个嵌入。
	// 结果按余弦相似度降序排列（越靠前越相似），包含相似度分数。
	SearchSimilar(ctx context.Context, queryVector []float32, topK int) ([]*entity.ScoredEmbedding, error)
}
