// Package usecase 实现应用用例层。
package usecase

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/pkg/models"
)

// TitleGenerator 编排会话标题的 AI 自动生成。
// 当前已停用：WailsApp.GenerateTitle 直接使用 FallbackTitle，避免将用户首条消息原文发往云端。
// 类型与构造函数保留，减少 Wire 变动；云端标题恢复方案见 TODO(#041)。
type TitleGenerator struct {
	llmClient port.LLMClient
}

// NewTitleGenerator 构造函数，供 Wire 注入。
func NewTitleGenerator(llm port.LLMClient) *TitleGenerator {
	return &TitleGenerator{llmClient: llm}
}

// Generate 基于用户首条消息调用 AI 模型生成标题。
// 当前无生产调用方，保留用于未来随会话 provider 判定并脱敏后的云端标题恢复方案。
func (g *TitleGenerator) Generate(ctx context.Context, userMessage string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	messages := []models.Message{
		{
			Role:    models.RoleSystem,
			Content: "请将用户消息提炼为4-8个汉字的简短标题，不含敏感词和标点，只输出标题本身，不要解释。",
		},
		{
			Role:    models.RoleUser,
			Content: userMessage,
		},
	}

	title, err := g.llmClient.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("llm title generation failed: %w", err)
	}

	return sanitizeTitle(title), nil
}

// FallbackTitle 本地降级规则，不依赖任何外部服务。
// 供 WailsApp 等上层在 AI 生成超时/失败时直接调用。
func FallbackTitle(userMessage string) string {
	trimmed := strings.TrimSpace(userMessage)
	if trimmed == "" {
		return time.Now().Format("健康咨询_20060102_150405")
	}

	// 规则1：中文含问号，取问号前最多8字
	if idx := strings.Index(trimmed, "？"); idx > 0 {
		prefix := trimmed[:idx]
		return truncateChinese(prefix, 8)
	}
	if idx := strings.Index(trimmed, "?"); idx > 0 {
		prefix := trimmed[:idx]
		return truncateChinese(prefix, 8)
	}

	// 规则2：判断是否为中文内容
	if isMostlyChinese(trimmed) {
		return truncateChinese(trimmed, 8)
	}

	// 规则3：英文取前4个单词
	words := strings.Fields(trimmed)
	if len(words) >= 4 {
		return strings.Join(words[:4], " ")
	}
	if len(words) > 0 {
		return strings.Join(words, " ")
	}

	// 兜底
	return time.Now().Format("健康咨询_20060102_150405")
}

// nonAlnumRegex 预编译的正则，用于 sanitizeTitle 去除非字母/数字字符。
var nonAlnumRegex = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// sanitizeTitle 清洗 AI 返回的标题：去除标点、空白、符号，仅保留字母/数字/汉字。
func sanitizeTitle(title string) string {
	// 去除首尾空白
	title = strings.TrimSpace(title)

	// 去除所有非字母、非数字字符（含中英文标点、空白、符号）
	title = nonAlnumRegex.ReplaceAllString(title, "")

	// 截断至 8 个汉字长度
	return truncateChinese(title, 8)
}

// truncateChinese 截断字符串至最多 maxRunes 个 rune，过长时追加 "…"。
func truncateChinese(s string, maxRunes int) string {
	return truncateRunes(s, maxRunes, "…")
}

// isMostlyChinese 判断字符串是否以中文字符为主（>50% rune 为中文）。
func isMostlyChinese(s string) bool {
	runes := []rune(s)
	if len(runes) == 0 {
		return false
	}
	chineseCount := 0
	for _, r := range runes {
		if unicode.Is(unicode.Han, r) {
			chineseCount++
		}
	}
	return chineseCount > len(runes)/2
}
