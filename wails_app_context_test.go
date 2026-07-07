package main

import (
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestContextUsageCacheKey 验证缓存 key 对 provider/model/消息内容敏感。
func TestContextUsageCacheKey(t *testing.T) {
	msgs := []models.Message{
		{Role: models.RoleUser, Content: "hello"},
		{Role: models.RoleAssistant, Content: "hi"},
	}

	base := contextUsageCacheKey("p1", "m1", msgs)

	// 相同输入 => 相同 key
	assert.Equal(t, base, contextUsageCacheKey("p1", "m1", msgs))

	// provider / model / 内容任一变化 => key 变化
	assert.NotEqual(t, base, contextUsageCacheKey("p2", "m1", msgs))
	assert.NotEqual(t, base, contextUsageCacheKey("p1", "m2", msgs))
	assert.NotEqual(t, base, contextUsageCacheKey("p1", "m1", []models.Message{
		{Role: models.RoleUser, Content: "hello!"},
	}))
}

// TestContextUsageCache_SetGet 验证缓存写入后可读回，未命中返回 false。
func TestContextUsageCache_SetGet(t *testing.T) {
	key := contextUsageCacheKey("set-get-provider", "set-get-model", []models.Message{
		{Role: models.RoleUser, Content: "unique-content-for-cache-test"},
	})

	if _, ok := contextUsageCacheGet(key); ok {
		t.Fatalf("预期缓存初始未命中")
	}

	want := ContextUsageResponse{UsedTokens: 42, MaxTokens: 8192, Ratio: 0.5, Approximate: true}
	contextUsageCacheSet(key, want)

	got, ok := contextUsageCacheGet(key)
	require.True(t, ok)
	assert.Equal(t, want, got)
}

// TestContextUsageCache_Expired 验证过期条目在读取时被清除并返回未命中。
func TestContextUsageCache_Expired(t *testing.T) {
	key := contextUsageCacheKey("expired-provider", "expired-model", []models.Message{
		{Role: models.RoleUser, Content: "expiry-content"},
	})

	contextUsageCacheMu.Lock()
	contextUsageCache[key] = contextUsageCacheEntry{
		resp:      ContextUsageResponse{UsedTokens: 7},
		expiresAt: time.Now().Add(-time.Second), // 已过期
	}
	contextUsageCacheMu.Unlock()

	_, ok := contextUsageCacheGet(key)
	assert.False(t, ok, "过期条目应视为未命中")

	// 过期项应在读取时被删除
	contextUsageCacheMu.Lock()
	_, exists := contextUsageCache[key]
	contextUsageCacheMu.Unlock()
	assert.False(t, exists, "过期条目应被清除")
}
