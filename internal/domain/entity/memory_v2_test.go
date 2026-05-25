package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRawDialogue_New(t *testing.T) {
	d := NewRawDialogue("session_001", RoleUser, "用户头疼", "kimi-v1")

	require.NotEmpty(t, d.MessageID)
	assert.Equal(t, "session_001", d.SessionID)
	assert.Equal(t, RoleUser, d.Role)
	assert.Equal(t, "用户头疼", d.Content)
	assert.Equal(t, "kimi-v1", d.ModelName)
	assert.Equal(t, ExtractionStatusUnprocessed, d.ExtractionStatus)
	assert.WithinDuration(t, time.Now().UTC(), d.Timestamp, time.Second)
	assert.WithinDuration(t, time.Now().UTC(), d.CreatedAt, time.Second)
}

func TestRawDialogue_New_AssistantRole(t *testing.T) {
	d := NewRawDialogue("session_002", RoleAssistant, "建议就医", "gpt-4o")
	assert.Equal(t, RoleAssistant, d.Role)
}

func TestRawDialogue_New_SystemRole(t *testing.T) {
	d := NewRawDialogue("session_003", RoleSystem, "系统提示", "")
	assert.Equal(t, RoleSystem, d.Role)
}

func TestExtractedFact_New(t *testing.T) {
	f := NewExtractedFact("用户", "患有", "偏头痛", 0.85, []string{"msg_001"})

	require.NotEmpty(t, f.FactID)
	assert.Equal(t, "用户", f.Subject)
	assert.Equal(t, "患有", f.Predicate)
	assert.Equal(t, "偏头痛", f.Object)
	assert.Equal(t, 0.85, f.Confidence)
	assert.Equal(t, []string{"msg_001"}, f.SourceMsgIDs)
	assert.Equal(t, FactStatusPending, f.Status)
	assert.Nil(t, f.ScoredAt)
	assert.Nil(t, f.ReviewedAt)
	assert.WithinDuration(t, time.Now().UTC(), f.CreatedAt, time.Second)
}

func TestExtractedFact_Validate_Complete(t *testing.T) {
	f := NewExtractedFact("用户", "服用", "阿司匹林", 0.75, []string{"msg_001", "msg_002"})
	err := f.Validate()
	assert.NoError(t, err)
}

func TestExtractedFact_Validate_EmptySubject(t *testing.T) {
	f := NewExtractedFact("", "患有", "头痛", 0.8, []string{"msg_001"})
	err := f.Validate()
	assert.ErrorIs(t, err, ErrInvalidFact)
}

func TestExtractedFact_Validate_EmptyPredicate(t *testing.T) {
	f := NewExtractedFact("用户", "", "头痛", 0.8, []string{"msg_001"})
	err := f.Validate()
	assert.ErrorIs(t, err, ErrInvalidFact)
}

func TestExtractedFact_Validate_EmptyObject(t *testing.T) {
	f := NewExtractedFact("用户", "患有", "", 0.8, []string{"msg_001"})
	err := f.Validate()
	assert.ErrorIs(t, err, ErrInvalidFact)
}

func TestExtractedFact_Validate_NegativeConfidence(t *testing.T) {
	f := NewExtractedFact("用户", "患有", "头痛", -0.1, []string{"msg_001"})
	err := f.Validate()
	assert.ErrorIs(t, err, ErrInvalidConfidence)
}

func TestExtractedFact_Validate_ConfidenceAboveOne(t *testing.T) {
	f := NewExtractedFact("用户", "患有", "头痛", 1.5, []string{"msg_001"})
	err := f.Validate()
	assert.ErrorIs(t, err, ErrInvalidConfidence)
}

func TestExtractedFact_Validate_EmptySourceMsgs(t *testing.T) {
	f := NewExtractedFact("用户", "患有", "头痛", 0.8, []string{})
	err := f.Validate()
	assert.ErrorIs(t, err, ErrInvalidFact)
}

func TestExtractedFact_SetStatus_Approved(t *testing.T) {
	f := NewExtractedFact("用户", "患有", "头痛", 0.9, []string{"msg_001"})
	f.SetStatus(FactStatusApproved)
	assert.Equal(t, FactStatusApproved, f.Status)
	assert.NotNil(t, f.ReviewedAt)
}

func TestExtractedFact_SetStatus_Rejected(t *testing.T) {
	f := NewExtractedFact("用户", "患有", "头痛", 0.3, []string{"msg_001"})
	f.SetStatus(FactStatusRejected)
	assert.Equal(t, FactStatusRejected, f.Status)
	assert.NotNil(t, f.ReviewedAt)
}

func TestExtractedFact_SetScoredAt(t *testing.T) {
	f := NewExtractedFact("用户", "患有", "头痛", 0.8, []string{"msg_001"})
	f.SetScoredAt()
	assert.NotNil(t, f.ScoredAt)
}

func TestSemanticEmbedding_New(t *testing.T) {
	vector := make([]float32, 384)
	for i := range vector {
		vector[i] = float32(i) / 384.0
	}

	e := NewSemanticEmbedding("fact_001", vector, "all-MiniLM-L6-v2")

	require.NotEmpty(t, e.EmbeddingID)
	assert.Equal(t, "fact_001", e.FactID)
	assert.Equal(t, 384, len(e.Vector))
	assert.Equal(t, "all-MiniLM-L6-v2", e.ModelVersion)
	assert.WithinDuration(t, time.Now().UTC(), e.CreatedAt, time.Second)
}

func TestSemanticEmbedding_New_WrongDimension(t *testing.T) {
	vector := make([]float32, 100)

	assert.Panics(t, func() {
		NewSemanticEmbedding("fact_001", vector, "all-MiniLM-L6-v2")
	})
}

func TestSemanticEmbedding_New_ZeroDimension(t *testing.T) {
	assert.Panics(t, func() {
		NewSemanticEmbedding("fact_001", []float32{}, "all-MiniLM-L6-v2")
	})
}

func TestSemanticEmbedding_VectorToBytes(t *testing.T) {
	vector := []float32{1.0, 2.0, 3.0}
	// 使用测试构造器绕过维度检查
	e := &SemanticEmbedding{Vector: vector}
	b := e.VectorToBytes()
	assert.Equal(t, 12, len(b)) // 3 * 4 bytes
}

func TestSemanticEmbedding_VectorFromBytes(t *testing.T) {
	vector := []float32{1.0, 2.0, 3.0}
	e := &SemanticEmbedding{Vector: vector}
	b := e.VectorToBytes()

	restored, err := VectorFromBytes(b)
	require.NoError(t, err)
	assert.Equal(t, vector, restored)
}

func TestSemanticEmbedding_VectorFromBytes_InvalidLength(t *testing.T) {
	_, err := VectorFromBytes([]byte{0x01, 0x02})
	assert.ErrorIs(t, err, ErrInvalidVector)
}

func TestSemanticEmbedding_RoundTrip(t *testing.T) {
	original := make([]float32, 384)
	for i := range original {
		original[i] = float32(i) * 0.001
	}

	e := &SemanticEmbedding{Vector: original}
	b := e.VectorToBytes()
	restored, err := VectorFromBytes(b)
	require.NoError(t, err)
	assert.Equal(t, original, restored)
}
