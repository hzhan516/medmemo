package ai

import (
	"context"
	"sync"

	"github.com/daulet/tokenizers"
)

// HFTokenCounter 基于 Hugging Face 分词器实现 port.TokenCounter。
// 在分词器尚未加载时使用字符启发式回退，保证估算始终可用。
type HFTokenCounter struct {
	mu        sync.RWMutex
	tokenizer *tokenizers.Tokenizer
	ready     bool
}

// NewHFTokenCounter 创建 HFTokenCounter 实例。
// 此时 tokenizer 尚未加载，ready 为 false，调用方后续可通过 Load 注入分词器。
func NewHFTokenCounter() *HFTokenCounter {
	return &HFTokenCounter{}
}

// Load 注入已初始化的 tokenizer 并将 ready 置为 true。
// 线程安全：内部使用写锁保护状态变更。
func (c *HFTokenCounter) Load(t *tokenizers.Tokenizer) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tokenizer != nil {
		_ = c.tokenizer.Close()
	}

	c.tokenizer = t
	c.ready = t != nil
}

// Count 返回 text 的 token 估算数量。
//
// 行为约定：
//   - text 为空时返回 (0, true)；
//   - 上下文被取消时返回 (0, false)；
//   - 分词器未就绪或为空时返回 (charHeuristic(text), false)；
//   - 分词器就绪时编码文本并返回 (len(ids), true)。
func (c *HFTokenCounter) Count(ctx context.Context, modelID, text string) (int, bool) {
	if text == "" {
		return 0, true
	}

	if ctx.Err() != nil {
		return 0, false
	}

	_ = modelID // 保留参数以满足接口；当前实现不区分模型

	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.ready || c.tokenizer == nil {
		return charHeuristic(text), false
	}

	ids, _, err := c.tokenizer.EncodeErr(text, false)
	if err != nil {
		return charHeuristic(text), false
	}

	return len(ids), true
}

// charHeuristic 基于字符数量的启发式 token 估算。
// CJK 字符按 1 token 估算，其他字符按每 4 个字符 1 token 向上取整。
func charHeuristic(text string) int {
	var cjk, other int
	for _, r := range text {
		if isCJK(r) {
			cjk++
		} else {
			other++
		}
	}
	return cjk + (other+3)/4
}

// isCJK 判断 rune 是否属于中日韩统一表意文字或假名、谚文。
func isCJK(r rune) bool {
	return (r >= '\u4e00' && r <= '\u9fff') ||
		(r >= '\u3400' && r <= '\u4dbf') ||
		(r >= '\u3040' && r <= '\u309f') ||
		(r >= '\u30a0' && r <= '\u30ff') ||
		(r >= '\uac00' && r <= '\ud7af')
}
