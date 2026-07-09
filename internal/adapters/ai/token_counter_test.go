package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHFTokenCounter_EmptyText 空文本应返回 (0, true)，表示精确结果。
func TestHFTokenCounter_EmptyText(t *testing.T) {
	c := NewHFTokenCounter()
	n, ok := c.Count(context.Background(), "any-model", "")
	assert.Equal(t, 0, n)
	assert.True(t, ok)
}

// TestHFTokenCounter_CanceledContext 上下文取消时应返回 (0, false)。
func TestHFTokenCounter_CanceledContext(t *testing.T) {
	c := NewHFTokenCounter()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n, ok := c.Count(ctx, "any-model", "hello world")
	assert.Equal(t, 0, n)
	assert.False(t, ok)
}

// TestHFTokenCounter_NotReadyFallback 分词器未就绪时使用字符启发式回退，ok=false。
func TestHFTokenCounter_NotReadyFallback(t *testing.T) {
	c := NewHFTokenCounter()
	// "hello" 为 5 个 ASCII 字符 => (5+3)/4 = 2
	n, ok := c.Count(context.Background(), "any-model", "hello")
	assert.Equal(t, 2, n)
	assert.False(t, ok, "未加载分词器时应标记为近似值")
}

// TestHFTokenCounter_LoadNilKeepsNotReady 传入 nil 分词器应保持未就绪状态。
func TestHFTokenCounter_LoadNilKeepsNotReady(t *testing.T) {
	c := NewHFTokenCounter()
	c.Load(nil)
	n, ok := c.Count(context.Background(), "m", "中文")
	// 两个 CJK 字符 => 2
	assert.Equal(t, 2, n)
	assert.False(t, ok)
}

// TestCharHeuristic 验证字符启发式估算：CJK 每字 1 token，其余每 4 字符 1 token 向上取整。
func TestCharHeuristic(t *testing.T) {
	cases := []struct {
		text string
		want int
	}{
		{"", 0},
		{"a", 1},      // (1+3)/4 = 1
		{"abcd", 1},   // (4+3)/4 = 1
		{"abcde", 2},  // (5+3)/4 = 2
		{"中", 1},      // 1 CJK
		{"中文世界", 4},   // 4 CJK
		{"中a", 2},     // 1 CJK + (1+3)/4=1
		{"你好abcd", 3}, // 2 CJK + (4+3)/4=1
	}
	for _, tt := range cases {
		assert.Equalf(t, tt.want, charHeuristic(tt.text), "charHeuristic(%q)", tt.text)
	}
}

// TestIsCJK 验证中日韩字符判定覆盖各 Unicode 区块与非 CJK 字符。
func TestIsCJK(t *testing.T) {
	cjk := []rune{'中', '文', '\u3400', 'あ', 'ア', '한'}
	for _, r := range cjk {
		assert.Truef(t, isCJK(r), "expected %U to be CJK", r)
	}
	nonCJK := []rune{'a', 'Z', '0', ' ', '!', '\n', '€'}
	for _, r := range nonCJK {
		assert.Falsef(t, isCJK(r), "expected %U to be non-CJK", r)
	}
}
