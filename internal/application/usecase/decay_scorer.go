package usecase

import (
	"math"
	"sort"
	"time"
)

// DecayScorer 基于指数衰减的记忆相关性评分器。
// 公式: score = similarity * exp(-lambda * ageDays)
type DecayScorer struct {
	lambda float64 // 衰减系数，默认 0.05（半衰期 ≈ 13.86 天）
}

// scoredItem 内部结构，用于批量排序
type scoredItem struct {
	index int
	score float64
}

// NewDecayScorer 使用默认衰减系数（lambda=0.05）创建评分器。
func NewDecayScorer() *DecayScorer {
	return &DecayScorer{lambda: 0.05}
}

// NewDecayScorerWithLambda 使用自定义衰减系数创建评分器。
// lambda 越大衰减越快。半衰期 = ln(2) / lambda。
func NewDecayScorerWithLambda(lambda float64) *DecayScorer {
	if lambda < 0 {
		lambda = 0
	}
	return &DecayScorer{lambda: lambda}
}

// Score 对单条记忆计算衰减后的相关性分数。
// similarity: 原始相似度（如余弦相似度），范围 [0,1]，超出范围会被截断
// ageDays: 记忆距今天数，负值按 0 处理
// 返回: 衰减后的分数，范围 [0,1]
func (s *DecayScorer) Score(similarity float64, ageDays float64) float64 {
	// 截断 similarity 到 [0,1]
	if similarity > 1.0 {
		similarity = 1.0
	}
	if similarity < 0.0 {
		similarity = 0.0
	}

	// 负年龄按 0 处理（未来创建的记忆不衰减）
	if ageDays < 0 {
		ageDays = 0
	}

	return similarity * math.Exp(-s.lambda*ageDays)
}

// ScoreFromCreatedAt 根据创建时间计算衰减分数。
// similarity: 原始相似度
// createdAt: 记忆创建时间
// referenceTime: 参考时间（通常取 time.Now()）
func (s *DecayScorer) ScoreFromCreatedAt(similarity float64, createdAt time.Time, referenceTime time.Time) float64 {
	ageDays := referenceTime.Sub(createdAt).Hours() / 24.0
	return s.Score(similarity, ageDays)
}

// Rank 批量计算衰减分数并返回按分数降序排列的结果。
// 参数为变长列表，每对为 (similarity, ageDays)。
// 返回与输入对数相同的分数切片，已按降序排列。
func (s *DecayScorer) Rank(pairs ...float64) []float64 {
	if len(pairs) == 0 {
		return nil
	}

	// 每两个参数构成一对
	pairCount := len(pairs) / 2
	items := make([]scoredItem, 0, pairCount)

	for i := 0; i < pairCount; i++ {
		similarity := pairs[i*2]
		ageDays := pairs[i*2+1]
		score := s.Score(similarity, ageDays)
		items = append(items, scoredItem{index: i, score: score})
	}

	// 按分数降序排序
	sort.Slice(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})

	// 提取排序后的分数
	result := make([]float64, pairCount)
	for i, item := range items {
		result[i] = item.score
	}

	return result
}
