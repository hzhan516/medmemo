package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntentResolver_Resolve(t *testing.T) {
	t.Parallel()
	resolver := NewIntentResolver(NewQueryExpansionService())

	tests := []struct {
		name             string
		query            string
		expectIntent     MemoryIntent
		expectConfidence IntentConfidence
	}{
		// High confidence cases
		{"weight_now", "我现在多重", IntentPersonalWeight, ConfidenceHigh},
		{"weight_jin", "我多少斤", IntentPersonalWeight, ConfidenceHigh},
		{"weight_kg", "我多少公斤", IntentPersonalWeight, ConfidenceHigh},
		{"weight_full", "我的体重是多少", IntentPersonalWeight, ConfidenceHigh},
		{"height_now", "我多高", IntentPersonalHeight, ConfidenceHigh},
		{"height_full", "我的身高是多少", IntentPersonalHeight, ConfidenceHigh},
		{"age_now", "我几岁", IntentPersonalAge, ConfidenceHigh},
		{"age_old", "我多大", IntentPersonalAge, ConfidenceHigh},
		{"age_full", "我的年龄是多少", IntentPersonalAge, ConfidenceHigh},
		{"allergy_full", "我对什么过敏", IntentAllergyHistory, ConfidenceHigh},
		{"allergy_history", "我过敏史是什么", IntentAllergyHistory, ConfidenceHigh},
		{"medication_full", "我正在吃什么药", IntentMedicationHistory, ConfidenceHigh},
		{"medication_take", "我服用什么药", IntentMedicationHistory, ConfidenceHigh},
		{"medication_what", "我吃什么药", IntentMedicationHistory, ConfidenceHigh},

		// Medium confidence cases
		{"weight_medium", "多重", IntentPersonalWeight, ConfidenceMedium},
		{"weight_keyword", "体重", IntentPersonalWeight, ConfidenceMedium},
		{"height_medium", "多高", IntentPersonalHeight, ConfidenceMedium},
		{"age_medium", "几岁", IntentPersonalAge, ConfidenceMedium},
		{"allergy_medium", "过敏史", IntentAllergyHistory, ConfidenceMedium},
		{"medication_medium", "用药", IntentMedicationHistory, ConfidenceMedium},

		// Low / blocked cases
		{"allergy_how", "过敏怎么办", IntentAllergyHistory, ConfidenceLow},
		{"medication_should", "我该吃什么药", IntentMedicationHistory, ConfidenceLow},
		{"medication_chest", "胸痛应该吃什么药", IntentMedicationHistory, ConfidenceLow},
		{"weight_how", "体重怎么办", IntentPersonalWeight, ConfidenceLow},

		// No match
		{"weather", "今天天气不错", "", 0},
		{"empty", "", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.Resolve(tt.query)
			if tt.expectConfidence == 0 {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Equal(t, tt.expectIntent, result.Intent)
			assert.Equal(t, tt.expectConfidence, result.Confidence)
		})
	}
}

func TestIntentResolver_Resolve_BlockedSuffixes(t *testing.T) {
	t.Parallel()
	resolver := NewIntentResolver(NewQueryExpansionService())

	// 阻断后缀应阻止 High/Medium 升级为本地短路
	blockedCases := []string{
		"过敏怎么办",
		"我该吃什么药",
		"胸痛应该吃什么药",
		"体重怎么减",
		"身高正常吗",
	}

	for _, query := range blockedCases {
		t.Run(query, func(t *testing.T) {
			result := resolver.Resolve(query)
			require.NotNil(t, result)
			assert.Equal(t, ConfidenceLow, result.Confidence, "含阻断后缀的 query 不应允许本地短路")
		})
	}
}

func TestIntentResolver_Resolve_AmbiguousNotShortCircuit(t *testing.T) {
	t.Parallel()
	resolver := NewIntentResolver(NewQueryExpansionService())

	// 这些 query 不应触发 High 本地短路
	ambiguousCases := []struct {
		query string
		desc  string
	}{
		{"我妈妈多重", "涉及第三方主体"},
		{"多重", "缺少主语代词"},
		{"吃什么药", "缺少主语代词"},
		{"过敏", "过于简短，缺少明确查询意图"},
	}

	for _, tc := range ambiguousCases {
		t.Run(tc.desc, func(t *testing.T) {
			result := resolver.Resolve(tc.query)
			if result != nil {
				assert.NotEqual(t, ConfidenceHigh, result.Confidence, "%s 不应触发 High 本地短路", tc.query)
			}
		})
	}
}
