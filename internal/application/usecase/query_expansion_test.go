package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryExpansionService_Normalize(t *testing.T) {
	svc := NewQueryExpansionService()

	tests := []struct {
		input    string
		expected string
	}{
		{"  我现在多重？  ", "我现在多重"},
		{"我多少斤？", "我多少斤"},
		{"我体重是多少！", "我体重是多少"},
		{"　我　现在　多重　", "我 现在 多重"},
		{"我现在多重，？。！", "我现在多重"},
		{"", ""},
		{"  ", ""},
		{"我现在多重？", "我现在多重"},
		{"我１１０公斤", "我110公斤"},
		{"我体重是多少？，！", "我体重是多少"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := svc.Normalize(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
