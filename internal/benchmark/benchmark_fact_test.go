//go:build benchmark

package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/application/usecase"
)

// mockLLMForBenchmark 用于 benchmark 的 mock LLM，固定延迟模拟。
type mockLLMForBenchmark struct {
	latency time.Duration
}

func (m *mockLLMForBenchmark) Chat(ctx context.Context, messages []string) (string, error) {
	if m.latency > 0 {
		time.Sleep(m.latency)
	}
	return `[{"subject":"用户","predicate":"患有","object":"头痛","confidence":0.9}]`, nil
}

// BenchmarkFactExtractionRate 验证事实提取处理速率。
// DoD 要求：>= 5条对话/分钟
func BenchmarkFactExtractionRate(b *testing.B) {
	// 模拟每条对话 LLM 调用延迟 2 秒（保守估计）
	// 理论速率：60/2 = 30 条/分钟，远高于 5 条/分钟门槛
	mockLatency := 2 * time.Second
	extractor := usecase.NewFactExtractor(&mockLLMForBenchmark{latency: mockLatency})

	start := time.Now()
	for i := 0; i < b.N; i++ {
		_, err := extractor.ParseFacts(context.Background(), "用户说头疼，最近血压偏高")
		if err != nil {
			b.Fatalf("parse facts failed: %v", err)
		}
	}
	elapsed := time.Since(start)

	ratePerMinute := float64(b.N) / elapsed.Minutes()
	b.ReportMetric(ratePerMinute, "dialogs/min")

	if ratePerMinute < 5 {
		b.Fatalf("extraction rate %.1f dialogs/min < 5", ratePerMinute)
	}
}
