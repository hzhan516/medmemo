package usecase

import (
	"strings"

	"github.com/hzhan516/medmemo/internal/domain/repository"
)

// KnowledgeReranker 提供轻量级重排序：精确短语匹配与标题匹配加分。
type KnowledgeReranker struct {
	phraseBoost float64
	titleBoost  float64
	// beforeRerank 仅用于测试，记录每次重排的入参与调用次数。
	beforeRerank func(results []*repository.KnowledgeSearchResult, query string)
}

// NewKnowledgeReranker 构造函数。
func NewKnowledgeReranker() *KnowledgeReranker {
	return &KnowledgeReranker{phraseBoost: 0.5, titleBoost: 0.3}
}

// SetBeforeRerankHook 仅用于测试，注入回调以观察 Rerank 调用。
func (r *KnowledgeReranker) SetBeforeRerankHook(hook func(results []*repository.KnowledgeSearchResult, query string)) {
	r.beforeRerank = hook
}

// Rerank 根据查询对结果进行规则重排。
func (r *KnowledgeReranker) Rerank(results []*repository.KnowledgeSearchResult, query string) []*repository.KnowledgeSearchResult {
	if r.beforeRerank != nil {
		r.beforeRerank(results, query)
	}
	lowerQuery := strings.ToLower(query)
	for _, res := range results {
		content := strings.ToLower(res.Content)
		if strings.Contains(content, lowerQuery) {
			res.Score += r.phraseBoost
		}
	}
	sortSearchResults(results)
	return results
}

func sortSearchResults(results []*repository.KnowledgeSearchResult) {
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}
