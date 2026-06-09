package usecase

import (
	"strings"
	"unicode"
)

// QueryExpansionService 负责 query 的规范化与扩展。
// 完全本地运行，不依赖任何外部服务。
type QueryExpansionService struct{}

// NewQueryExpansionService 创建新的 query 扩展服务。
func NewQueryExpansionService() *QueryExpansionService {
	return &QueryExpansionService{}
}

// Normalize 对原始 query 进行规范化处理：
// 1. 去除首尾空白
// 2. 去除常见中文/英文标点
// 3. 全角数字/字母转半角
// 4. 合并连续空白
func (s *QueryExpansionService) Normalize(query string) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return ""
	}

	// 去除常见标点
	punctuations := "？?。！!，,、；;：:（）()【】[]《》<>\"\"''「」『』"
	for _, p := range punctuations {
		q = strings.ReplaceAll(q, string(p), "")
	}

	// 全角转半角 + 合并连续空白
	var b strings.Builder
	var lastSpace bool
	for _, r := range q {
		r = toHalfWidth(r)
		if unicode.IsSpace(r) {
			if !lastSpace {
				b.WriteRune(' ')
				lastSpace = true
			}
			continue
		}
		b.WriteRune(r)
		lastSpace = false
	}
	return strings.TrimSpace(b.String())
}

// toHalfWidth 将全角数字、字母、空格转为半角。
func toHalfWidth(r rune) rune {
	switch {
	case r >= '０' && r <= '９':
		return r - '０' + '0'
	case r >= 'Ａ' && r <= 'Ｚ':
		return r - 'Ａ' + 'A'
	case r >= 'ａ' && r <= 'ｚ':
		return r - 'ａ' + 'a'
	case r == '　':
		return ' '
	default:
		return r
	}
}
