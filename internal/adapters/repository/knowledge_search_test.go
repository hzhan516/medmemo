package repository

import (
	"context"
	"testing"

	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnowledgeSearchService_KeywordOnly(t *testing.T) {
	repo, cleanup := newTestKnowledgeRepo(t)
	defer cleanup()

	ctx := context.Background()
	chunker := usecase.NewKnowledgeChunker(512, 32)
	tokenizer := usecase.NewKnowledgeTokenizer()
	importer := usecase.NewKnowledgeImporter(repo, chunker, tokenizer, nil)
	searchSvc := usecase.NewKnowledgeSearchService(repo, tokenizer, nil)

	content := []byte("# 健康指南\n\n感冒通常由病毒引起。发热是身体免疫反应。")
	_, err := importer.ImportFile(ctx, "guide.md", content, false)
	require.NoError(t, err)

	results, err := searchSvc.SearchKeyword(ctx, "感冒症状", 10)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.Equal(t, "keyword", results[0].SourceType)
}

func TestKnowledgeSearchService_HybridFallbackToKeyword(t *testing.T) {
	repo, cleanup := newTestKnowledgeRepo(t)
	defer cleanup()

	ctx := context.Background()
	chunker := usecase.NewKnowledgeChunker(512, 32)
	tokenizer := usecase.NewKnowledgeTokenizer()
	importer := usecase.NewKnowledgeImporter(repo, chunker, tokenizer, nil)
	searchSvc := usecase.NewKnowledgeSearchService(repo, tokenizer, nil)

	content := []byte("# 健康指南\n\n感冒通常由病毒引起。")
	_, err := importer.ImportFile(ctx, "guide.md", content, false)
	require.NoError(t, err)

	results, err := searchSvc.Search(ctx, "感冒", 10)
	require.NoError(t, err)
	require.NotEmpty(t, results)
}
