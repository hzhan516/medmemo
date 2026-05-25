package entity

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

// =============================================================================
// Layer 1: 原始对话 (RawDialogue)
// =============================================================================

// DialogueRole 表示对话消息的角色类型。
type DialogueRole string

const (
	RoleUser      DialogueRole = "user"
	RoleAssistant DialogueRole = "assistant"
	RoleSystem    DialogueRole = "system"
)

// ExtractionStatus 表示消息的事实提取状态。
type ExtractionStatus string

const (
	ExtractionStatusUnprocessed ExtractionStatus = "unprocessed"
	ExtractionStatusProcessing  ExtractionStatus = "processing"
	ExtractionStatusProcessed   ExtractionStatus = "processed"
	ExtractionStatusFailed      ExtractionStatus = "failed"
)

// RawDialogue 表示 Layer 1 的原始对话消息实体。
type RawDialogue struct {
	MessageID         string
	SessionID         string
	Role              DialogueRole
	Content           string
	ModelName         string
	Timestamp         time.Time
	ExtractionStatus  ExtractionStatus
	CreatedAt         time.Time
}

// NewRawDialogue 创建新的原始对话记录。
func NewRawDialogue(sessionID string, role DialogueRole, content, modelName string) *RawDialogue {
	now := time.Now().UTC()
	return &RawDialogue{
		MessageID:        fmt.Sprintf("msg_%d", now.UnixNano()),
		SessionID:        sessionID,
		Role:             role,
		Content:          content,
		ModelName:        modelName,
		Timestamp:        now,
		ExtractionStatus: ExtractionStatusUnprocessed,
		CreatedAt:        now,
	}
}

// MarkProcessing 将提取状态标记为处理中。
func (d *RawDialogue) MarkProcessing() {
	d.ExtractionStatus = ExtractionStatusProcessing
}

// MarkProcessed 将提取状态标记为已处理。
func (d *RawDialogue) MarkProcessed() {
	d.ExtractionStatus = ExtractionStatusProcessed
}

// MarkFailed 将提取状态标记为处理失败。
func (d *RawDialogue) MarkFailed() {
	d.ExtractionStatus = ExtractionStatusFailed
}

// =============================================================================
// Layer 2: 提取事实 (ExtractedFact)
// =============================================================================

// FactStatus 表示事实的审核状态。
type FactStatus string

const (
	FactStatusPending  FactStatus = "pending"
	FactStatusApproved FactStatus = "approved"
	FactStatusRejected FactStatus = "rejected"
)

// ExtractedFact 表示 Layer 2 的结构化事实三元组实体。
type ExtractedFact struct {
	FactID       string
	Subject      string
	Predicate    string
	Object       string
	Confidence   float64
	SourceMsgIDs []string
	Status       FactStatus
	ScoredAt     *time.Time
	ReviewedAt   *time.Time
	CreatedAt    time.Time
}

// NewExtractedFact 创建新的事实记录。
func NewExtractedFact(subject, predicate, object string, confidence float64, sourceMsgIDs []string) *ExtractedFact {
	now := time.Now().UTC()
	return &ExtractedFact{
		FactID:       fmt.Sprintf("fact_%d", now.UnixNano()),
		Subject:      subject,
		Predicate:    predicate,
		Object:       object,
		Confidence:   confidence,
		SourceMsgIDs: sourceMsgIDs,
		Status:       FactStatusPending,
		CreatedAt:    now,
	}
}

// Validate 验证事实三元组的完整性。
func (f *ExtractedFact) Validate() error {
	if f.Subject == "" || f.Predicate == "" || f.Object == "" {
		return ErrInvalidFact
	}
	if len(f.SourceMsgIDs) == 0 {
		return ErrInvalidFact
	}
	if f.Confidence < 0 || f.Confidence > 1 {
		return ErrInvalidConfidence
	}
	return nil
}

// SetStatus 设置审核状态并记录审核时间。
func (f *ExtractedFact) SetStatus(status FactStatus) {
	now := time.Now().UTC()
	f.Status = status
	f.ReviewedAt = &now
}

// SetScoredAt 记录评分时间。
func (f *ExtractedFact) SetScoredAt() {
	now := time.Now().UTC()
	f.ScoredAt = &now
}

// =============================================================================
// Layer 3: 语义嵌入 (SemanticEmbedding)
// =============================================================================

const EmbeddingDimension = 384

// ScoredEmbedding 封装语义嵌入及其相似度分数（由向量搜索返回）。
type ScoredEmbedding struct {
	*SemanticEmbedding
	Similarity float64 // 余弦相似度，范围 [0,1]
}

// SemanticEmbedding 表示 Layer 3 的语义向量嵌入实体。
type SemanticEmbedding struct {
	EmbeddingID  string
	FactID       string
	Vector       []float32
	ModelVersion string
	CreatedAt    time.Time
}

// NewSemanticEmbedding 创建新的语义嵌入记录。
func NewSemanticEmbedding(factID string, vector []float32, modelVersion string) *SemanticEmbedding {
	if len(vector) != EmbeddingDimension {
		panic(fmt.Sprintf("embedding vector must have dimension %d, got %d", EmbeddingDimension, len(vector)))
	}
	now := time.Now().UTC()
	return &SemanticEmbedding{
		EmbeddingID:  fmt.Sprintf("emb_%d", now.UnixNano()),
		FactID:       factID,
		Vector:       vector,
		ModelVersion: modelVersion,
		CreatedAt:    now,
	}
}

// VectorToBytes 将 float32 向量序列化为 little-endian 字节数组。
func (e *SemanticEmbedding) VectorToBytes() []byte {
	buf := new(bytes.Buffer)
	for _, v := range e.Vector {
		_ = binary.Write(buf, binary.LittleEndian, v)
	}
	return buf.Bytes()
}

// VectorFromBytes 将 little-endian 字节数组反序列化为 float32 向量。
func VectorFromBytes(data []byte) ([]float32, error) {
	if len(data)%4 != 0 {
		return nil, ErrInvalidVector
	}
	count := len(data) / 4
	vector := make([]float32, count)
	buf := bytes.NewReader(data)
	for i := 0; i < count; i++ {
		if err := binary.Read(buf, binary.LittleEndian, &vector[i]); err != nil {
			return nil, fmt.Errorf("failed to decode vector: %w", err)
		}
	}
	return vector, nil
}
