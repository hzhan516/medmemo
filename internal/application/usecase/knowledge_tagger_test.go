package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/hzhan516/medmemo/internal/domain/entity"
)

func TestKnowledgeSourceTagger_Tag_MedicalGuideline(t *testing.T) {
	tagger := NewKnowledgeSourceTagger()
	ks := tagger.Tag(entity.SourceMedicalGuideline, "中华医学会消化指南2023")

	assert.Equal(t, entity.SourceMedicalGuideline, ks.Type)
	assert.InDelta(t, 0.95, ks.Confidence, 0.001)
	assert.Equal(t, "中华医学会消化指南2023", ks.Citation)
}

func TestKnowledgeSourceTagger_Tag_EvidenceDB(t *testing.T) {
	tagger := NewKnowledgeSourceTagger()
	ks := tagger.Tag(entity.SourceEvidenceDB, "PubMedQA")

	assert.Equal(t, entity.SourceEvidenceDB, ks.Type)
	assert.InDelta(t, 0.85, ks.Confidence, 0.001)
}

func TestKnowledgeSourceTagger_Tag_KnowledgeGraph(t *testing.T) {
	tagger := NewKnowledgeSourceTagger()
	ks := tagger.Tag(entity.SourceKnowledgeGraph, "CMeKG")

	assert.Equal(t, entity.SourceKnowledgeGraph, ks.Type)
	assert.InDelta(t, 0.75, ks.Confidence, 0.001)
}

func TestKnowledgeSourceTagger_Tag_LLMInternal(t *testing.T) {
	tagger := NewKnowledgeSourceTagger()
	ks := tagger.Tag(entity.SourceLLMInternal, "")

	assert.Equal(t, entity.SourceLLMInternal, ks.Type)
	assert.InDelta(t, 0.60, ks.Confidence, 0.001)
}

func TestKnowledgeSourceTagger_Tag_UnknownDefaultsToLLMInternal(t *testing.T) {
	tagger := NewKnowledgeSourceTagger()
	// 未知来源类型应降级为 llm_internal
	ks := tagger.Tag(entity.SourceType("unknown_source"), "some citation")

	assert.Equal(t, entity.SourceLLMInternal, ks.Type)
	assert.InDelta(t, 0.60, ks.Confidence, 0.001)
}

func TestKnowledgeSourceTagger_CalculateSourceScore(t *testing.T) {
	tagger := NewKnowledgeSourceTagger()

	// 单一医学指南来源
	sources := []entity.KnowledgeSource{
		{Type: entity.SourceMedicalGuideline, Confidence: 0.95, Citation: "指南1"},
	}
	score := tagger.CalculateSourceScore(sources)
	assert.InDelta(t, 0.95, score, 0.001)

	// 混合来源：取平均值
	sources = []entity.KnowledgeSource{
		{Type: entity.SourceMedicalGuideline, Confidence: 0.95, Citation: "指南1"},
		{Type: entity.SourceEvidenceDB, Confidence: 0.85, Citation: "PubMed"},
	}
	score = tagger.CalculateSourceScore(sources)
	assert.InDelta(t, 0.90, score, 0.001)

	// 空来源列表：降级为 llm_internal 基准值
	sources = []entity.KnowledgeSource{}
	score = tagger.CalculateSourceScore(sources)
	assert.InDelta(t, 0.60, score, 0.001)
}

func TestKnowledgeSourceTagger_Tag_WithCitation(t *testing.T) {
	tagger := NewKnowledgeSourceTagger()
	ks := tagger.Tag(entity.SourceMedicalGuideline, "Huatuo-26M")

	assert.Equal(t, "Huatuo-26M", ks.Citation)
}

func TestKnowledgeSourceTagger_Tag_EmptyCitation(t *testing.T) {
	tagger := NewKnowledgeSourceTagger()
	ks := tagger.Tag(entity.SourceLLMInternal, "")

	assert.Equal(t, "", ks.Citation)
	assert.Equal(t, entity.SourceLLMInternal, ks.Type)
}
