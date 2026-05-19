// Package onnx 封装 Hugot ONNX 推理运行时。
// ONNX Session 非线程安全，本包需实现 Worker 模型确保串行化调用。
package onnx

import (
	"context"
	"fmt"

	"github.com/google/wire"
)

// NERWorker ONNX NER 推理 Worker，持有独立 Session。
// 固定 2 个 Worker，每个 Session 内存开销约 80-100MB，总预算 200MB 以内。
type NERWorker struct {
	id      int
	session any // TODO(作者): 替换为 hugot 实际 Session 类型 [Issue#018]
}

// NewNERWorker 创建推理 Worker。
func NewNERWorker(id int, modelPath string) (*NERWorker, error) {
	// TODO(作者): 加载 int8 量化 DistilBERT-ONNX 模型 [Issue#026]
	return &NERWorker{id: id}, nil
}

// Predict 执行 NER 推理，必须在单 Worker 内串行调用。
func (w *NERWorker) Predict(ctx context.Context, text string) ([]EntitySpan, error) {
	// TODO(作者): 调用 Hugot Run 并解析 BIO 标签 [Issue#019]
	return nil, fmt.Errorf("NERWorker.Predict not implemented")
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
	workers []*NERWorker
	taskCh  chan nerTask
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

// NewEngine 创建推理引擎，初始化固定 Worker。
func NewEngine(modelPath string) (*Engine, error) {
	const workerCount = 2
	workers := make([]*NERWorker, workerCount)
	for i := 0; i < workerCount; i++ {
		w, err := NewNERWorker(i, modelPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create NER worker %d: %w", i, err)
		}
		workers[i] = w
	}
	return &Engine{
		workers: workers,
		taskCh:  make(chan nerTask, 16), // 有缓冲 channel，容量 16
	}, nil
}

// ONNXSet 供 Wire 使用的 ProviderSet。
var ONNXSet = wire.NewSet(
	NewEngine,
)
