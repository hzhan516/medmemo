// Package onnx 封装 Hugot ONNX 推理运行时。
// ONNX Session 非线程安全，本包通过 Worker 模型确保串行化调用。
package onnx

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

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
				Score: float32(e.Score),
			})
		}
	}
	return spans, nil
}

// verifyModelSHA256 对 model.onnx 执行可选的 SHA-256 完整性校验。
// 若同目录下存在 .sha256 文件则读取并比对；不存在则跳过校验。
func verifyModelSHA256(modelFile string) error {
	sumFile := modelFile + ".sha256"
	expected, err := os.ReadFile(sumFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 无校验文件，跳过
		}
		return fmt.Errorf("read sha256 file failed: %w", err)
	}

	data, err := os.ReadFile(modelFile)
	if err != nil {
		return fmt.Errorf("read model file failed: %w", err)
	}

	hash := sha256.Sum256(data)
	actual := hex.EncodeToString(hash[:])

	// .sha256 文件格式可能是 "<hash>  <filename>" 或纯 hash
	expectedStr := string(expected)
	if idx := len(actual); idx <= len(expectedStr) {
		expectedStr = expectedStr[:idx]
	}
	if actual != expectedStr {
		return fmt.Errorf("SHA-256 mismatch: expected %s, got %s", expectedStr, actual)
	}
	return nil
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
	Score float32 // 置信度，范围 [0, 1]
}

