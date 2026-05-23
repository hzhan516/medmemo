// Package dto 实现数据传输对象转换层。
// 纯函数设计，无状态、无副作用，转换错误返回 error 而非 panic。
package dto

import (
	"fmt"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
)

// ConversationDTO 是对外暴露的会话数据对象。
type ConversationDTO struct {
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	Messages  []MessageDTO `json:"messages"`
	Model     string       `json:"model"`
	CreatedAt int64        `json:"created_at"`
}

type MessageDTO struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// ToConversationDTO 将领域实体转换为 DTO。
func ToConversationDTO(conv *entity.Conversation) ConversationDTO {
	msgs := make([]MessageDTO, len(conv.Messages))
	for i, m := range conv.Messages {
		msgs[i] = MessageDTO{
			Role:      string(m.Role),
			Content:   m.Content,
			Timestamp: m.Timestamp.Unix(),
		}
	}
	return ConversationDTO{
		ID:        string(conv.ID),
		Title:     conv.Title,
		Messages:  msgs,
		Model:     string(conv.Model),
		CreatedAt: conv.CreatedAt.Unix(),
	}
}

// FromMessageDTO 将 DTO 转换回领域 Message。
func FromMessageDTO(dto MessageDTO) (entity.Message, error) {
	role := models.Role(dto.Role)
	if role != models.RoleUser && role != models.RoleAssistant && role != models.RoleSystem {
		return entity.Message{}, fmt.Errorf("invalid message role: %s", dto.Role)
	}
	return entity.Message{
		Role:    role,
		Content: dto.Content,
	}, nil
}
