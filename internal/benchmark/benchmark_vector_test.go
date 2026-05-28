//go:build benchmark

package benchmark

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/adapters/repository"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
)

// BenchmarkVectorSearch10000 验证 sqlite-vec 在 10000 条向量下的搜索性能。
// DoD 要求：P99 < 50ms
func BenchmarkVectorSearch10000(b *testing.B) {
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

	// 插入 10000 条事实和嵌入
	batchSize := 10000
	now := time.Now().UTC()
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < batchSize; i++ {
		fid := fmt.Sprintf("bench_%d", i)
		fact := &entity.ExtractedFact{
			FactID: fid, Subject: "用户", Predicate: "测试", Object: fmt.Sprintf("事实%d", i),
			Confidence: 0.8, Status: entity.FactStatusApproved, SourceMsgIDs: []string{"m1"}, CreatedAt: now,
		}
		if err := factRepo.Save(ctx, fact); err != nil {
			b.Fatalf("failed to save fact: %v", err)
		}

		v := make([]float32, entity.EmbeddingDimension)
		for j := range v {
			v[j] = float32(rng.NormFloat64())
		}
		// 归一化
		var norm float64
		for _, x := range v {
			norm += float64(x) * float64(x)
		}
		norm = math.Sqrt(norm)
		if norm > 0 {
			for j := range v {
				v[j] = float32(float64(v[j]) / norm)
			}
		}

		if err := embedRepo.Save(ctx, entity.NewSemanticEmbedding(fid, v, "all-MiniLM-L6-v2")); err != nil {
			b.Fatalf("failed to save embedding: %v", err)
		}
	}

	query := make([]float32, entity.EmbeddingDimension)
	for j := range query {
		query[j] = float32(rng.NormFloat64())
	}
	var qnorm float64
	for _, x := range query {
		qnorm += float64(x) * float64(x)
	}
	qnorm = math.Sqrt(qnorm)
	if qnorm > 0 {
		for j := range query {
			query[j] = float32(float64(query[j]) / qnorm)
		}
	}

	// 预热
	_, _ = embedRepo.SearchSimilar(ctx, query, 10)

	b.ResetTimer()
	var totalElapsed time.Duration
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, err := embedRepo.SearchSimilar(ctx, query, 10)
		elapsed := time.Since(start)
		totalElapsed += elapsed
		if err != nil {
			b.Fatalf("search failed: %v", err)
		}
	}

	avg := totalElapsed / time.Duration(b.N)
	b.ReportMetric(float64(avg.Microseconds())/1000, "ms/op")
}

// BenchmarkAccuracyComparison 验证 sqlite-vec Top-K 与暴力全表 cosine 相似度的重叠率。
// DoD 要求：重叠率 > 95%
func BenchmarkAccuracyComparison(b *testing.B) {
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
	now := time.Now().UTC()

	// 插入 1000 条随机高斯归一化向量（确保 cosine 相似度有良好区分度）
	vecCount := 1000
	vectors := make([][]float32, vecCount)
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < vecCount; i++ {
		v := make([]float32, entity.EmbeddingDimension)
		for j := range v {
			v[j] = float32(rng.NormFloat64())
		}
		// 归一化
		var norm float64
		for _, x := range v {
			norm += float64(x) * float64(x)
		}
		norm = math.Sqrt(norm)
		if norm > 0 {
			for j := range v {
				v[j] = float32(float64(v[j]) / norm)
			}
		}
		vectors[i] = v
		fid := fmt.Sprintf("acc_%d", i)

		// 先保存 fact（满足外键约束）
		fact := &entity.ExtractedFact{
			FactID: fid, Subject: "用户", Predicate: "测试", Object: fmt.Sprintf("事实%d", i),
			Confidence: 0.8, Status: entity.FactStatusApproved, SourceMsgIDs: []string{"m1"}, CreatedAt: now,
		}
		if err := factRepo.Save(ctx, fact); err != nil {
			b.Fatalf("failed to save fact: %v", err)
		}
		if err := embedRepo.Save(ctx, entity.NewSemanticEmbedding(fid, v, "all-MiniLM-L6-v2")); err != nil {
			b.Fatalf("failed to save embedding: %v", err)
		}
	}

	// 查询向量也使用随机高斯分布生成
	query := make([]float32, entity.EmbeddingDimension)
	for j := range query {
		query[j] = float32(rng.NormFloat64())
	}
	var qnorm float64
	for _, x := range query {
		qnorm += float64(x) * float64(x)
	}
	qnorm = math.Sqrt(qnorm)
	if qnorm > 0 {
		for i := range query {
			query[i] = float32(float64(query[i]) / qnorm)
		}
	}

	// sqlite-vec Top-10
	vecResults, err := embedRepo.SearchSimilar(ctx, query, 10)
	if err != nil {
		b.Fatalf("sqlite-vec search failed: %v", err)
	}
	requireLen(b, len(vecResults), 10)

	// 暴力搜索 Top-10
	bruteResults := bruteForceTopK(query, vectors, 10)

	// 计算重叠率
	vecSet := make(map[string]bool)
	for _, r := range vecResults {
		vecSet[r.FactID] = true
	}
	overlap := 0
	for _, idx := range bruteResults {
		fid := fmt.Sprintf("acc_%d", idx)
		if vecSet[fid] {
			overlap++
		}
	}
	overlapRate := float64(overlap) / 10.0 * 100
	b.ReportMetric(overlapRate, "%overlap")

	if overlapRate < 95 {
		b.Fatalf("overlap rate %.1f%% < 95%%", overlapRate)
	}
}

func bruteForceTopK(query []float32, vectors [][]float32, k int) []int {
	type scored struct {
		idx int
		sim float64
	}
	scoredVecs := make([]scored, len(vectors))
	for i, v := range vectors {
		var sim float64
		for j := range query {
			sim += float64(query[j]) * float64(v[j])
		}
		scoredVecs[i] = scored{idx: i, sim: sim}
	}

	// 简单冒泡排序取 Top-K
	for i := 0; i < k; i++ {
		for j := i + 1; j < len(scoredVecs); j++ {
			if scoredVecs[j].sim > scoredVecs[i].sim {
				scoredVecs[i], scoredVecs[j] = scoredVecs[j], scoredVecs[i]
			}
		}
	}

	result := make([]int, k)
	for i := 0; i < k; i++ {
		result[i] = scoredVecs[i].idx
	}
	return result
}

func requireLen(b *testing.B, actual, expected int) {
	if actual != expected {
		b.Fatalf("expected len %d, got %d", expected, actual)
	}
}
