package repository

import (
	"context"

	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/pkg/models"
)

// ConversationRepository 定义会话的持久化接口。
type ConversationRepository interface {
	Save(ctx context.Context, conv *entity.Conversation) error
	GetByID(ctx context.Context, id models.ConversationID) (*entity.Conversation, error)
	ListRecent(ctx context.Context, limit int) ([]*entity.Conversation, error)
	Delete(ctx context.Context, id models.ConversationID) error
}
