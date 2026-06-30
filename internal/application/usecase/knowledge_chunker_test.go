package usecase

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnowledgeChunker_ChunkMarkdown(t *testing.T) {
	chunker := NewKnowledgeChunker(200, 20)
	content := []byte(`# 健康指南

## 感冒
感冒通常由病毒引起。

## 发热
发热是身体免疫反应。`)

	chunks := chunker.ChunkMarkdown("guide.md", content)
	require.NotEmpty(t, chunks)
	assert.Equal(t, 0, chunks[0].ChunkIndex)
	found := false
	for _, c := range chunks {
		if strings.Contains(c.Content, "感冒") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected at least one chunk to contain 感冒")
}

func TestKnowledgeChunker_ChunkJSONL(t *testing.T) {
	chunker := NewKnowledgeChunker(200, 20)
	content := []byte(`{"title": "感冒", "text": "感冒通常由病毒引起。"}
{"title": "发热", "content": "发热是身体免疫反应。"}
`)

	chunks, err := chunker.ChunkJSONL(content)
	require.NoError(t, err)
	require.Len(t, chunks, 2)
	assert.Equal(t, "感冒通常由病毒引起。", chunks[0].Content)
	assert.Equal(t, "发热是身体免疫反应。", chunks[1].Content)
}

func TestKnowledgeChunker_ChunkJSONL_MissingText(t *testing.T) {
	chunker := NewKnowledgeChunker(200, 20)
	content := []byte(`{"id": "1"}`)

	_, err := chunker.ChunkJSONL(content)
	require.Error(t, err)
}
