package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
)

// CitationBuilder 将知识检索结果转换为前端可展示的引用条目。
type CitationBuilder struct {
	repo repository.KnowledgeRepository
}

// NewCitationBuilder 构造函数。
func NewCitationBuilder(repo repository.KnowledgeRepository) *CitationBuilder {
	return &CitationBuilder{repo: repo}
}

// Build 从检索结果中生成引用列表与用于注入系统上下文的摘要文本。
// 按文档去重，最多返回 maxCitations 条，摘要长度限制 300 字符。
func (b *CitationBuilder) Build(ctx context.Context, results []*repository.KnowledgeSearchResult, maxCitations int) ([]entity.KnowledgeCitation, []string, error) {
	if maxCitations <= 0 {
		maxCitations = 3
	}

	citations := make([]entity.KnowledgeCitation, 0, maxCitations)
	snippets := make([]string, 0, maxCitations)
	seen := make(map[string]bool)

	for _, r := range results {
		if r == nil {
			continue
		}
		if len(citations) >= maxCitations {
			break
		}
		if seen[r.DocumentID] {
			continue
		}
		seen[r.DocumentID] = true

		doc, err := b.repo.FindDocument(ctx, r.DocumentID)
		if err != nil {
			// 文档缺失时不阻断整体引用生成
			continue
		}

		snippet := truncateCitationSnippet(r.Content, 300)
		citations = append(citations, entity.KnowledgeCitation{
			ID:      r.DocumentID,
			Title:   firstNonEmpty(doc.Title, doc.Citation, "知识库文档"),
			Source:  string(doc.SourceType),
			URL:     doc.URL,
			Snippet: snippet,
			Score:   r.Score,
		})
		snippets = append(snippets, fmt.Sprintf("[%s] %s", doc.Citation, snippet))
	}

	return citations, snippets, nil
}

func truncateCitationSnippet(s string, maxLen int) string {
	if maxLen <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
