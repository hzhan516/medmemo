package onnx

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/knights-analytics/hugot/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewEngine_NoModel_ReturnsUnavailable 验证无模型时引擎降级为不可用，不 panic。
func TestNewEngine_NoModel_ReturnsUnavailable(t *testing.T) {
	engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
	require.NoError(t, err, "无模型时不应返回错误")
	assert.NotNil(t, engine)
	assert.False(t, engine.IsAvailable(), "模型缺失时引擎应标记为不可用")

	err = engine.Close()
	assert.NoError(t, err)
}

// TestNewEngine_NoLibrary_ReturnsUnavailable 验证无动态库时引擎降级为不可用。
func TestNewEngine_NoLibrary_ReturnsUnavailable(t *testing.T) {
	engine, err := NewEngine(EngineConfig{ResourceDir: "/nonexistent/resources", ModelPath: "resources/models/distilbert-ner"})
	require.NoError(t, err, "无动态库时不应返回错误")
	assert.NotNil(t, engine)
	assert.False(t, engine.IsAvailable(), "动态库缺失时引擎应标记为不可用")
	assert.False(t, engine.HasEmbeddingPipeline(), "动态库缺失时 embedding 引擎应不可用")
	assert.NotEmpty(t, engine.RuntimeLibPath())
	assert.Contains(t, engine.EmbeddingFailureReason(), "ONNX Runtime library not found")

	err = engine.Close()
	assert.NoError(t, err)
}

// TestEngine_Predict_WhenUnavailable_ReturnsError 验证不可用引擎调用 Predict 返回错误。
func TestEngine_Predict_WhenUnavailable_ReturnsError(t *testing.T) {
	engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
	require.NoError(t, err)
	defer engine.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	spans, err := engine.Predict(ctx, "测试文本")
	assert.Error(t, err)
	assert.Nil(t, spans)
	assert.Contains(t, err.Error(), "not available")
}

// TestNormalizeLabel 验证 BIO 标签归一化逻辑。
func TestNormalizeLabel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"B-PER", "PER"},
		{"I-PER", "PER"},
		{"B-ORG", "ORG"},
		{"I-LOC", "LOC"},
		{"PER", "PER"},
		{"O", "O"},
		{"B-DISEASE", "DISEASE"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeLabel(tt.input))
		})
	}
}

// TestPlatformLibPath 验证各平台动态库路径解析。
func TestPlatformLibPath(t *testing.T) {
	path, err := PlatformLibPath("/opt/medmemo")
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "/opt/medmemo/lib/")
}

// TestEntitySpan_Struct 验证 EntitySpan 数据结构。
func TestEntitySpan_Struct(t *testing.T) {
	span := EntitySpan{
		Text:  "张三",
		Label: "PER",
		Start: 0,
		End:   6,
		Score: 0.9876,
	}
	assert.Equal(t, "张三", span.Text)
	assert.Equal(t, "PER", span.Label)
	assert.Equal(t, 0, span.Start)
	assert.Equal(t, 6, span.End)
	assert.InDelta(t, float32(0.9876), span.Score, 0.0001)
}

// TestVerifyModelSHA256_Missing 验证无 .sha256 文件时跳过校验。
func TestVerifyModelSHA256_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	modelFile := filepath.Join(tmpDir, "model.onnx")
	require.NoError(t, os.WriteFile(modelFile, []byte("dummy"), 0644))

	err := verifyModelSHA256(modelFile)
	assert.NoError(t, err, "无 .sha256 文件时应跳过校验")
}

// TestVerifyModelSHA256_Mismatch 验证 SHA-256 不匹配时返回错误。
func TestVerifyModelSHA256_Mismatch(t *testing.T) {
	tmpDir := t.TempDir()
	modelFile := filepath.Join(tmpDir, "model.onnx")
	shaFile := modelFile + ".sha256"

	require.NoError(t, os.WriteFile(modelFile, []byte("dummy model data"), 0644))
	require.NoError(t, os.WriteFile(shaFile, []byte("0000000000000000000000000000000000000000000000000000000000000000"), 0644))

	err := verifyModelSHA256(modelFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mismatch")
}

// TestVerifyModelSHA256_Valid 验证正确的 SHA-256 通过校验。
func TestVerifyModelSHA256_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	modelFile := filepath.Join(tmpDir, "model.onnx")
	shaFile := modelFile + ".sha256"

	data := []byte("valid model content for sha256 test")
	require.NoError(t, os.WriteFile(modelFile, data, 0644))

	hash := sha256.Sum256(data)
	require.NoError(t, os.WriteFile(shaFile, []byte(hex.EncodeToString(hash[:])), 0644))

	err := verifyModelSHA256(modelFile)
	assert.NoError(t, err)
}

