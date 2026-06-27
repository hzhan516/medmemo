package usecase

import (
	"context"
	"fmt"

	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/domain/repository"
)

// KnowledgeSearchService 提供知识库检索能力。
type KnowledgeSearchService struct {
	repo        repository.KnowledgeRepository
	tokenizer   *KnowledgeTokenizer
	embeddingSvc port.EmbeddingService
}

// NewKnowledgeSearchService 构造函数。
func NewKnowledgeSearchService(repo repository.KnowledgeRepository, tokenizer *KnowledgeTokenizer, embeddingSvc port.EmbeddingService) *KnowledgeSearchService {
	return &KnowledgeSearchService{repo: repo, tokenizer: tokenizer, embeddingSvc: embeddingSvc}
}

// Search 执行混合检索；若向量服务不可用则自动回退到纯关键词检索。
func (s *KnowledgeSearchService) Search(ctx context.Context, query string, limit int) ([]*repository.KnowledgeSearchResult, error) {
	terms := tokenizeToSlice(s.tokenizer, query)
	if len(terms) == 0 {
		return nil, nil
	}

	keywordResults, err := s.repo.SearchKeyword(ctx, terms, limit)
	if err != nil {
		return nil, fmt.Errorf("keyword search failed: %w", err)
	}
	for _, r := range keywordResults {
		r.SourceType = "keyword"
	}

	var vectorResults []*repository.KnowledgeSearchResult
	if s.embeddingSvc != nil && s.embeddingSvc.IsAvailable() {
		embedding, err := s.embeddingSvc.EmbedSingle(ctx, query)
		if err == nil {
			vectorResults, err = s.repo.SearchVector(ctx, embedding, limit)
			if err != nil {
				// 向量检索失败不阻断，记录并回退到关键词
				vectorResults = nil
			}
		}
	}
	for _, r := range vectorResults {
		r.SourceType = "vector"
	}

	if len(vectorResults) == 0 {
		return keywordResults, nil
	}

	hybrid := RRFMerge([][]*repository.KnowledgeSearchResult{keywordResults, vectorResults}, []float64{1.0, 1.0}, 60, limit)
	for _, r := range hybrid {
		if r.SourceType != "keyword" && r.SourceType != "vector" {
			r.SourceType = "hybrid"
		}
	}
	return hybrid, nil
}

// SearchKeyword 仅执行关键词检索。
func (s *KnowledgeSearchService) SearchKeyword(ctx context.Context, query string, limit int) ([]*repository.KnowledgeSearchResult, error) {
	terms := tokenizeToSlice(s.tokenizer, query)
	if len(terms) == 0 {
		return nil, nil
	}
	return s.repo.SearchKeyword(ctx, terms, limit)
}

// SearchVector 仅执行向量检索。
func (s *KnowledgeSearchService) SearchVector(ctx context.Context, query string, limit int) ([]*repository.KnowledgeSearchResult, error) {
	if s.embeddingSvc == nil || !s.embeddingSvc.IsAvailable() {
		return nil, nil
	}
	embedding, err := s.embeddingSvc.EmbedSingle(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	return s.repo.SearchVector(ctx, embedding, limit)
}

func tokenizeToSlice(tokenizer *KnowledgeTokenizer, s string) []string {
	freq := tokenizer.Tokenize(s)
	terms := make([]string, 0, len(freq))
	for term := range freq {
		terms = append(terms, term)
	}
	return terms
}
