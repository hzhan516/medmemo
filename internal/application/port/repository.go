package port

import (
	"context"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
)

// ProviderStore 定义 Provider 配置的持久化接口。
type ProviderStore interface {
	// Create 创建新的 Provider 配置。
	Create(ctx context.Context, provider *models.ProviderConfig) error
	// Update 更新已有 Provider 配置。
	Update(ctx context.Context, provider *models.ProviderConfig) error
	// Delete 删除指定 Provider 配置。
	Delete(ctx context.Context, id string) error
	// Get 按 ID 查询 Provider 配置。
	Get(ctx context.Context, id string) (*models.ProviderConfig, error)
	// List 查询全部 Provider 配置，按 sort_order ASC, updated_at DESC 排序。
	List(ctx context.Context) ([]*models.ProviderConfig, error)
}

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
	Restore(ctx context.Context, id models.ConversationID) error
	HardDelete(ctx context.Context, id models.ConversationID) error
	ArchiveOlderThan(ctx context.Context, cutoff time.Time) error
	PermanentlyDeleteOlderThan(ctx context.Context, cutoff time.Time) error
	// UpdateTimestamp 仅更新会话的 updated_at 时间戳，不覆盖其他字段。
	UpdateTimestamp(ctx context.Context, id models.ConversationID, updatedAt time.Time) error
	// UpdateTitle 仅更新会话的标题，不覆盖其他字段。
	UpdateTitle(ctx context.Context, id models.ConversationID, title string) error
}

// DisclaimerRepository 定义免责声明同意记录的持久化接口。
type DisclaimerRepository interface {
	// GetAcceptance 查询用户已同意的免责声明记录。
	// 若用户尚未同意，返回 nil 与 nil error。
	GetAcceptance(ctx context.Context) (*entity.DisclaimerAcceptance, error)
	// SaveAcceptance 保存或更新用户的免责声明同意记录。
	SaveAcceptance(ctx context.Context, record *entity.DisclaimerAcceptance) error
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
