package usecase

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTitleGenerator_Generate_Success(t *testing.T) {
	mock := &mockLLMClient{chatReply: "  失眠问题，怎么办？  "}
	gen := NewTitleGenerator(mock)

	title, err := gen.Generate(context.Background(), "我最近失眠很严重，怎么办")
	require.NoError(t, err)
	assert.Equal(t, "失眠问题怎么办", title)
}

func TestTitleGenerator_Generate_Error(t *testing.T) {
	mock := &mockLLMClient{chatErr: fmt.Errorf("network error")}
	gen := NewTitleGenerator(mock)

	_, err := gen.Generate(context.Background(), "头痛怎么办")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "llm title generation failed")
}

func TestTitleGenerator_Generate_ContextCanceled(t *testing.T) {
	mock := &mockLLMClient{chatErr: context.Canceled}
	gen := NewTitleGenerator(mock)

	_, err := gen.Generate(context.Background(), "胃痛")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "llm title generation failed")
}

func TestFallbackTitle_ChineseQuestion(t *testing.T) {
	assert.Equal(t, "头痛怎么办", FallbackTitle("头痛怎么办？吃什么药"))
	assert.Equal(t, "胃痛", FallbackTitle("胃痛？"))
}

func TestFallbackTitle_ChinesePlain(t *testing.T) {
	assert.Equal(t, "最近失眠很严重", FallbackTitle("最近失眠很严重"))
	assert.Equal(t, "这是十个汉字的标…", FallbackTitle("这是十个汉字的标题测试"))
}

func TestFallbackTitle_English(t *testing.T) {
	assert.Equal(t, "I have a headache", FallbackTitle("I have a headache and fever"))
	assert.Equal(t, "Headache pain", FallbackTitle("Headache pain"))
}

func TestFallbackTitle_Empty(t *testing.T) {
	result := FallbackTitle("")
	assert.True(t, strings.HasPrefix(result, "健康咨询_"))
}

func TestSanitizeTitle(t *testing.T) {
	assert.Equal(t, "失眠问题", sanitizeTitle("失眠问题！"))
	assert.Equal(t, "胃痛怎么办", sanitizeTitle("  胃痛，怎么办？  "))
	assert.Equal(t, "abcdefgh", sanitizeTitle("a-b-c-d-e-f-g-h!!!"))
}

func TestTruncateChinese(t *testing.T) {
	assert.Equal(t, "一二三四五六七八", truncateChinese("一二三四五六七八", 8))
	assert.Equal(t, "一二三四五六七八…", truncateChinese("一二三四五六七八九", 8))
	assert.Equal(t, "short", truncateChinese("short", 8))
}

func TestIsMostlyChinese(t *testing.T) {
	assert.True(t, isMostlyChinese("头痛怎么办"))
	assert.False(t, isMostlyChinese("I have a headache"))
	assert.False(t, isMostlyChinese("123456"))
	// 中英文混合，中文过半
	assert.True(t, isMostlyChinese("头痛怎么办head"))
}
