package desensitizer

import (
	"fmt"
	"strings"
	"testing"
)

// 短文本（~120 字符），包含 1 个敏感实体。
func BenchmarkRuleEngine_ShortText(b *testing.B) {
	engine := NewRuleEngine()
	text := "我叫张三，身份证号是 110105199001011234，手机号 13800138000，请帮我看看健康问题。"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Process(text)
	}
}

// 长文本（~2000 字符），包含多个敏感实体。
func BenchmarkRuleEngine_LongText(b *testing.B) {
	engine := NewRuleEngine()
	var parts []string
	for i := 0; i < 20; i++ {
		parts = append(parts, fmt.Sprintf(
			"用户%d的身份证是 %d，手机号 %d，邮箱 user%d@example.com，",
			i, 110105199001011234+i, 13800138000+i, i,
		))
	}
	text := strings.Join(parts, "")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Process(text)
	}
}

// AC 自动机单独性能。
func BenchmarkAhoCorasick_Search(b *testing.B) {
	ac := NewAhoCorasick([]string{"http://", "https://", "@"})
	text := strings.Repeat("这是一段普通中文文本，没有任何敏感信息。", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ac.Search(text)
	}
}
