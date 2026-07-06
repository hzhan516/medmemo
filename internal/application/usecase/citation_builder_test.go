package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// citationRepoStub 仅实现 CitationBuilder 依赖的 FindDocument，
// 其余 KnowledgeRepository 方法由内嵌接口占位（调用会 panic，本用例不触发）。
type citationRepoStub struct {
	repository.KnowledgeRepository
	docs      map[string]*entity.KnowledgeDocument
	failOnID  map[string]bool
	callOrder []string
}

func (s *citationRepoStub) FindDocument(_ context.Context, id string) (*entity.KnowledgeDocument, error) {
	s.callOrder = append(s.callOrder, id)
	if s.failOnID[id] {
		return nil, errors.New("document not found")
	}
	doc, ok := s.docs[id]
	if !ok {
		return nil, errors.New("document missing")
	}
	return doc, nil
}

func newCitationRepoStub() *citationRepoStub {
	return &citationRepoStub{
		docs:     make(map[string]*entity.KnowledgeDocument),
		failOnID: make(map[string]bool),
	}
}

func TestCitationBuilder_Build_BasicMapping(t *testing.T) {
	t.Parallel()

	repo := newCitationRepoStub()
	repo.docs["doc-1"] = &entity.KnowledgeDocument{
		DocumentID: "doc-1",
		Title:      "感冒护理指南",
		Citation:   "中华医学会2023",
		URL:        "https://example.com/doc-1",
		SourceType: entity.KnowledgeSourceMarkdown,
	}

	builder := NewCitationBuilder(repo)
	results := []*repository.KnowledgeSearchResult{
		{DocumentID: "doc-1", Content: "感冒通常由病毒引起。", Score: 0.9},
	}

	citations, snippets, err := builder.Build(context.Background(), results, 3)
	require.NoError(t, err)
	require.Len(t, citations, 1)
	require.Len(t, snippets, 1)

	c := citations[0]
	assert.Equal(t, "doc-1", c.ID)
	assert.Equal(t, "感冒护理指南", c.Title)
	assert.Equal(t, string(entity.KnowledgeSourceMarkdown), c.Source)
	assert.Equal(t, "https://example.com/doc-1", c.URL)
	assert.Equal(t, "感冒通常由病毒引起。", c.Snippet)
	assert.InDelta(t, 0.9, c.Score, 0.0001)

	// 摘要文本以 "[引用] 内容" 形式拼接
	assert.Equal(t, "[中华医学会2023] 感冒通常由病毒引起。", snippets[0])
}

func TestCitationBuilder_Build_DeduplicatesByDocument(t *testing.T) {
	t.Parallel()

	repo := newCitationRepoStub()
	repo.docs["doc-1"] = &entity.KnowledgeDocument{DocumentID: "doc-1", Title: "T1", Citation: "C1"}
	repo.docs["doc-2"] = &entity.KnowledgeDocument{DocumentID: "doc-2", Title: "T2", Citation: "C2"}

	builder := NewCitationBuilder(repo)
	results := []*repository.KnowledgeSearchResult{
		{DocumentID: "doc-1", Content: "片段A", Score: 0.9},
		{DocumentID: "doc-1", Content: "片段B", Score: 0.8}, // 同文档，应被去重
		{DocumentID: "doc-2", Content: "片段C", Score: 0.7},
	}

	citations, _, err := builder.Build(context.Background(), results, 5)
	require.NoError(t, err)
	require.Len(t, citations, 2)
	assert.Equal(t, "doc-1", citations[0].ID)
	assert.Equal(t, "doc-2", citations[1].ID)
	// FindDocument 对重复文档只应调用一次
	assert.Equal(t, []string{"doc-1", "doc-2"}, repo.callOrder)
}

func TestCitationBuilder_Build_RespectsMaxCitations(t *testing.T) {
	t.Parallel()

	repo := newCitationRepoStub()
	repo.docs["doc-1"] = &entity.KnowledgeDocument{DocumentID: "doc-1", Title: "T1", Citation: "C1"}
	repo.docs["doc-2"] = &entity.KnowledgeDocument{DocumentID: "doc-2", Title: "T2", Citation: "C2"}
	repo.docs["doc-3"] = &entity.KnowledgeDocument{DocumentID: "doc-3", Title: "T3", Citation: "C3"}

	builder := NewCitationBuilder(repo)
	results := []*repository.KnowledgeSearchResult{
		{DocumentID: "doc-1", Content: "a"},
		{DocumentID: "doc-2", Content: "b"},
		{DocumentID: "doc-3", Content: "c"},
	}

	citations, _, err := builder.Build(context.Background(), results, 2)
	require.NoError(t, err)
	assert.Len(t, citations, 2)
}