// TestDefaultModelPath 验证默认模型路径拼接。
func TestDefaultModelPath(t *testing.T) {
	path := DefaultModelPath("/opt/medmemo")
	assert.Equal(t, filepath.Join("/opt/medmemo", "models", "distilbert-ner"), path)
}

// TestNewEngine_ModelDirExistsButModelMissing 验证模型目录存在但 model.onnx 缺失时降级。
func TestNewEngine_ModelDirExistsButModelMissing(t *testing.T) {
	tmpDir := t.TempDir()
	// 创建虚拟动态库文件
	libDir := filepath.Join(tmpDir, "lib", "linux")
	require.NoError(t, os.MkdirAll(libDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(libDir, "libonnxruntime.so"), []byte("dummy"), 0644))

	// 创建模型目录（空）
	modelDir := filepath.Join(tmpDir, "models", "distilbert-ner")
	require.NoError(t, os.MkdirAll(modelDir, 0755))

	engine, err := NewEngine(EngineConfig{ResourceDir: tmpDir, ModelPath: modelDir})
	require.NoError(t, err)
	assert.NotNil(t, engine)
	assert.False(t, engine.IsAvailable(), "model.onnx 缺失时应降级")
	assert.NoError(t, engine.Close())
}

// TestNewEngine_SHA256Mismatch 验证 SHA-256 校验失败时降级。
func TestNewEngine_SHA256Mismatch(t *testing.T) {
	tmpDir := t.TempDir()
	// 创建虚拟动态库文件
	libDir := filepath.Join(tmpDir, "lib", "linux")
	require.NoError(t, os.MkdirAll(libDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(libDir, "libonnxruntime.so"), []byte("dummy"), 0644))

	// 创建模型目录 + model.onnx + 错误的 .sha256
	modelDir := filepath.Join(tmpDir, "models", "distilbert-ner")
	require.NoError(t, os.MkdirAll(modelDir, 0755))
	modelFile := filepath.Join(modelDir, "model.onnx")
	require.NoError(t, os.WriteFile(modelFile, []byte("dummy model"), 0644))
	require.NoError(t, os.WriteFile(modelFile+".sha256", []byte("0000000000000000000000000000000000000000000000000000000000000000"), 0644))

	engine, err := NewEngine(EngineConfig{ResourceDir: tmpDir, ModelPath: modelDir})
	require.NoError(t, err)
	assert.NotNil(t, engine)
	assert.False(t, engine.IsAvailable(), "SHA-256 不匹配时应降级")
	assert.NoError(t, engine.Close())
}

