package usecase

import (
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/entity"
)

func BenchmarkConfidenceAggregator_Calculate(b *testing.B) {
	agg := NewConfidenceAggregator()
	sources := []entity.KnowledgeSource{
		{Type: entity.SourceMedicalGuideline, Confidence: 0.95, Citation: "中华医学会消化指南2023"},
		{Type: entity.SourceEvidenceDB, Confidence: 0.85, Citation: "PubMedQA"},
	}
	reasoning := entity.ReasoningChain{
		HasSymptomAnalysis: true,
		HasDifferentialDx:  true,
		HasRecommendation:  true,
		HasUncertaintyAck:  true,
		HasEmergencyCheck:  true,
		MissingInfoList:    []string{"疼痛持续时间"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = agg.Calculate(sources, reasoning, 80.0, 0.75, 85.0)
	}
}

func BenchmarkConfidenceAggregator_CalculateParallel(b *testing.B) {
	agg := NewConfidenceAggregator()
	sources := []entity.KnowledgeSource{
		{Type: entity.SourceMedicalGuideline, Confidence: 0.95, Citation: "指南"},
	}
	reasoning := entity.ReasoningChain{
		HasSymptomAnalysis: true,
		HasDifferentialDx:  true,
		HasRecommendation:  true,
		HasUncertaintyAck:  true,
		HasEmergencyCheck:  true,
		MissingInfoList:    []string{},
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = agg.Calculate(sources, reasoning, 80.0, 0.75, 85.0)
		}
	})
}
