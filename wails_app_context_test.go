package main

import (
	"testing"

	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
)

// TestResolveStrategy 验证策略字符串解析与非法值回退。
func TestResolveStrategy(t *testing.T) {
	cases := []struct {
		input string
		want  usecase.CompressionStrategyKind
	}{
		{"summarize_and_replace", usecase.StrategySummarizeAndReplace},
		{"drop_earliest_n", usecase.StrategyDropEarliestN},
		{"llm_self_summarization", usecase.StrategyLLMSelfSummarize},
		{"", usecase.StrategySummarizeAndReplace},
		{"invalid", usecase.StrategySummarizeAndReplace},
	}

	for _, tt := range cases {
		got := resolveStrategy(tt.input)
		assert.Equal(t, tt.want, got, "input=%q", tt.input)
	}
}

// TestBuildCompressionConfigFrom 验证压缩配置与 provider/model 回退逻辑。
func TestBuildCompressionConfigFrom(t *testing.T) {
	t.Run("model disabled uses active provider", func(t *testing.T) {
		cfg, pid, mid := buildCompressionConfigFrom(models.CompressionSettings{
			UseModel:    false,
			AnchorCount: 2,
			RecentCount: 8,
		}, "active-p", "active-m")
		assert.Equal(t, usecase.StrategySummarizeAndReplace, cfg.Strategy)
		assert.Equal(t, 2, cfg.AnchorCount)
		assert.Equal(t, 8, cfg.RecentCount)
		assert.Equal(t, "active-p", pid)
		assert.Equal(t, "active-m", mid)
	})

	t.Run("model enabled with explicit provider", func(t *testing.T) {
		cfg, pid, mid := buildCompressionConfigFrom(models.CompressionSettings{
			UseModel:    true,
			ProviderID:  "custom-p",
			ModelID:     "custom-m",
			AnchorCount: 3,
			RecentCount: 10,
		}, "active-p", "active-m")
		assert.Equal(t, usecase.StrategyLLMSelfSummarize, cfg.Strategy)
		assert.Equal(t, 3, cfg.AnchorCount)
		assert.Equal(t, 10, cfg.RecentCount)
		assert.Equal(t, "custom-p", pid)
		assert.Equal(t, "custom-m", mid)
	})

	t.Run("model enabled falls back to active when fields empty", func(t *testing.T) {
		cfg, pid, mid := buildCompressionConfigFrom(models.CompressionSettings{
			UseModel: true,
		}, "active-p", "active-m")
		assert.Equal(t, usecase.StrategyLLMSelfSummarize, cfg.Strategy)
		assert.Equal(t, "active-p", pid)
		assert.Equal(t, "active-m", mid)
	})

	t.Run("default anchor and recent when zero", func(t *testing.T) {
		cfg, _, _ := buildCompressionConfigFrom(models.CompressionSettings{
			UseModel: false,
		}, "p", "m")
		assert.Equal(t, 1, cfg.AnchorCount)
		assert.Equal(t, 6, cfg.RecentCount)
	})
}
