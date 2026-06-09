package usecase

import (
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestLocalAnswerService_Format(t *testing.T) {
	svc := NewLocalAnswerService()

	tests := []struct {
		intent   MemoryIntent
		fact     *entity.ExtractedFact
		expected string
	}{
		{
			intent:   IntentPersonalWeight,
			fact:     &entity.ExtractedFact{Object: "110公斤"},
			expected: "记录中显示，你当前体重为 110公斤。",
		},
		{
			intent:   IntentPersonalHeight,
			fact:     &entity.ExtractedFact{Object: "175厘米"},
			expected: "记录中显示，你当前身高为 175厘米。",
		},
		{
			intent:   IntentPersonalAge,
			fact:     &entity.ExtractedFact{Object: "35岁"},
			expected: "记录中显示，你当前年龄为 35岁。",
		},
		{
			intent:   IntentAllergyHistory,
			fact:     &entity.ExtractedFact{Object: "对青霉素过敏"},
			expected: "记录中显示，你的过敏相关信息为：对青霉素过敏。",
		},
		{
			intent:   IntentMedicationHistory,
			fact:     &entity.ExtractedFact{Object: "阿司匹林"},
			expected: "记录中显示，你的用药相关信息为：阿司匹林。",
		},
		{
			intent:   MemoryIntent("unknown"),
			fact:     &entity.ExtractedFact{Object: "未知"},
			expected: "",
		},
		{
			intent:   IntentPersonalWeight,
			fact:     nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.intent), func(t *testing.T) {
			got := svc.Format(tt.intent, tt.fact)
			assert.Equal(t, tt.expected, got)
		})
	}
}