// TestVerifyModelSHA256_ModelFileMissing 验证 .sha256 存在但 model.onnx 不存在时返回错误。
func TestVerifyModelSHA256_ModelFileMissing(t *testing.T) {
	tmpDir := t.TempDir()
	modelFile := filepath.Join(tmpDir, "model.onnx")
	shaFile := modelFile + ".sha256"
	// 创建 .sha256 文件但不创建 model.onnx
	require.NoError(t, os.WriteFile(shaFile, []byte("dummyhash"), 0644))

	err := verifyModelSHA256(modelFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read model file failed")
}

// TestVerifyModelSHA256_HashFileReadError 验证 .sha256 文件存在但不可读时返回错误。
func TestVerifyModelSHA256_HashFileReadError(t *testing.T) {
	tmpDir := t.TempDir()
	modelFile := filepath.Join(tmpDir, "model.onnx")
	shaFile := modelFile + ".sha256"
	require.NoError(t, os.WriteFile(modelFile, []byte("dummy"), 0644))
	// 创建不可读的 .sha256 目录（导致 ReadFile 失败）
	require.NoError(t, os.Mkdir(shaFile, 0755))

	err := verifyModelSHA256(modelFile)
	require.Error(t, err)
}

// TestNewNERWorker 验证 Worker 构造函数。
func TestNewNERWorker(t *testing.T) {
	w := NewNERWorker(1, nil)
	assert.NotNil(t, w)
	assert.Equal(t, 1, w.id)
}

// TestEngine_Close_NilSession 验证无 session 时 Close 安全退出。
func TestEngine_Close_NilSession(t *testing.T) {
	// 使用无模型的引擎，session 为 nil
	engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
	require.NoError(t, err)
	assert.NoError(t, engine.Close())
}

// TestIsNERAvailable 验证 IsNERAvailable 与 IsAvailable 在 NER 状态变化时的返回值。
func TestIsNERAvailable(t *testing.T) {
	t.Run("pipeline非nil时IsNERAvailable返回true", func(t *testing.T) {
		engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
		require.NoError(t, err)
		defer engine.Close()

		engine.pipeline = &pipelines.TokenClassificationPipeline{}
		engine.nerAvailable = true
		assert.True(t, engine.IsNERAvailable())
	})

	t.Run("仅NER可用时IsAvailable为true且IsEmbeddingAvailable为false", func(t *testing.T) {
		engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
		require.NoError(t, err)
		defer engine.Close()

		engine.nerAvailable = true
		engine.pipeline = &pipelines.TokenClassificationPipeline{}
		engine.embeddingAvailable = false
		engine.embeddingPipeline = nil
		assert.True(t, engine.IsAvailable())
		assert.True(t, engine.IsNERAvailable())
		assert.False(t, engine.IsEmbeddingAvailable())
	})

	t.Run("NER与Embedding均不可用时IsAvailable返回false", func(t *testing.T) {
		engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
		require.NoError(t, err)
		defer engine.Close()

		engine.nerAvailable = false
		engine.embeddingAvailable = false
		engine.pipeline = nil
		engine.embeddingPipeline = nil
		assert.False(t, engine.IsAvailable())
		assert.False(t, engine.IsNERAvailable())
		assert.False(t, engine.IsEmbeddingAvailable())
	})
}

// TestIsEmbeddingAvailable 验证 IsEmbeddingAvailable 与 IsAvailable 在 Embedding 状态变化时的返回值。
func TestIsEmbeddingAvailable(t *testing.T) {
	t.Run("embeddingPipeline非nil且embeddingAvailable为true时返回true", func(t *testing.T) {
		engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
		require.NoError(t, err)
		defer engine.Close()

		engine.embeddingPipeline = &pipelines.FeatureExtractionPipeline{}
		engine.embeddingAvailable = true
		assert.True(t, engine.IsEmbeddingAvailable())
	})

	t.Run("仅Embedding可用时IsAvailable为true且IsNERAvailable为false", func(t *testing.T) {
		engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
		require.NoError(t, err)
		defer engine.Close()

		engine.embeddingAvailable = true
		engine.embeddingPipeline = &pipelines.FeatureExtractionPipeline{}
		engine.nerAvailable = false
		engine.pipeline = nil
		assert.True(t, engine.IsAvailable())
		assert.True(t, engine.IsEmbeddingAvailable())
		assert.False(t, engine.IsNERAvailable())
	})
}

// TestEngine_Embed_WhenUnavailable 验证 embeddingAvailable=false 时 Embed 返回 not available 错误。
func TestEngine_Embed_WhenUnavailable(t *testing.T) {
	engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
	require.NoError(t, err)
	defer engine.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	embeddings, err := engine.Embed(ctx, []string{"测试文本"})
	assert.Error(t, err)
	assert.Nil(t, embeddings)
	assert.Contains(t, err.Error(), "not available")
}

// TestWorkerLoop_ContextCancellation_NER 验证 NER worker 在 caller cancel 时不会阻塞在 resultCh 发送上。
func TestWorkerLoop_ContextCancellation_NER(t *testing.T) {
	// 构造一个无模型引擎，手动注入 pipeline 和 worker 以控制行为
	engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
	require.NoError(t, err)

	// 使用自定义 worker 避免调用真实 ONNX pipeline（零值 pipeline 会 panic）
	mockWorker := &mockNERWorker{delay: 50 * time.Millisecond}
	engine.nerAvailable = true
	engine.workers = []*NERWorker{}

	// 重新创建 taskCh 并启动 worker（原 taskCh 已关闭）
	engine.taskCh = make(chan nerTask, 16)
	engine.wg.Add(1)
	go mockWorker.run(engine)

	// 创建一个已取消的 context，且 resultCh 无缓冲、无接收方
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	resultCh := make(chan nerResult) // 无缓冲，无接收方

	// 发送任务：worker 处理完尝试向 resultCh 发送时，应通过 ctx.Done() 分支退出
	done := make(chan struct{})
	go func() {
		engine.taskCh <- nerTask{ctx: ctx, text: "测试", resultCh: resultCh}
		close(done)
	}()

	select {
	case <-done:
		// 任务成功入队（taskCh 有缓冲）
	case <-time.After(2 * time.Second):
		t.Fatal("向 taskCh 发送任务超时")
	}

	// worker 应在短时间内完成（不会永远阻塞在 resultCh 发送上）
	closed := make(chan struct{})
	go func() {
		engine.wg.Wait()
		close(closed)
	}()

	// 关闭 taskCh 让 worker 退出
	close(engine.taskCh)

	select {
	case <-closed:
		// worker 正常退出，说明没有 goroutine 泄漏
	case <-time.After(2 * time.Second):
		t.Fatal("worker goroutine 在 caller cancel 后泄漏（NER）")
	}
}

// mockNERWorker 模拟 NERWorker，用于测试 workerLoop 的 select 行为。
type mockNERWorker struct {
	delay time.Duration
}

func (w *mockNERWorker) run(e *Engine) {
	defer e.wg.Done()
	for task := range e.taskCh {
		if w.delay > 0 {
			time.Sleep(w.delay)
		}
		select {
		case task.resultCh <- nerResult{spans: []EntitySpan{{Text: "mock", Label: "TEST"}}, err: nil}:
		case <-task.ctx.Done():
			// caller cancelled, result discarded to prevent goroutine leak
		}
	}
}

// TestWorkerLoop_ContextCancellation_Embedding 验证 embedding worker 在 caller cancel 时不会阻塞。
func TestWorkerLoop_ContextCancellation_Embedding(t *testing.T) {
	engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
	require.NoError(t, err)

	mockWorker := &mockEmbeddingWorker{delay: 50 * time.Millisecond}
	engine.embeddingAvailable = true
	engine.embeddingWorkers = []*EmbeddingWorker{}

	engine.embeddingTaskCh = make(chan embeddingTask, 16)
	engine.embeddingWg.Add(1)
	go mockWorker.run(engine)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resultCh := make(chan embeddingResult) // 无缓冲，无接收方

	done := make(chan struct{})
	go func() {
		engine.embeddingTaskCh <- embeddingTask{ctx: ctx, texts: []string{"测试"}, resultCh: resultCh}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("向 embeddingTaskCh 发送任务超时")
	}

	closed := make(chan struct{})
	go func() {
		engine.embeddingWg.Wait()
		close(closed)
	}()

	close(engine.embeddingTaskCh)

	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("embedding worker goroutine 在 caller cancel 后泄漏")
	}
}

