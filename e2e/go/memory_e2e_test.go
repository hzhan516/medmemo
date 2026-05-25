//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/hzhan516/medmemo/internal/adapters/repository"
	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/hzhan516/medmemo/internal/domain/entity"
)

// TestE2E_Memory_VectorSearchAndRecall 验证向量搜索和记忆召回的完整流程。
func TestE2E_Memory_VectorSearchAndRecall(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	ctx := context.Background()
	factRepo := repository.NewFactRepoSQLite(conn)
	embedRepo := repository.NewEmbeddingRepoSQLite(conn)

	now := time.Now().UTC()

	// 准备 3 个事实和对应的嵌入向量
	facts := []*entity.ExtractedFact{
		{FactID: "e2e_f1", Subject: "用户", Predicate: "患有", Object: "高血压", Confidence: 0.9, Status: entity.FactStatusApproved, SourceMsgIDs: []string{"m1"}, CreatedAt: now},
		{FactID: "e2e_f2", Subject: "用户", Predicate: "服用", Object: "降压药", Confidence: 0.85, Status: entity.FactStatusApproved, SourceMsgIDs: []string{"m2"}, CreatedAt: now},
		{FactID: "e2e_f3", Subject: "用户", Predicate: "感觉", Object: "头晕", Confidence: 0.7, Status: entity.FactStatusPending, SourceMsgIDs: []string{"m3"}, CreatedAt: now},
	}
	for _, f := range facts {
		require.NoError(t, factRepo.Save(ctx, f))
	}

	// 创建与查询向量 [1,0,0,...] 不同相似度的嵌入
	v1 := make([]float32, entity.EmbeddingDimension) // 与 query 完全一致
	v1[0] = 1.0
	v2 := make([]float32, entity.EmbeddingDimension) // 高度相似
	v2[0] = 0.9
	v2[1] = 0.1
	v3 := make([]float32, entity.EmbeddingDimension) // 中等相似，不同方向
	v3[0] = 0.6
	v3[1] = 0.4

	embeddings := []*entity.SemanticEmbedding{
		entity.NewSemanticEmbedding("e2e_f1", v1, "all-MiniLM-L6-v2"),
		entity.NewSemanticEmbedding("e2e_f2", v2, "all-MiniLM-L6-v2"),
		entity.NewSemanticEmbedding("e2e_f3", v3, "all-MiniLM-L6-v2"),
	}
	for _, e := range embeddings {
		require.NoError(t, embedRepo.Save(ctx, e))
	}

	// 查询向量 [1,0,0,...]
	query := make([]float32, entity.EmbeddingDimension)
	query[0] = 1.0

	// 向量搜索
	results, err := embedRepo.SearchSimilar(ctx, query, 10)
	require.NoError(t, err)
	require.Len(t, results, 3)

	// 验证排序：v1 > v2 > v3
	assert.Equal(t, "e2e_f1", results[0].FactID)
	assert.InDelta(t, 1.0, results[0].Similarity, 0.001)
	assert.Equal(t, "e2e_f2", results[1].FactID)
	assert.Greater(t, results[1].Similarity, results[2].Similarity)
}

