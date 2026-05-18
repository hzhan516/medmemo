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
