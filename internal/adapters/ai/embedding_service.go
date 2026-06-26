// Package ai 实现 AI 模型客户端适配器簇。
package ai

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/hzhan516/medmemo/internal/application/port"
)

// 编译期接口实现检查。
var _ port.EmbeddingService = (*EmbeddingServiceAdapter)(nil)

// EmbeddingEngine 是 ONNX 嵌入引擎的最小接口，用于解耦测试。
type EmbeddingEngine interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	HasEmbeddingPipeline() bool
}

type embeddingFailureReporter interface {
	EmbeddingFailureReason() string
}

type runtimeLibPathReporter interface {
	RuntimeLibPath() string
}

// EmbeddingServiceAdapter 实现 port.EmbeddingService，包装 ONNX 嵌入引擎。
// 提供批量推理、LRU 缓存和 L2 归一化能力。
type EmbeddingServiceAdapter struct {
	engine        EmbeddingEngine
	modelVersion  string
	batchSize     int
	cache         *embeddingCache
	normalization bool
}

// NewEmbeddingServiceAdapter 构造函数。
func NewEmbeddingServiceAdapter(engine EmbeddingEngine, modelVersion string) *EmbeddingServiceAdapter {
	return &EmbeddingServiceAdapter{
		engine:        engine,
		modelVersion:  modelVersion,
		batchSize:     32,
		cache:         newEmbeddingCache(1000),
		normalization: true,
	}
}

// missEntry 封装缓存 miss 项的原始文本与版本化缓存 key。
type missEntry struct {
	text     string // 原始文本，用于 ONNX 推理
	cacheKey string // modelVersion + "\x00" + text，用于缓存读写
}

// Embed 批量生成文本嵌入。
func (s *EmbeddingServiceAdapter) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if s.engine == nil || !s.engine.HasEmbeddingPipeline() {
		return nil, fmt.Errorf("embedding engine not available")
	}

	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// 阶段一：查缓存，收集 miss
	cacheKeyToIndices := make(map[string][]int) // cacheKey -> original indices
	var missList []missEntry                    // 有序 miss 列表
	result := make([][]float32, len(texts))

	for i, text := range texts {
		cacheKey := s.modelVersion + "\x00" + text
		if cached, ok := s.cache.Get(cacheKey); ok {
			result[i] = cached
			continue
		}
		cacheKeyToIndices[cacheKey] = append(cacheKeyToIndices[cacheKey], i)
		if len(cacheKeyToIndices[cacheKey]) == 1 {
			missList = append(missList, missEntry{text: text, cacheKey: cacheKey})
		}
	}

	if len(missList) == 0 {
		return result, nil
	}

	// 阶段二：批量推理（仅传原始文本给 ONNX）
	var missTexts []string
	for _, entry := range missList {
		missTexts = append(missTexts, entry.text)
	}

	var allEmbeddings [][]float32
	for start := 0; start < len(missTexts); start += s.batchSize {
		end := start + s.batchSize
		if end > len(missTexts) {
			end = len(missTexts)
		}
		batch := missTexts[start:end]

		embeddings, err := s.engine.Embed(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("embedding inference failed for batch %d-%d: %w", start, end, err)
		}
		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	// 阶段三：回填结果（用 cacheKey 写缓存，用 indices 写 result）
	for i, entry := range missList {
		vec := allEmbeddings[i]
		if s.normalization {
			vec = normalizeL2(vec)
		}
		s.cache.Set(entry.cacheKey, vec)
		for _, idx := range cacheKeyToIndices[entry.cacheKey] {
			result[idx] = vec
		}
	}

	return result, nil
}

// EmbedSingle 单条文本嵌入生成。
func (s *EmbeddingServiceAdapter) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	result, err := s.Embed(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("failed to embed single text: %w", err)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("empty embedding result")
	}
	return result[0], nil
}

// IsAvailable 返回真实嵌入推理能力是否可用。
func (s *EmbeddingServiceAdapter) IsAvailable() bool {
	return s.engine != nil && s.engine.HasEmbeddingPipeline()
}

// FailureReason 返回嵌入引擎最近一次初始化失败原因。
func (s *EmbeddingServiceAdapter) FailureReason() string {
	if s.engine == nil {
		return "embedding engine not initialized"
	}
	if reporter, ok := s.engine.(embeddingFailureReporter); ok {
		return reporter.EmbeddingFailureReason()
	}
	if !s.engine.HasEmbeddingPipeline() {
		return "embedding engine not available"
	}
	return ""
}

// RuntimeLibPath 返回嵌入引擎解析到的 ONNX Runtime 动态库路径。
func (s *EmbeddingServiceAdapter) RuntimeLibPath() string {
	if s.engine == nil {
		return ""
	}
	if reporter, ok := s.engine.(runtimeLibPathReporter); ok {
		return reporter.RuntimeLibPath()
	}
	return ""
}

// ModelVersion 返回当前使用的模型版本。
func (s *EmbeddingServiceAdapter) ModelVersion() string {
	return s.modelVersion
}

// normalizeL2 对向量进行 L2 归一化。
func normalizeL2(v []float32) []float32 {
	var norm float64
	for _, x := range v {
		norm += float64(x) * float64(x)
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		out := make([]float32, len(v))
		return out
	}
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(float64(x) / norm)
	}
	return out
}

// =============================================================================
// LRU Cache (简单实现，避免外部依赖)
// =============================================================================

type cacheEntry struct {
	key   string
	value []float32
	next  *cacheEntry
	prev  *cacheEntry
}

type embeddingCache struct {
	mu       sync.RWMutex
	capacity int
	entries  map[string]*cacheEntry
	head     *cacheEntry // 最近使用
	tail     *cacheEntry // 最久未使用
}

func newEmbeddingCache(capacity int) *embeddingCache {
	c := &embeddingCache{
		capacity: capacity,
		entries:  make(map[string]*cacheEntry),
		head:     &cacheEntry{},
		tail:     &cacheEntry{},
	}
	c.head.next = c.tail
	c.tail.prev = c.head
	return c
}

func (c *embeddingCache) Get(key string) ([]float32, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	c.mu.Lock()
	c.moveToFront(entry)
	c.mu.Unlock()
	return entry.value, true
}

func (c *embeddingCache) Set(key string, value []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.entries[key]; ok {
		entry.value = copyVector(value)
		c.moveToFront(entry)
		return
	}

	entry := &cacheEntry{key: key, value: copyVector(value)}
	c.entries[key] = entry
	c.addToFront(entry)

	if len(c.entries) > c.capacity {
		c.evictLRU()
	}
}

func (c *embeddingCache) moveToFront(entry *cacheEntry) {
	c.remove(entry)
	c.addToFront(entry)
}

func (c *embeddingCache) addToFront(entry *cacheEntry) {
	entry.next = c.head.next
	entry.prev = c.head
	c.head.next.prev = entry
	c.head.next = entry
}

func (c *embeddingCache) remove(entry *cacheEntry) {
	entry.prev.next = entry.next
	entry.next.prev = entry.prev
}

func (c *embeddingCache) evictLRU() {
	entry := c.tail.prev
	if entry == c.head {
		return
	}
	c.remove(entry)
	delete(c.entries, entry.key)
}

func copyVector(v []float32) []float32 {
	out := make([]float32, len(v))
	copy(out, v)
	return out
}
