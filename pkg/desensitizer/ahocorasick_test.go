package desensitizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAhoCorasick_SinglePattern(t *testing.T) {
		t.Parallel()
	ac := NewAhoCorasick([]string{"http://"})
	matches := ac.Search("访问 http://example.com 获取信息")
	assert.Len(t, matches, 1)
	assert.Equal(t, "http://", matches[0].Pattern)
	assert.Equal(t, 7, matches[0].Start)
	assert.Equal(t, 14, matches[0].End)
}

func TestAhoCorasick_MultiplePatterns(t *testing.T) {
		t.Parallel()
	ac := NewAhoCorasick([]string{"@", "http://", "https://"})
	matches := ac.Search("联系 a@b.com 或访问 https://example.com")

	// 期望命中 @、https://
	patterns := make(map[string]int)
	for _, m := range matches {
		patterns[m.Pattern]++
	}
	assert.Equal(t, 1, patterns["@"])
	assert.Equal(t, 1, patterns["https://"])
	assert.Equal(t, 0, patterns["http://"])
}

func TestAhoCorasick_OverlappingMatches(t *testing.T) {
		t.Parallel()
	ac := NewAhoCorasick([]string{"aa", "aaa"})
	matches := ac.Search("aaaa")
	// aaaa 中包含：aa(0-2), aa(1-3), aaa(0-3), aaa(1-4)
	assert.Equal(t, 5, len(matches))
}

func TestAhoCorasick_ChineseText(t *testing.T) {
		t.Parallel()
	ac := NewAhoCorasick([]string{"@"})
	matches := ac.Search("请联系 张三@公司.com")
	assert.Len(t, matches, 1)
	assert.Equal(t, 16, matches[0].Start) // 3*5 + 1 = 16 (请联系 张三)
	assert.Equal(t, 17, matches[0].End)
}

func TestAhoCorasick_NoMatch(t *testing.T) {
		t.Parallel()
	ac := NewAhoCorasick([]string{"http://", "https://"})
	matches := ac.Search("这是一段纯中文文本，没有任何链接")
	assert.Empty(t, matches)
}

func TestAhoCorasick_EmptyPatterns(t *testing.T) {
		t.Parallel()
	ac := NewAhoCorasick([]string{})
	matches := ac.Search("任意文本")
	assert.Empty(t, matches)
}

func TestAhoCorasick_EmptyText(t *testing.T) {
		t.Parallel()
	ac := NewAhoCorasick([]string{"@"})
	matches := ac.Search("")
	assert.Empty(t, matches)
}

func TestAhoCorasick_MultipleSamePattern(t *testing.T) {
		t.Parallel()
	ac := NewAhoCorasick([]string{"@"})
	matches := ac.Search("a@b.com 和 c@d.com")
	assert.Len(t, matches, 2)
	assert.Equal(t, 1, matches[0].Start)
	assert.Equal(t, 13, matches[1].Start)
}