// Engine ONNX 推理引擎，管理 NER 和 Embedding 两个 Worker Pool。
type Engine struct {
	workers      []*NERWorker
	taskCh       chan nerTask
	session      *hugot.Session
	pipeline     *pipelines.TokenClassificationPipeline
	wg           sync.WaitGroup
	modelPath    string
	libPath      string
	nerAvailable bool

	// 嵌入推理相关
	embeddingPipeline  *pipelines.FeatureExtractionPipeline
	embeddingWorkers   []*EmbeddingWorker
	embeddingTaskCh    chan embeddingTask
	embeddingWg        sync.WaitGroup
	embeddingAvailable bool
	embeddingFailure   string
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

type embeddingTask struct {
	ctx      context.Context
	texts    []string
	resultCh chan embeddingResult
}

type embeddingResult struct {
	embeddings [][]float32
	err        error
}

// EngineConfig 是 Engine 的构造参数，避免 Wire 无法区分多个 string 参数。
type EngineConfig struct {
	ResourceDir        string
	ModelPath          string // NER 模型路径
	EmbeddingModelPath string // 嵌入模型路径（可选）
}

// NewEngine 创建推理引擎。
// NER 和 Embedding 独立初始化，任一缺失不影响另一个。
// 若动态库缺失则两者都不可用，不阻塞应用启动。
func NewEngine(cfg EngineConfig) (*Engine, error) {
	e := &Engine{
		taskCh:          make(chan nerTask, 16),
		embeddingTaskCh: make(chan embeddingTask, 16),
		modelPath:       cfg.ModelPath,
	}

	// 诊断日志：输出关键路径信息，一次运行即可定位失败层级
	fmt.Printf("[ONNX Engine] GOOS=%s GOARCH=%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("[ONNX Engine] ResourceDir=%s ModelPath=%s EmbeddingModelPath=%s\n",
		cfg.ResourceDir, cfg.ModelPath, cfg.EmbeddingModelPath)

	libPath, err := PlatformLibPath(cfg.ResourceDir)
	if err != nil {
		fmt.Printf("[ONNX Engine] PlatformLibPath failed: %v\n", err)
		e.embeddingFailure = fmt.Sprintf("platform library path failed: %v", err)
		return e, nil // 降级：不支持的平台
	}
	e.libPath = libPath
	fmt.Printf("[ONNX Engine] Looking for ONNX lib at: %s\n", libPath)
	info, err := os.Stat(libPath)
	if err != nil {
		fmt.Printf("[ONNX Engine] ONNX lib not found: %v\n", err)
		// fallback: 尝试 .so.1（PlatformLibPath 已处理，这里额外兜底）
		if strings.HasSuffix(libPath, ".so") {
			fallback := libPath + ".1"
			fmt.Printf("[ONNX Engine] Trying fallback: %s\n", fallback)
			if _, err2 := os.Stat(fallback); err2 == nil {
				libPath = fallback
				fmt.Printf("[ONNX Engine] Fallback found, using: %s\n", libPath)
				goto libFound
			}
		}
		// AppImage fallback: exeDir/../share/resources/
		if exe, err := os.Executable(); err == nil {
			exeDir := filepath.Dir(exe)
			appImageRes := filepath.Join(exeDir, "..", "share", "resources")
			fallbackLib, _ := PlatformLibPath(appImageRes)
			fmt.Printf("[ONNX Engine] Trying AppImage fallback: %s\n", fallbackLib)
			if _, err2 := os.Stat(fallbackLib); err2 == nil {
				libPath = fallbackLib
				fmt.Printf("[ONNX Engine] AppImage fallback found, using: %s\n", libPath)
				goto libFound
			}
		}
		fmt.Printf("[ONNX Engine] ONNX lib not found anywhere, engine unavailable\n")
		e.embeddingFailure = fmt.Sprintf("ONNX Runtime library not found: %s: %v", libPath, err)
		return e, nil // 降级：动态库不存在
	}
	fmt.Printf("[ONNX Engine] ONNX lib found: size=%d mode=%s isSymlink=%v\n",
		info.Size(), info.Mode(), info.Mode()&os.ModeSymlink != 0)
libFound:
	e.libPath = libPath

	// 初始化 hugot ORT Session（全局单例，NER 和 Embedding 共享）
	// WithOnnxLibraryPath 期望目录路径，不是文件路径
	ctx := context.Background()
	libDir := filepath.Dir(libPath)
	session, err := hugot.NewORTSession(ctx, options.WithOnnxLibraryPath(libDir))
	if err != nil {
		fmt.Printf("[ONNX Engine] Session init failed: %v\n", err)
		e.embeddingFailure = fmt.Sprintf("ONNX Runtime session init failed: %v", err)
		return e, nil // 降级：Session 初始化失败
	}
	fmt.Printf("[ONNX Engine] ORT Session created successfully\n")
	e.session = session

	// 独立初始化 NER Pipeline（可选，失败不阻塞 Embedding）
	e.initNERPipeline(cfg.ModelPath)

	// 独立初始化 Embedding Pipeline（可选，失败不阻塞 NER）
	if cfg.EmbeddingModelPath != "" {
		e.initEmbeddingPipeline(cfg.EmbeddingModelPath)
	} else {
		e.embeddingFailure = "embedding model path not configured"
	}

	// NER 和 Embedding 各自独立标记可用性，避免混为一谈导致 Predict/Embed 调用死等
	e.nerAvailable = e.pipeline != nil
	if !e.nerAvailable && !e.embeddingAvailable && e.session != nil {
		// NER 和 Embedding 都未就绪，释放 Session
		_ = e.session.Destroy()
		e.session = nil
	}

	// 最终状态校验：防止 pipeline 创建成功但状态标记未同步的 bug
	if e.embeddingPipeline != nil && !e.embeddingAvailable {
		fmt.Printf("[ONNX Engine] BUG: embeddingPipeline set but embeddingAvailable=false, fixing\n")
		e.embeddingAvailable = true
	}
	if e.pipeline != nil && !e.nerAvailable {
		fmt.Printf("[ONNX Engine] BUG: pipeline set but nerAvailable=false, fixing\n")
		e.nerAvailable = true
	}

	fmt.Printf("[ONNX Engine] Final state: nerAvailable=%v embeddingAvailable=%v\n",
		e.nerAvailable, e.embeddingAvailable)
	return e, nil
}

// initNERPipeline 初始化 NER 推理 Pipeline（内部方法，失败不返回错误）。
func (e *Engine) initNERPipeline(modelPath string) {
	fmt.Printf("[ONNX Engine] initNERPipeline: modelPath=%s session=%v\n",
		modelPath, e.session != nil)
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		fmt.Printf("[ONNX Engine] NER model dir not found: %s\n", modelPath)
		return // NER 模型未下载
	}

	modelFile := filepath.Join(modelPath, "model.onnx")
	if err := verifyModelSHA256(modelFile); err != nil {
		fmt.Printf("[ONNX Engine] NER model SHA256 verify failed: %v\n", err)
		return // 校验失败
	}

	config := hugot.TokenClassificationConfig{
		ModelPath:    modelPath,
		OnnxFilename: "model.onnx",
		Name:         "ner-default",
		Options: []hugot.TokenClassificationOption{
			pipelines.WithSimpleAggregation(),
		},
	}
	pipeline, err := hugot.NewPipeline[*pipelines.TokenClassificationPipeline](e.session, config)
	if err != nil {
		fmt.Printf("[ONNX Engine] NER pipeline creation failed: %v\n", err)
		return // NER Pipeline 创建失败
	}
	e.pipeline = pipeline

	// 创建并启动 2 个 NER Worker
	const workerCount = 2
	e.workers = make([]*NERWorker, workerCount)
	for i := 0; i < workerCount; i++ {
		e.workers[i] = NewNERWorker(i, pipeline)
	}
	for i := 0; i < workerCount; i++ {
		e.wg.Add(1)
		go e.workerLoop(e.workers[i])
	}
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
	if !e.nerAvailable {
		return nil, fmt.Errorf("NER pipeline not available")
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

// IsAvailable 返回引擎整体是否已就绪（NER 或 Embedding 任一可用即视为就绪）。
// 供上层做粗略状态判断；调用 Predict/Embed 前建议使用更精确的 IsNERAvailable/IsEmbeddingAvailable。
func (e *Engine) IsAvailable() bool {
	return e.nerAvailable || e.embeddingAvailable
}

// IsNERAvailable 返回 NER Pipeline 是否已初始化成功。
func (e *Engine) IsNERAvailable() bool {
	return e.nerAvailable
}

// IsEmbeddingAvailable 返回嵌入 Pipeline 是否已初始化成功。
func (e *Engine) IsEmbeddingAvailable() bool {
	return e.embeddingAvailable
}

// HasEmbeddingPipeline 返回嵌入 Pipeline 是否已初始化。
func (e *Engine) HasEmbeddingPipeline() bool {
	return e.embeddingAvailable
}

// EmbeddingFailureReason 返回嵌入引擎不可用的最近原因。
func (e *Engine) EmbeddingFailureReason() string {
	return e.embeddingFailure
}

// RuntimeLibPath 返回当前解析到的 ONNX Runtime 动态库路径。
func (e *Engine) RuntimeLibPath() string {
	return e.libPath
}

// Embed 向 Embedding Worker Pool 提交嵌入推理任务。
func (e *Engine) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if !e.embeddingAvailable {
		return nil, fmt.Errorf("embedding pipeline not available")
	}
	resultCh := make(chan embeddingResult, 1)
	select {
	case e.embeddingTaskCh <- embeddingTask{ctx: ctx, texts: texts, resultCh: resultCh}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	select {
	case res := <-resultCh:
		return res.embeddings, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// initEmbeddingPipeline 初始化嵌入 Pipeline（内部方法，失败不返回错误）。
func (e *Engine) initEmbeddingPipeline(modelPath string) {
	fmt.Printf("[ONNX Engine] initEmbeddingPipeline: modelPath=%s session=%v\n",
		modelPath, e.session != nil)
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		fmt.Printf("[ONNX Engine] Embedding model dir not found: %s\n", modelPath)
		e.embeddingFailure = fmt.Sprintf("embedding model dir not found: %s", modelPath)
		return // 模型未下载
	}

	// 列出目录内容，便于诊断
	entries, _ := os.ReadDir(modelPath)
	for _, entry := range entries {
		info, _ := entry.Info()
		fmt.Printf("[ONNX Engine]   modelDir entry: %s size=%d\n", entry.Name(), info.Size())
	}

	modelFile := filepath.Join(modelPath, "model.onnx")
	if err := verifyModelSHA256(modelFile); err != nil {
		fmt.Printf("[ONNX Engine] Embedding model SHA256 verify failed: %v\n", err)
		e.embeddingFailure = fmt.Sprintf("embedding model SHA256 verify failed: %v", err)
		return // 校验失败
	}

	config := hugot.FeatureExtractionConfig{
		ModelPath:    modelPath,
		OnnxFilename: "model.onnx",
		Name:         "embedding-default",
		Options: []hugot.FeatureExtractionOption{
			pipelines.WithNormalization(),
		},
	}
	pipeline, err := hugot.NewPipeline[*pipelines.FeatureExtractionPipeline](e.session, config)
	if err != nil {
		fmt.Printf("[ONNX Engine] Embedding pipeline creation failed: %v\n", err)
		fmt.Printf("[ONNX Engine]   session=%v modelPath=%s onnxFile=%s\n",
			e.session != nil, modelPath, config.OnnxFilename)
		e.embeddingFailure = fmt.Sprintf("embedding pipeline creation failed: %v", err)
		return // Pipeline 创建失败
	}

	// 预热推理：ONNX Runtime 首次推理时执行图优化（JIT），耗时可达 30-60 秒。
	// 在初始化阶段提前完成，避免用户首次提问时触发 context deadline exceeded。
	fmt.Printf("[ONNX Engine] Warming up embedding pipeline...\n")
	warmupStart := time.Now()
	warmupCtx, warmupCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	_, warmupErr := pipeline.RunPipeline(warmupCtx, []string{"warmup"})
	warmupCancel()
	if warmupErr != nil {
		fmt.Printf("[ONNX Engine] Embedding warmup failed: %v\n", warmupErr)
		e.embeddingFailure = fmt.Sprintf("embedding warmup failed: %v", warmupErr)
		return
	} else {
		fmt.Printf("[ONNX Engine] Embedding warmup completed in %v\n", time.Since(warmupStart))
	}
	e.embeddingPipeline = pipeline

	// 创建 2 个 Embedding Worker
	const embWorkerCount = 2
	e.embeddingWorkers = make([]*EmbeddingWorker, embWorkerCount)
	for i := 0; i < embWorkerCount; i++ {
		e.embeddingWorkers[i] = NewEmbeddingWorker(i, pipeline)
	}

	// 启动 Embedding Worker goroutine
	for i := 0; i < embWorkerCount; i++ {
		e.embeddingWg.Add(1)
		go e.embeddingWorkerLoop(e.embeddingWorkers[i])
	}

	e.embeddingAvailable = true
	e.embeddingFailure = ""
}

func (e *Engine) embeddingWorkerLoop(worker *EmbeddingWorker) {
	defer e.embeddingWg.Done()
	for task := range e.embeddingTaskCh {
		embeddings, err := worker.Embed(task.ctx, task.texts)
		task.resultCh <- embeddingResult{embeddings: embeddings, err: err}
	}
}

// Close 优雅关闭引擎：停止 Worker、关闭 Session 释放 ONNX 资源。
func (e *Engine) Close() error {
	if e.taskCh != nil {
		close(e.taskCh)
	}
	e.wg.Wait()

	if e.embeddingTaskCh != nil {
		close(e.embeddingTaskCh)
	}
	e.embeddingWg.Wait()

	if e.session != nil {
		return e.session.Destroy()
	}
	return nil
}

// ONNXSet 供 Wire 使用的 ProviderSet。
var ONNXSet = wire.NewSet(
	NewEngine,
)
