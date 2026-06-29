package usecase

import (
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/stretchr/testify/assert"
)

func TestRRFMerge(t *testing.T) {
	kw := []*repository.KnowledgeSearchResult{
		{ChunkID: "c1", Score: 1.0},
		{ChunkID: "c2", Score: 0.8},
	}
	vec := []*repository.KnowledgeSearchResult{
		{ChunkID: "c2", Score: 1.0},
		{ChunkID: "c3", Score: 0.9},
	}

	merged := RRFMerge([][]*repository.KnowledgeSearchResult{kw, vec}, []float64{1.0, 1.0}, 60, 10)
	assert.Len(t, merged, 3)
	assert.Equal(t, "c2", merged[0].ChunkID) // 同时出现在两路中，RRF 分最高
}

func TestRRFMerge_EmptyList(t *testing.T) {
	merged := RRFMerge([][]*repository.KnowledgeSearchResult{{}}, []float64{1.0}, 60, 10)
	assert.Empty(t, merged)
}
