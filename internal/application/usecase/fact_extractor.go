// Package usecase 应用用例层，编排领域对象完成完整业务流程。
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/domain/repository"
)

// ErrRateLimited 表示事实提取速率超过限制。
var ErrRateLimited = fmt.Errorf("fact extraction rate limited")

// FactLLMClient 是事实提取所需的最小 LLM 接口。
type FactLLMClient interface {
	Chat(ctx context.Context, messages []string) (string, error)
}

// FactExtractor 基于 LLM 的事实提取服务。
type FactExtractor struct {
	llm       FactLLMClient
	rateLimit int         // 每分钟最大调用次数
	callTimes []time.Time // 最近调用时间戳
	mu        sync.Mutex
}

// NewFactExtractor 构造函数。
func NewFactExtractor(llm FactLLMClient) *FactExtractor {
	return &FactExtractor{
		llm:       llm,
		rateLimit: 5,
		callTimes: make([]time.Time, 0),
	}
}

// ParseFacts 从单条文本中解析结构化事实三元组。
// 实际生产环境应使用 Extract 方法处理批量对话。
func (f *FactExtractor) ParseFacts(text string) ([]*entity.ExtractedFact, error) {
	if err := f.checkRateLimit(); err != nil {
		return nil, err
	}

	prompt := buildFactExtractionPrompt(text)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := f.llm.Chat(ctx, []string{prompt})
	if err != nil {
		return nil, fmt.Errorf("llm chat failed: %w", err)
	}

	return f.parseResponse(response)
}

// Extract 从多条原始对话中提取事实。
func (f *FactExtractor) Extract(ctx context.Context, dialogues []*entity.RawDialogue) ([]*entity.ExtractedFact, error) {
	if err := f.checkRateLimit(); err != nil {
		return nil, err
	}

	var contents []string
	for _, d := range dialogues {
		contents = append(contents, d.Content)
	}
	combined := strings.Join(contents, "\n")

	prompt := buildFactExtractionPrompt(combined)
	response, err := f.llm.Chat(ctx, []string{prompt})
	if err != nil {
		return nil, fmt.Errorf("llm extraction failed: %w", err)
	}

	facts, err := f.parseResponse(response)
	if err != nil {
		return nil, err
	}

	// 关联 source_msg_ids
	msgIDs := make([]string, len(dialogues))
	for i, d := range dialogues {
		msgIDs[i] = d.MessageID
	}
	for _, fact := range facts {
		fact.SourceMsgIDs = msgIDs
	}

	return facts, nil
}

func (f *FactExtractor) checkRateLimit() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-time.Minute)

	// 清理过期的调用记录
	var valid []time.Time
	for _, t := range f.callTimes {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	f.callTimes = valid

	if len(f.callTimes) >= f.rateLimit {
		return ErrRateLimited
	}

	f.callTimes = append(f.callTimes, now)
	return nil
}

func (f *FactExtractor) parseResponse(response string) ([]*entity.ExtractedFact, error) {
	response = strings.TrimSpace(response)
	if response == "" {
		return nil, fmt.Errorf("empty llm response")
	}

	// 尝试提取 JSON 数组（LLM 可能在 markdown 代码块中返回）
	if idx := strings.Index(response, "["); idx != -1 {
		if endIdx := strings.LastIndex(response, "]"); endIdx != -1 && endIdx > idx {
			response = response[idx : endIdx+1]
		}
	}

	var rawFacts []struct {
		Subject    string  `json:"subject"`
		Predicate  string  `json:"predicate"`
		Object     string  `json:"object"`
		Confidence float64 `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(response), &rawFacts); err != nil {
		return nil, fmt.Errorf("failed to parse fact json: %w", err)
	}

	seen := make(map[string]struct{}, len(rawFacts))
	var facts []*entity.ExtractedFact
	for _, rf := range rawFacts {
		subject := strings.TrimSpace(rf.Subject)
		predicate := strings.TrimSpace(rf.Predicate)
		object := strings.TrimSpace(rf.Object)
		if subject == "" || predicate == "" || object == "" {
			continue // 过滤不完整三元组
		}
		key := factTripleKey(subject, predicate, object)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if rf.Confidence < 0 || rf.Confidence > 1 {
			rf.Confidence = 0.5 // 默认置信度
		}
		facts = append(facts, entity.NewExtractedFact(subject, predicate, object, rf.Confidence, nil))
	}

	return facts, nil
}

func factTripleKey(subject, predicate, object string) string {
	parts := []string{subject, predicate, object}
	for i, part := range parts {
		parts[i] = strings.ToLower(strings.TrimSpace(part))
	}
	return strings.Join(parts, "\x00")
}

func buildFactExtractionPrompt(text string) string {
	return fmt.Sprintf(`从以下对话中提取结构化事实，以 JSON 数组格式返回。
每个事实包含 subject(主体)、predicate(谓词)、object(客体)、confidence(置信度 0-1)。
如果无法提取事实，返回空数组 []。

对话内容：
%s

要求：
1. subject 通常是"用户"或对话中提到的具体人物
2. predicate 是动作或状态，如"患有"、"服用"、"检查"
3. object 是具体的疾病、药品、症状等
4. confidence 基于信息明确程度打分

输出格式示例：
[{"subject":"用户","predicate":"患有","object":"偏头痛","confidence":0.9}]`, text)
}

// =============================================================================
// FactExtractorWorker 后台 Worker
// =============================================================================

// FactExtractorWorker 异步消费未处理对话并进行事实提取。
type FactExtractorWorker struct {
	extractor    *FactExtractor
	dialogueRepo repository.RawDialogueRepository
	factRepo     repository.FactRepository
	wg           sync.WaitGroup
	stopCh       chan struct{}
}

// NewFactExtractorWorker 构造函数。
func NewFactExtractorWorker(
	extractor *FactExtractor,
	dialogueRepo repository.RawDialogueRepository,
	factRepo repository.FactRepository,
) *FactExtractorWorker {
	return &FactExtractorWorker{
		extractor:    extractor,
		dialogueRepo: dialogueRepo,
		factRepo:     factRepo,
		stopCh:       make(chan struct{}),
	}
}

// Start 启动后台提取 goroutine。
func (w *FactExtractorWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.run(ctx)
}

// Wait 等待 worker 优雅停止。
func (w *FactExtractorWorker) Wait() {
	w.wg.Wait()
}

func (w *FactExtractorWorker) run(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *FactExtractorWorker) processBatch(ctx context.Context) {
	dialogues, err := w.dialogueRepo.GetUnprocessed(ctx, 5)
	if err != nil {
		return
	}
	if len(dialogues) == 0 {
		return
	}

	// 标记为处理中
	for _, d := range dialogues {
		_ = w.dialogueRepo.MarkProcessing(ctx, d.MessageID)
	}

	facts, err := w.extractor.Extract(ctx, dialogues)
	if err != nil {
		// 标记为失败
		for _, d := range dialogues {
			_ = w.dialogueRepo.MarkFailed(ctx, d.MessageID)
		}
		return
	}

	// 保存提取到的事实
	for _, f := range facts {
		if err := w.factRepo.Save(ctx, f); err != nil {
			// 单个事实保存失败不影响其他
			continue
		}
	}

	// 标记为已处理
	for _, d := range dialogues {
		_ = w.dialogueRepo.MarkProcessed(ctx, d.MessageID)
	}
}
