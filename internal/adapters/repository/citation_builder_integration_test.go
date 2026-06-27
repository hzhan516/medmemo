package repository

import (
	"context"
	"testing"

	"github.com/hzhan516/medmemo/internal/application/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCitationBuilder_Build(t *testing.T) {
	repo, cleanup := newTestKnowledgeRepo(t)
	defer cleanup()

	ctx := context.Background()
	chunker := usecase.NewKnowledgeChunker(512, 32)
	tokenizer := usecase.NewKnowledgeTokenizer()
	importer := usecase.NewKnowledgeImporter(repo, chunker, tokenizer, nil)

	content := []byte("# 健康饮食\n\n多吃蔬菜有助于降低心血管疾病风险。\n\n# 运动\n\n每周至少 150 分钟中等强度运动。")
	_, err := importer.ImportFile(ctx, "health.md", content, false)
	require.NoError(t, err)

	searcher := usecase.NewKnowledgeSearchService(repo, tokenizer, nil)
	builder := usecase.NewCitationBuilder(repo)

	results, err := searcher.Search(ctx, "心血管疾病", 10)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	citations, snippets, err := builder.Build(ctx, results, 3)
	require.NoError(t, err)
	require.NotEmpty(t, citations)
	require.NotEmpty(t, snippets)

	assert.Equal(t, "health", citations[0].Title)
	assert.Contains(t, citations[0].Snippet, "心血管")
	assert.Greater(t, citations[0].Score, 0.0)
	assert.Contains(t, snippets[0], "心血管")
}