// mockEmbeddingWorker 模拟 EmbeddingWorker，用于测试 workerLoop 的 select 行为。
type mockEmbeddingWorker struct {
	delay time.Duration
}

func (w *mockEmbeddingWorker) run(e *Engine) {
	defer e.embeddingWg.Done()
	for task := range e.embeddingTaskCh {
		if w.delay > 0 {
			time.Sleep(w.delay)
		}
		select {
		case task.resultCh <- embeddingResult{embeddings: [][]float32{{0.1, 0.2}}, err: nil}:
		case <-task.ctx.Done():
			// caller cancelled, result discarded to prevent goroutine leak
		}
	}
}

// TestPredict_ContextCancellation_ReturnsError 验证 Predict 在 ctx 取消时返回 ctx.Err()。
func TestPredict_ContextCancellation_ReturnsError(t *testing.T) {
	engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
	require.NoError(t, err)
	defer engine.Close()

	// 模拟一个可用的引擎（无需真实模型）
	engine.nerAvailable = true
	engine.workers = nil // 不启动真实 worker

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	_, err = engine.Predict(ctx, "测试")
	assert.ErrorIs(t, err, context.Canceled)
}

// TestEmbed_ContextCancellation_ReturnsError 验证 Embed 在 ctx 取消时返回 ctx.Err()。
func TestEmbed_ContextCancellation_ReturnsError(t *testing.T) {
	engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
	require.NoError(t, err)
	defer engine.Close()

	engine.embeddingAvailable = true
	engine.embeddingWorkers = nil

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = engine.Embed(ctx, []string{"测试"})
	assert.ErrorIs(t, err, context.Canceled)
}

// TestWorkerLoop_SlowConsumer_NER 验证 NER worker 在慢消费者场景下不会泄漏。
func TestWorkerLoop_SlowConsumer_NER(t *testing.T) {
	engine, err := NewEngine(EngineConfig{ResourceDir: "resources", ModelPath: "resources/models/nonexistent"})
	require.NoError(t, err)

	mockWorker := &mockNERWorker{delay: 200 * time.Millisecond}
	engine.nerAvailable = true
	engine.workers = []*NERWorker{}
	engine.taskCh = make(chan nerTask, 16)
	engine.wg.Add(1)
	go mockWorker.run(engine)

	// 消费者故意延迟接收：ctx 50ms 超时，worker 处理需 200ms
	// 这样 ctx 会在 worker 完成前就已超时，确保 select 命中 ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	resultCh := make(chan nerResult) // 无缓冲，强制 select 竞争

	done := make(chan struct{})
	go func() {
		engine.taskCh <- nerTask{ctx: ctx, text: "测试", resultCh: resultCh}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("任务入队超时")
	}

	// 等待 worker 处理完（200ms）且 ctx 已超时（50ms）
	time.Sleep(300 * time.Millisecond)

	// worker 应该已经处理完任务，结果因 ctx 超时被丢弃
	select {
	case <-resultCh:
		t.Fatal("ctx 超时后不应收到结果")
	default:
		// 正确：结果已被丢弃
	}

	close(engine.taskCh)
	closed := make(chan struct{})
	go func() {
		engine.wg.Wait()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("worker 退出超时")
	}
}
