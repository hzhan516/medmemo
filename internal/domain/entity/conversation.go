// Package entity 定义核心业务实体与领域规则。
// 严格遵守零外部依赖铁律：仅允许 Go 标准库和 pkg/models/。
package entity

import (
	"fmt"
	"time"

	"github.com/hzhan516/medmemo/pkg/models"
)

// Conversation 表示一次用户与 AI 的对话会话。
type Conversation struct {
	ID        models.ConversationID
	Title     string
	Messages  []Message
	Model     models.ProviderType
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time // 软删除时间戳，nil 表示未删除
}

// Message 是会话中的单条消息，封装 models.Message 并附加领域元数据。
type Message struct {
	ID         string
	Role       models.Role
	Content    string
	Timestamp  time.Time
	IsModified bool // 标记该消息是否被用户编辑过，用于记忆一致性追踪。
}

// NewConversation 创建新会话，标题默认为空，由第一条消息自动生成。
func NewConversation(model models.ProviderType) *Conversation {
	now := time.Now().UTC()
	return &Conversation{
		ID:        models.ConversationID(fmt.Sprintf("conv_%d", now.UnixNano())),
		Messages:  make([]Message, 0),
		Model:     model,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddMessage 向会话追加消息并更新时间戳。
func (c *Conversation) AddMessage(role models.Role, content string) {
	c.Messages = append(c.Messages, Message{
		ID:        fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		Role:      role,
		Content:   content,
		Timestamp: time.Now().UTC(),
	})
	c.UpdatedAt = time.Now().UTC()
}

// Rename 重命名会话标题。
func (c *Conversation) Rename(title string) {
	c.Title = title
	c.UpdatedAt = time.Now().UTC()
}
