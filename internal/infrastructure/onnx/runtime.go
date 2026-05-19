// Package onnx 封装 Hugot ONNX 推理运行时。
// ONNX Session 非线程安全，本包通过 Worker 模型确保串行化调用。
package onnx

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/google/wire"
	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/options"
	"github.com/knights-analytics/hugot/pipelines"
)

// NERWorker ONNX NER 推理 Worker，通过 sync.Mutex 保护 pipeline.Run 的串行调用。
// hugot ORT Session 为全局单例，2 个 Worker 共享同一个 Pipeline，各自持有独立锁。
type NERWorker struct {
	id       int
	pipeline *pipelines.TokenClassificationPipeline
	mu       sync.Mutex
}

// NewNERWorker 创建推理 Worker。
func NewNERWorker(id int, pipeline *pipelines.TokenClassificationPipeline) *NERWorker {
	return &NERWorker{id: id, pipeline: pipeline}
}

// Predict 执行 NER 推理，必须在单 Worker 内串行调用。
func (w *NERWorker) Predict(ctx context.Context, text string) ([]EntitySpan, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	result, err := w.pipeline.RunPipeline(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("NER pipeline run failed: %w", err)
	}

	var spans []EntitySpan
	if len(result.Entities) > 0 {
		for _, e := range result.Entities[0] {
			spans = append(spans, EntitySpan{
				Text:  e.Word,
				Label: normalizeLabel(e.Entity),
				Start: int(e.Start),
				End:   int(e.End),
			})
		}
	}
	return spans, nil
}

// normalizeLabel 将 hugot BIO 标签（如 B-PER、I-PER）归一化为实体类型（PER）。
func normalizeLabel(entity string) string {
	if len(entity) > 2 && (entity[0] == 'B' || entity[0] == 'I') && entity[1] == '-' {
		return entity[2:]
	}
	return entity
}

// EntitySpan 表示识别到的实体区间。
type EntitySpan struct {
	Text  string
	Label string // PER, ORG, DISEASE, DRUG 等
	Start int
	End   int
}

// Engine ONNX 推理引擎，管理 Worker Pool 和任务分发。
type Engine struct {
	workers   []*NERWorker
	taskCh    chan nerTask
	session   *hugot.Session
	pipeline  *pipelines.TokenClassificationPipeline
	wg        sync.WaitGroup
	modelPath string
	libPath   string
	available bool
}

type nerTask struct {
	ctx      context.Context
	text     string
	resultCh chan nerResult
}

type nerResult struct {
	spans []EntitySpan
	err   error
}

// EngineConfig 是 Engine 的构造参数，避免 Wire 无法区分多个 string 参数。
type EngineConfig struct {
	ResourceDir string
	ModelPath   string
}

// NewEngine 创建推理引擎。
// 若动态库缺失或模型路径无效，返回 available=false 的引擎，不阻塞应用启动。
func NewEngine(cfg EngineConfig) (*Engine, error) {
	e := &Engine{
		taskCh:    make(chan nerTask, 16),
		modelPath: cfg.ModelPath,
	}

	libPath, err := PlatformLibPath(cfg.ResourceDir)
	if err != nil {
		return e, nil // 降级：不支持的平台，不报错
	}
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		return e, nil // 降级：动态库不存在，不报错
	}
	e.libPath = libPath

	// 检查模型目录是否存在（包含 model.onnx 和 tokenizer.json）
	if _, err := os.Stat(cfg.ModelPath); os.IsNotExist(err) {
		return e, nil // 降级：模型未下载，不报错
	}

	// 初始化 hugot ORT Session（全局单例）
	ctx := context.Background()
	session, err := hugot.NewORTSession(ctx, options.WithOnnxLibraryPath(libPath))
	if err != nil {
		return e, nil // 降级：Session 初始化失败，不报错
	}
	e.session = session

	// 创建 TokenClassificationPipeline
	config := hugot.TokenClassificationConfig{
		ModelPath:    cfg.ModelPath,
		OnnxFilename: "model.onnx",
		Name:         "ner-default",
		Options: []hugot.TokenClassificationOption{
			pipelines.WithSimpleAggregation(),
		},
	}
	pipeline, err := hugot.NewPipeline[*pipelines.TokenClassificationPipeline](session, config)
	if err != nil {
		_ = session.Destroy()
		e.session = nil
		return e, nil // 降级：Pipeline 创建失败，不报错
	}
	e.pipeline = pipeline

	// 创建 2 个 Worker
	const workerCount = 2
	e.workers = make([]*NERWorker, workerCount)
	for i := 0; i < workerCount; i++ {
		e.workers[i] = NewNERWorker(i, pipeline)
	}

	// 启动 Worker goroutine
	for i := 0; i < workerCount; i++ {
		e.wg.Add(1)
		go e.workerLoop(e.workers[i])
	}

	e.available = true
	return e, nil
}

func (e *Engine) workerLoop(worker *NERWorker) {
	defer e.wg.Done()
	for task := range e.taskCh {
		spans, err := worker.Predict(task.ctx, task.text)
		task.resultCh <- nerResult{spans: spans, err: err}
	}
}

// Predict 向 Worker Pool 提交推理任务，等待结果返回。
func (e *Engine) Predict(ctx context.Context, text string) ([]EntitySpan, error) {
	if !e.available {
		return nil, fmt.Errorf("ONNX engine not available")
	}
	resultCh := make(chan nerResult, 1)
	select {
	case e.taskCh <- nerTask{ctx: ctx, text: text, resultCh: resultCh}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	select {
	case res := <-resultCh:
		return res.spans, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// IsAvailable 返回引擎是否已就绪（动态库、模型、Session、Pipeline 均初始化成功）。
func (e *Engine) IsAvailable() bool {
	return e.available
}

// Close 优雅关闭引擎：停止 Worker、关闭 Session 释放 ONNX 资源。
func (e *Engine) Close() error {
	if e.taskCh != nil {
		close(e.taskCh)
	}
	e.wg.Wait()
	if e.session != nil {
		return e.session.Destroy()
	}
	return nil
}

// ONNXSet 供 Wire 使用的 ProviderSet。
var ONNXSet = wire.NewSet(
	NewEngine,
)