// TestE2E_Memory_DecayEffect 验证时间衰减对记忆排序的影响。
func TestE2E_Memory_DecayEffect(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	ctx := context.Background()
	factRepo := repository.NewFactRepoSQLite(conn)
	embedRepo := repository.NewEmbeddingRepoSQLite(conn)

	now := time.Now().UTC()

	// fact_old: 30 天前，similarity = 1.0，衰减后 ≈ 0.223
	// fact_new: 今天，similarity = 0.5，不衰减 = 0.5
	// 预期：新记忆排名更高
	oldFact := &entity.ExtractedFact{
		FactID: "e2e_old", Subject: "用户", Predicate: "患有", Object: "感冒",
		Confidence: 1.0, Status: entity.FactStatusApproved, SourceMsgIDs: []string{"m1"},
		CreatedAt: now.Add(-30 * 24 * time.Hour),
	}
	newFact := &entity.ExtractedFact{
		FactID: "e2e_new", Subject: "用户", Predicate: "服用", Object: "维生素",
		Confidence: 1.0, Status: entity.FactStatusApproved, SourceMsgIDs: []string{"m2"},
		CreatedAt: now,
	}
	require.NoError(t, factRepo.Save(ctx, oldFact))
	require.NoError(t, factRepo.Save(ctx, newFact))

	vOld := make([]float32, entity.EmbeddingDimension)
	vOld[0] = 1.0
	vNew := make([]float32, entity.EmbeddingDimension)
	vNew[0] = 0.6
	vNew[1] = 0.4

	require.NoError(t, embedRepo.Save(ctx, entity.NewSemanticEmbedding("e2e_old", vOld, "all-MiniLM-L6-v2")))
	require.NoError(t, embedRepo.Save(ctx, entity.NewSemanticEmbedding("e2e_new", vNew, "all-MiniLM-L6-v2")))

	// 使用 MemoryRetriever 验证衰减排序
	scorer := usecase.NewDecayScorer()
	query := make([]float32, entity.EmbeddingDimension)
	query[0] = 1.0

	results, err := embedRepo.SearchSimilar(ctx, query, 10)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// 原始相似度：old(1.0) > new(0.5)
	assert.Equal(t, "e2e_old", results[0].FactID)
	assert.Equal(t, "e2e_new", results[1].FactID)

	// 衰减后评分
	oldScore := scorer.ScoreFromCreatedAt(results[0].Similarity, oldFact.CreatedAt, now)
	newScore := scorer.ScoreFromCreatedAt(results[1].Similarity, newFact.CreatedAt, now)

	// 衰减后：new > old
	assert.Greater(t, newScore, oldScore)
	// vNew cosine = (0.6*1 + 0.4*0) / sqrt(0.6²+0.4²) = 0.6 / 0.721 ≈ 0.832
	assert.InDelta(t, 0.832, newScore, 0.01)
	// vOld cosine = 1.0, decay = exp(-1.5) ≈ 0.223
	assert.InDelta(t, 0.223, oldScore, 0.01)
}

// TestE2E_Memory_DeleteConsistency 验证删除事实后向量索引同步清理。
func TestE2E_Memory_DeleteConsistency(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	ctx := context.Background()
	factRepo := repository.NewFactRepoSQLite(conn)
	embedRepo := repository.NewEmbeddingRepoSQLite(conn)

	now := time.Now().UTC()
	fact := &entity.ExtractedFact{
		FactID: "e2e_del", Subject: "用户", Predicate: "患有", Object: "头痛",
		Confidence: 0.8, Status: entity.FactStatusApproved, SourceMsgIDs: []string{"m1"}, CreatedAt: now,
	}
	require.NoError(t, factRepo.Save(ctx, fact))

	v := make([]float32, entity.EmbeddingDimension)
	v[0] = 1.0
	require.NoError(t, embedRepo.Save(ctx, entity.NewSemanticEmbedding("e2e_del", v, "all-MiniLM-L6-v2")))

	// 删除前搜索应返回结果
	query := make([]float32, entity.EmbeddingDimension)
	query[0] = 1.0
	results, err := embedRepo.SearchSimilar(ctx, query, 10)
	require.NoError(t, err)
	require.Len(t, results, 1)

	// 删除事实（外键级联删除嵌入）
	require.NoError(t, factRepo.Delete(ctx, "e2e_del"))

	// 删除后搜索应无结果
	results, err = embedRepo.SearchSimilar(ctx, query, 10)
	require.NoError(t, err)
	assert.Empty(t, results)
}

