package usecase

import (
	"sort"

	"github.com/hzhan516/medmemo/internal/domain/repository"
)

// RRFMerge 使用 Reciprocal Rank Fusion 合并多路检索结果。
// results 为各路结果数组；weights 为对应权重；k 为 RRF 平滑常数；limit 为返回数量。
func RRFMerge(results [][]*repository.KnowledgeSearchResult, weights []float64, k float64, limit int) []*repository.KnowledgeSearchResult {
	if k <= 0 {
		k = 60
	}
	if len(weights) == 0 {
		weights = make([]float64, len(results))
		for i := range weights {
			weights[i] = 1.0
		}
	}

	scores := make(map[string]*repository.KnowledgeSearchResult)
	for listIdx, list := range results {
		weight := weights[listIdx]
		if weight == 0 {
			continue
		}
		for rank, item := range list {
			key := item.ChunkID
			if existing, ok := scores[key]; ok {
				existing.Score += weight / (k + float64(rank+1))
			} else {
				clone := *item
				clone.Score = weight / (k + float64(rank+1))
				scores[key] = &clone
			}
		}
	}

	merged := make([]*repository.KnowledgeSearchResult, 0, len(scores))
	for _, v := range scores {
		merged = append(merged, v)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].Score > merged[j].Score })
	if limit > 0 && limit < len(merged) {
		merged = merged[:limit]
	}
	return merged
}
