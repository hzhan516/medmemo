package port

import (
	"context"

	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/pkg/models"
)

// 将领域层定义的仓库接口在应用层重新导出，
// 符合 Clean Architecture 中"接口由消费者端声明"的原则。

type MemoryRepository interface {
	Save(ctx context.Context, mem *entity.HealthMemory) error
	GetByID(ctx context.Context, id models.MemoryID) (*entity.HealthMemory, error)
	Search(ctx context.Context, query string, limit int) ([]*entity.HealthMemory, error)
	SemanticSearch(ctx context.Context, embedding []float32, topK int) ([]*entity.HealthMemory, error)
	ListByTier(ctx context.Context, tier entity.MemoryTier, limit int) ([]*entity.HealthMemory, error)
	Delete(ctx context.Context, id models.MemoryID) error
}

type FamilyRepository interface {
	SaveMember(ctx context.Context, member *entity.FamilyMember) error
	GetMemberByID(ctx context.Context, id models.MemberID) (*entity.FamilyMember, error)
	ListAllMembers(ctx context.Context) ([]*entity.FamilyMember, error)
	DeleteMember(ctx context.Context, id models.MemberID) error
	FindRelations(ctx context.Context, id models.MemberID) ([]entity.FamilyRelation, error)
	FindByDisease(ctx context.Context, diseaseName string) ([]*entity.FamilyMember, error)
}

type ConversationRepository interface {
	Save(ctx context.Context, conv *entity.Conversation) error
	GetByID(ctx context.Context, id models.ConversationID) (*entity.Conversation, error)
	ListRecent(ctx context.Context, limit int) ([]*entity.Conversation, error)
	Delete(ctx context.Context, id models.ConversationID) error
}

// MessageRepository 定义消息的持久化接口。
type MessageRepository interface {
	// Save 保存单条消息。
	Save(ctx context.Context, convID models.ConversationID, msg *entity.Message) error
	// ListByConversation 按会话查询消息，支持 cursor-based 分页。
	// 返回消息列表和下一页 cursor（空字符串表示无更多数据）。
	ListByConversation(ctx context.Context, convID models.ConversationID, cursor string, limit int) ([]*entity.Message, string, error)
	// SoftDelete 软删除消息。
	SoftDelete(ctx context.Context, msgID string) error
	// Restore 恢复软删除的消息。
	Restore(ctx context.Context, msgID string) error
}
