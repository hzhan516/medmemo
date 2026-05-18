package entity

import (
	"fmt"
	"time"

	"github.com/medmemo/medmemo/pkg/models"
)

// MemoryTier 表示记忆层级（L1/L2/L3）。
type MemoryTier int

const (
	TierWorking   MemoryTier = iota + 1 // L1 工作记忆：当前会话上下文
	TierShortTerm                       // L2 短期记忆：近期对话归档
	TierLongTerm                        // L3 长期记忆：持久化知识图谱与向量索引
)

// HealthMemory 表示一条健康相关的结构化记忆单元。
type HealthMemory struct {
	ID         models.MemoryID
	Tier       MemoryTier
	Content    string
	Tags       []string // 如 "症状", "诊断", "用药"
	SourceConv models.ConversationID
	Confidence float64 // 0-1，记忆可信度
	CreatedAt  time.Time
	AccessedAt time.Time // 最近访问时间，用于时间衰减计算
}

// NewHealthMemory 创建新的健康记忆。
func NewHealthMemory(tier MemoryTier, content string, source models.ConversationID) *HealthMemory {
	now := time.Now()
	return &HealthMemory{
		ID:         models.MemoryID(fmt.Sprintf("mem_%d", now.UnixNano())),
		Tier:       tier,
		Content:    content,
		Tags:       make([]string, 0),
		SourceConv: source,
		Confidence: 1.0,
		CreatedAt:  now,
		AccessedAt: now,
	}
}

// MarkAccessed 更新访问时间，用于检索时的时间衰减权重计算。
func (m *HealthMemory) MarkAccessed() {
	m.AccessedAt = time.Now()
}
