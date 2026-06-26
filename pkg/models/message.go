// Package models 定义跨层共享的基础数据模型。
// 该包仅依赖 Go 标准库，可被 internal/domain/ 引用。
package models

// Role 表示对话消息的角色类型。
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// Message 表示单条对话消息，用于 LLM 上下文传递。
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// ConversationID 会话唯一标识。
type ConversationID string

// MemoryID 记忆单元唯一标识。
type MemoryID string

// MemberID 家族成员唯一标识。
type MemberID string
