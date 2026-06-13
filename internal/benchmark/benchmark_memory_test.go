//go:build benchmark

package benchmark

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/adapters/repository"
	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
)

// mockEmbeddingService 用于 benchmark 的 mock 嵌入服务。
type mockEmbeddingService struct{}

func (m *mockEmbeddingService) IsAvailable() bool {
	return true
}

func (m *mockEmbeddingService) ModelVersion() string {
	return "mock-benchmark"
}

func (m *mockEmbeddingService) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		v := make([]float32, entity.EmbeddingDimension)
		v[0] = 1.0
		result[i] = v
	}
	return result, nil
}

func (m *mockEmbeddingService) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	v := make([]float32, entity.EmbeddingDimension)
	v[0] = 1.0
	return v, nil
}

// BenchmarkRetrieveForContext 验证完整记忆召回流程的性能。
// DoD 要求：P99 < 200ms
func BenchmarkRetrieveForContext(b *testing.B) {
	tmpDir := b.TempDir()
	connector, err := database.NewSQLiteConnector(tmpDir)
	if err != nil {
		b.Fatalf("failed to create connector: %v", err)
	}
	defer connector.Close()

	ctx := context.Background()
	if err := connector.Migrate(ctx); err != nil {
		b.Fatalf("failed to migrate: %v", err)
	}

	factRepo := repository.NewFactRepoSQLite(connector)
	embedRepo := repository.NewEmbeddingRepoSQLite(connector)

	// 准备 100 条 approved 事实和嵌入
	now := time.Now().UTC()
	for i := 0; i < 100; i++ {
		fid := fmt.Sprintf("bench_mem_%d", i)
		fact := &entity.ExtractedFact{
			FactID: fid, Subject: "用户", Predicate: "测试", Object: fmt.Sprintf("事实%d", i),
			Confidence: 0.8, Status: entity.FactStatusApproved, SourceMsgIDs: []string{"m1"}, CreatedAt: now,
		}
		if err := factRepo.Save(ctx, fact); err != nil {
			b.Fatalf("failed to save fact: %v", err)
		}

		v := make([]float32, entity.EmbeddingDimension)
		v[0] = float32(i+1) * 0.01
		if err := embedRepo.Save(ctx, entity.NewSemanticEmbedding(fid, v, "all-MiniLM-L6-v2")); err != nil {
			b.Fatalf("failed to save embedding: %v", err)
		}
	}

	mockSvc := &mockEmbeddingService{}
	retriever := usecase.NewMemoryRetriever(mockSvc, embedRepo, factRepo, usecase.NewDecayScorer(), nil, nil, nil)

	// 预热
	_, _ = retriever.RetrieveForContext(ctx, "测试查询", "sess_1", 3)

	b.ResetTimer()
	var totalElapsed time.Duration
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, err := retriever.RetrieveForContext(ctx, "测试查询", fmt.Sprintf("sess_%d", i), 3)
		elapsed := time.Since(start)
		totalElapsed += elapsed
		if err != nil {
			b.Fatalf("retrieve failed: %v", err)
		}
	}

	avg := totalElapsed / time.Duration(b.N)
	b.ReportMetric(float64(avg.Microseconds())/1000, "ms/op")
}