func TestCitationBuilder_Build_DefaultsMaxWhenNonPositive(t *testing.T) {
	t.Parallel()

	repo := newCitationRepoStub()
	for _, id := range []string{"doc-1", "doc-2", "doc-3", "doc-4"} {
		repo.docs[id] = &entity.KnowledgeDocument{DocumentID: id, Title: id, Citation: id}
	}

	builder := NewCitationBuilder(repo)
	results := []*repository.KnowledgeSearchResult{
		{DocumentID: "doc-1"}, {DocumentID: "doc-2"},
		{DocumentID: "doc-3"}, {DocumentID: "doc-4"},
	}

	// maxCitations <= 0 时应回退到默认值 3
	citations, _, err := builder.Build(context.Background(), results, 0)
	require.NoError(t, err)
	assert.Len(t, citations, 3)
}

func TestCitationBuilder_Build_SkipsNilAndMissingDocuments(t *testing.T) {
	t.Parallel()

	repo := newCitationRepoStub()
	repo.docs["doc-2"] = &entity.KnowledgeDocument{DocumentID: "doc-2", Title: "T2", Citation: "C2"}
	repo.failOnID["doc-1"] = true // 文档查询失败应跳过而非阻断

	builder := NewCitationBuilder(repo)
	results := []*repository.KnowledgeSearchResult{
		nil, // nil 结果应被忽略
		{DocumentID: "doc-1", Content: "缺失文档"},
		{DocumentID: "doc-2", Content: "有效文档"},
	}

	citations, _, err := builder.Build(context.Background(), results, 3)
	require.NoError(t, err)
	require.Len(t, citations, 1)
	assert.Equal(t, "doc-2", citations[0].ID)
}

func TestCitationBuilder_Build_TitleFallback(t *testing.T) {
	t.Parallel()

	repo := newCitationRepoStub()
	// Title 为空 -> 回退到 Citation
	repo.docs["doc-1"] = &entity.KnowledgeDocument{DocumentID: "doc-1", Citation: "仅有引用"}
	// Title 与 Citation 均为空 -> 回退到默认 "知识库文档"
	repo.docs["doc-2"] = &entity.KnowledgeDocument{DocumentID: "doc-2"}

	builder := NewCitationBuilder(repo)
	results := []*repository.KnowledgeSearchResult{
		{DocumentID: "doc-1", Content: "x"},
		{DocumentID: "doc-2", Content: "y"},
	}

	citations, _, err := builder.Build(context.Background(), results, 3)
	require.NoError(t, err)
	require.Len(t, citations, 2)
	assert.Equal(t, "仅有引用", citations[0].Title)
	assert.Equal(t, "知识库文档", citations[1].Title)
}

func TestCitationBuilder_Build_TruncatesLongSnippet(t *testing.T) {
	t.Parallel()

	repo := newCitationRepoStub()
	repo.docs["doc-1"] = &entity.KnowledgeDocument{DocumentID: "doc-1", Title: "T", Citation: "C"}

	builder := NewCitationBuilder(repo)
	longContent := strings.Repeat("测", 350) // 350 个中文字符，超过 300 上限
	results := []*repository.KnowledgeSearchResult{
		{DocumentID: "doc-1", Content: longContent},
	}

	citations, _, err := builder.Build(context.Background(), results, 3)
	require.NoError(t, err)
	require.Len(t, citations, 1)

	snippet := citations[0].Snippet
	assert.True(t, strings.HasSuffix(snippet, "..."), "long snippet should end with ellipsis")
	// 截断后保留 300 个 rune 加上省略号
	assert.Equal(t, 300, len([]rune(strings.TrimSuffix(snippet, "..."))))
}

func TestCitationBuilder_Build_EmptyResults(t *testing.T) {
	t.Parallel()

	builder := NewCitationBuilder(newCitationRepoStub())
	citations, snippets, err := builder.Build(context.Background(), nil, 3)
	require.NoError(t, err)
	assert.Empty(t, citations)
	assert.Empty(t, snippets)
}
