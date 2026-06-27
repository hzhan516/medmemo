package usecase

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hzhan516/medmemo/internal/domain/entity"
)

// KnowledgeChunker 将原始文档内容切分为适合检索的片段。
type KnowledgeChunker struct {
	chunkSize  int
	overlap    int
	tokenEstimator func(string) int
}

// NewKnowledgeChunker 构造函数。
// chunkSize 为每个片段的目标字符数，overlap 为相邻片段的重叠字符数。
func NewKnowledgeChunker(chunkSize, overlap int) *KnowledgeChunker {
	if chunkSize <= 0 {
		chunkSize = 512
	}
	if overlap < 0 || overlap >= chunkSize {
		overlap = chunkSize / 8
	}
	return &KnowledgeChunker{
		chunkSize:      chunkSize,
		overlap:        overlap,
		tokenEstimator: defaultTokenEstimator,
	}
}

// NewDefaultKnowledgeChunker 返回默认参数的片段切分器，供 Wire 注入。
func NewDefaultKnowledgeChunker() *KnowledgeChunker {
	return NewKnowledgeChunker(200, 20)
}

// defaultTokenEstimator 使用字符数粗略估计 token 数（中文约 1.5 字符/token）。
func defaultTokenEstimator(s string) int {
	length := len([]rune(s))
	return length*2/3 + 1
}

// ChunkMarkdown 按 Markdown 标题或段落边界切分。
func (c *KnowledgeChunker) ChunkMarkdown(title string, content []byte) []*entity.KnowledgeChunk {
	text := string(content)
	// 按二级及以下标题切分
	sections := splitByHeaders(text)
	if len(sections) == 0 {
		sections = []string{text}
	}

	var chunks []*entity.KnowledgeChunk
	idx := 0
	for _, sec := range sections {
		sec = strings.TrimSpace(sec)
		if sec == "" {
			continue
		}
		for start := 0; start < len(sec); {
			end := start + c.chunkSize
			if end > len(sec) {
				end = len(sec)
			}
			piece := sec[start:end]
			chunks = append(chunks, &entity.KnowledgeChunk{
				ChunkIndex: idx,
				Content:    piece,
				TokenCount: c.tokenEstimator(piece),
			})
			idx++
			if end >= len(sec) {
				break
			}
			start = end - c.overlap
			if start < 0 {
				start = 0
			}
		}
	}
	return chunks
}

// ChunkJSONL 解析 JSON Lines，提取每行的 text/content/title 字段。
func (c *KnowledgeChunker) ChunkJSONL(content []byte) ([]*entity.KnowledgeChunk, error) {
	var chunks []*entity.KnowledgeChunk
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Split(bufio.ScanLines)
	idx := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			return nil, fmt.Errorf("failed to parse JSONL line %d: %w", idx, err)
		}
		text := extractTextField(obj)
		if text == "" {
			return nil, fmt.Errorf("JSONL line %d missing text/content/title field", idx)
		}
		chunks = append(chunks, &entity.KnowledgeChunk{
			ChunkIndex: idx,
			Content:    text,
			TokenCount: c.tokenEstimator(text),
		})
		idx++
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan JSONL content: %w", err)
	}
	return chunks, nil
}

// splitByHeaders 按 Markdown 标题切分文本。
func splitByHeaders(text string) []string {
	lines := strings.Split(text, "\n")
	var sections []string
	var current strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			if current.Len() > 0 {
				sections = append(sections, current.String())
				current.Reset()
			}
		}
		current.WriteString(line)
		current.WriteString("\n")
	}
	if current.Len() > 0 {
		sections = append(sections, current.String())
	}
	return sections
}

// extractTextField 从 JSON 对象中提取文本字段。
func extractTextField(obj map[string]interface{}) string {
	for _, key := range []string{"text", "content", "title", "snippet"} {
		if v, ok := obj[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}
