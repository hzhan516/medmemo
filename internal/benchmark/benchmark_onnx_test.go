//go:build benchmark

package benchmark

import (
	"context"
	"os"
	"runtime"
	"testing"

	"github.com/hzhan516/medmemo/internal/infrastructure/onnx"
)

// modelPath 返回 ONNX 模型路径，不存在时跳过测试。
func modelPath() string {
	resourceDir := "." // 使用当前目录作为资源根
	path := onnx.DefaultModelPath(resourceDir)
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}

// BenchmarkEmbedSingle 验证单条文本嵌入推理延迟。
// DoD 要求：P99 < 100ms（CPU，中等负载）
func BenchmarkEmbedSingle(b *testing.B) {
	model := modelPath()
	if model == "" {
		b.Skip("ONNX model not found, skipping benchmark")
	}

	engine, err := onnx.NewEngine(onnx.EngineConfig{ResourceDir: ".", ModelPath: model})
	if err != nil {
		b.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()
	// 预热
	_, _ = engine.Embed(ctx, []string{"预热文本"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Embed(ctx, []string{"这是一个测试句子，用于测量嵌入推理延迟。"})
		if err != nil {
			b.Fatalf("embed failed: %v", err)
		}
	}
}

// BenchmarkONNXMemoryLeak 验证 10000 次推理后内存增长 < 10MB。
func BenchmarkONNXMemoryLeak(b *testing.B) {
	model := modelPath()
	if model == "" {
		b.Skip("ONNX model not found, skipping benchmark")
	}

	engine, err := onnx.NewEngine(onnx.EngineConfig{ResourceDir: ".", ModelPath: model})
	if err != nil {
		b.Fatalf("failed to create engine: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()
	// 强制 GC 以获取基线
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// 执行 10000 次推理
	for i := 0; i < 10000; i++ {
		_, err := engine.Embed(ctx, []string{"内存泄漏测试"})
		if err != nil {
			b.Fatalf("embed failed: %v", err)
		}
	}

	// 强制 GC 后测量
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	growth := int64(m2.Alloc) - int64(m1.Alloc)
	growthMB := float64(growth) / (1024 * 1024)
	b.ReportMetric(growthMB, "MB_growth")

	if growthMB > 10 {
		b.Fatalf("memory growth %.2fMB > 10MB after 10000 inferences", growthMB)
	}
}