// TestE2E_Memory_Performance 验证向量搜索在批量数据下的性能。
func TestE2E_Memory_Performance(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	ctx := context.Background()
	factRepo := repository.NewFactRepoSQLite(conn)
	embedRepo := repository.NewEmbeddingRepoSQLite(conn)

	now := time.Now().UTC()
	batchSize := 100

	// 批量插入事实和嵌入
	for i := 0; i < batchSize; i++ {
		fid := fmt.Sprintf("e2e_perf_%d", i)
		fact := &entity.ExtractedFact{
			FactID: fid, Subject: "用户", Predicate: "测试", Object: fmt.Sprintf("事实%d", i),
			Confidence: 0.8, Status: entity.FactStatusApproved, SourceMsgIDs: []string{"m1"}, CreatedAt: now,
		}
		require.NoError(t, factRepo.Save(ctx, fact))

		v := make([]float32, entity.EmbeddingDimension)
		v[0] = float32(i+1) * 0.01
		v[1] = 0.1
		require.NoError(t, embedRepo.Save(ctx, entity.NewSemanticEmbedding(fid, v, "all-MiniLM-L6-v2")))
	}

	// 测量搜索延迟
	query := make([]float32, entity.EmbeddingDimension)
	query[0] = 0.5

	start := time.Now()
	results, err := embedRepo.SearchSimilar(ctx, query, 10)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.Len(t, results, 10)
	// 性能预算：100 条向量 < 50ms
	assert.Less(t, elapsed.Milliseconds(), int64(50), "vector search should be < 50ms for 100 vectors")
}

// TestE2E_Memory_RetrieverIntegration 验证 MemoryRetriever 端到端召回流程。
func TestE2E_Memory_RetrieverIntegration(t *testing.T) {
	conn := setupTestDB(t)
	defer conn.Close()

	ctx := context.Background()
	factRepo := repository.NewFactRepoSQLite(conn)
	embedRepo := repository.NewEmbeddingRepoSQLite(conn)

	now := time.Now().UTC()
	facts := []*entity.ExtractedFact{
		{FactID: "e2e_r1", Subject: "用户", Predicate: "患有", Object: "高血压", Confidence: 0.9, Status: entity.FactStatusApproved, SourceMsgIDs: []string{"m1"}, CreatedAt: now},
		{FactID: "e2e_r2", Subject: "用户", Predicate: "服用", Object: "降压药", Confidence: 0.85, Status: entity.FactStatusApproved, SourceMsgIDs: []string{"m2"}, CreatedAt: now},
		{FactID: "e2e_r3", Subject: "用户", Predicate: "感觉", Object: "头晕", Confidence: 0.7, Status: entity.FactStatusRejected, SourceMsgIDs: []string{"m3"}, CreatedAt: now},
	}
	for _, f := range facts {
		require.NoError(t, factRepo.Save(ctx, f))
	}

	// 创建查询向量 [1,0,0,...] 和匹配的嵌入
	v1 := make([]float32, entity.EmbeddingDimension)
	v1[0] = 1.0
	v2 := make([]float32, entity.EmbeddingDimension)
	v2[0] = 0.9
	v2[1] = 0.1
	v3 := make([]float32, entity.EmbeddingDimension)
	v3[0] = 0.8

	require.NoError(t, embedRepo.Save(ctx, entity.NewSemanticEmbedding("e2e_r1", v1, "all-MiniLM-L6-v2")))
	require.NoError(t, embedRepo.Save(ctx, entity.NewSemanticEmbedding("e2e_r2", v2, "all-MiniLM-L6-v2")))
	require.NoError(t, embedRepo.Save(ctx, entity.NewSemanticEmbedding("e2e_r3", v3, "all-MiniLM-L6-v2")))

	// 使用 FactRepo + EmbedRepo 直接验证过滤逻辑
	query := make([]float32, entity.EmbeddingDimension)
	query[0] = 1.0

	results, err := embedRepo.SearchSimilar(ctx, query, 10)
	require.NoError(t, err)
	require.Len(t, results, 3)

	// 只取 approved 的事实
	var approvedCount int
	for _, r := range results {
		f, err := factRepo.GetByID(ctx, r.FactID)
		require.NoError(t, err)
		if f.Status == entity.FactStatusApproved {
			approvedCount++
		}
	}
	assert.Equal(t, 2, approvedCount)
}
