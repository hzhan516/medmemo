package usecase

import (
	"strings"
	"unicode"
)

// KnowledgeTokenizer 将查询/文档切分为词项。
// CJK 使用 bigram/trigram，ASCII 使用单词切分。
type KnowledgeTokenizer struct {
	stopwords map[string]bool
}

// NewKnowledgeTokenizer 构造函数。
func NewKnowledgeTokenizer() *KnowledgeTokenizer {
	return &KnowledgeTokenizer{
		stopwords: map[string]bool{
			"the": true, "a": true, "an": true, "is": true, "are": true, "was": true, "were": true,
			"and": true, "or": true, "of": true, "to": true, "in": true, "for": true, "with": true,
			"的": true, "了": true, "和": true, "是": true, "在": true, "有": true, "我": true, "就": true,
			"不": true, "人": true, "都": true, "一": true, "一个": true, "上": true, "也": true, "很": true,
			"到": true, "说": true, "要": true, "去": true, "你": true, "会": true, "着": true, "没有": true,
			"看": true, "好": true, "自己": true, "这": true,
		},
	}
}

// Tokenize 返回输入字符串的词项集合及词频。
func (t *KnowledgeTokenizer) Tokenize(s string) map[string]int {
	s = strings.ToLower(s)
	freq := make(map[string]int)
	runes := []rune(s)

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if isCJK(r) {
			// CJK bigram + trigram
			if i+1 < len(runes) && isCJK(runes[i+1]) {
				bigram := string(runes[i : i+2])
				if !t.isStopword(bigram) {
					freq[bigram]++
				}
			}
			if i+2 < len(runes) && isCJK(runes[i+1]) && isCJK(runes[i+2]) {
				trigram := string(runes[i : i+3])
				if !t.isStopword(trigram) {
					freq[trigram]++
				}
			}
		} else if isASCIILetterOrDigit(r) {
			// 提取连续 ASCII 单词
			start := i
			for i < len(runes) && isASCIILetterOrDigit(runes[i]) {
				i++
			}
			word := string(runes[start:i])
			if !t.isStopword(word) && len(word) > 1 {
				freq[word]++
			}
			i-- // 回退一位，外层循环会 ++
		}
	}
	return freq
}

func (t *KnowledgeTokenizer) isStopword(s string) bool {
	return t.stopwords[s]
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		(r >= 0x3040 && r <= 0x309F) || // Hiragana
		(r >= 0x30A0 && r <= 0x30FF) || // Katakana
		(r >= 0xAC00 && r <= 0xD7AF) // Hangul
}

func isASCIILetterOrDigit(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}
