// Package detector 实现敏感信息检测适配器，
// 将基础设施层的检测能力封装为 application/port 定义的接口。
package detector

import (
	"context"

	"github.com/google/wire"
	"github.com/hzhan516/medmemo/internal/infrastructure/onnx"
	"github.com/hzhan516/medmemo/pkg/models"
)

// defaultConfidenceThreshold 为 NER 实体置信度过滤阈值。
// DistilBERT 多语言模型在中文人名/地点上 F1 约 0.82–0.88，
// 0.75 可在召回与精确之间取得平衡，过滤大部分低置信度误报。
const defaultConfidenceThreshold float32 = 0.75

// strictConfidenceThreshold 为严格级专用的 NER 置信度阈值。
// 严格级以召回优先：将阈值降至 0.5 可捕获更多低置信度候选实体，
// 尽量减少出网 PII。代价是精确率下降、可能过度遮蔽（P3 不可逆），
// 并轻微影响 prompt 保真度；此权衡对严格级用户可接受，详见 docs/COMPLIANCE.md。
const strictConfidenceThreshold float32 = 0.5

// ONNXNERDetector 基于 ONNX NER 模型的敏感信息检测器。
// 将 hugot/DistilBERT 推理结果适配为 application/port.NERDetector 接口。
type ONNXNERDetector struct {
	engine    *onnx.Engine
	threshold float32
}

// NewONNXNERDetector 创建标准级 ONNX NER 检测器（默认阈值 0.75）。
func NewONNXNERDetector(engine *onnx.Engine) *ONNXNERDetector {
	return newONNXNERDetectorWithThreshold(engine, defaultConfidenceThreshold)
}

// newONNXNERDetectorWithThreshold 以指定置信度阈值创建检测器。
func newONNXNERDetectorWithThreshold(engine *onnx.Engine, threshold float32) *ONNXNERDetector {
	return &ONNXNERDetector{
		engine:    engine,
		threshold: threshold,
	}
}

// StrictONNXNERDetector 是严格级 NER 检测器，复用与标准级相同的 *onnx.Engine，
// 仅将置信度阈值降至 strictConfidenceThreshold 以提升召回。
// 复用同一引擎可避免重复加载模型/占用额外内存，且引擎内部通过 Worker Pool 串行化推理，
// 满足 ONNX Session.Run 非线程安全的约束。
type StrictONNXNERDetector struct {
	*ONNXNERDetector
}

// NewStrictONNXNERDetector 创建严格级 NER 检测器，复用传入的同一 *onnx.Engine。
func NewStrictONNXNERDetector(engine *onnx.Engine) *StrictONNXNERDetector {
	return &StrictONNXNERDetector{
		ONNXNERDetector: newONNXNERDetectorWithThreshold(engine, strictConfidenceThreshold),
	}
}

// Predict 执行 NER 推理并返回标准化实体列表。
// 引擎不可用或推理出错时降级返回空列表，不阻断调用方流水线。
func (d *ONNXNERDetector) Predict(ctx context.Context, text string) ([]models.SensitiveEntity, error) {
	if d.engine == nil || !d.engine.IsNERAvailable() {
		return nil, nil
	}

	spans, err := d.engine.Predict(ctx, text)
	if err != nil {
		// 降级：推理失败时不报错，返回空列表让上层继续执行
		return nil, nil
	}

	return filterSpansByThreshold(spans, d.threshold), nil
}

// filterSpansByThreshold 按置信度阈值过滤 NER 结果并归一化为敏感实体（纯函数）。
// 低于阈值或无法映射类型的实体被丢弃。
func filterSpansByThreshold(spans []onnx.EntitySpan, threshold float32) []models.SensitiveEntity {
	var entities []models.SensitiveEntity
	for _, span := range spans {
		if span.Score < threshold {
			continue
		}
		entityType := mapNERLabel(span.Label)
		if entityType == "" {
			continue
		}
		entities = append(entities, models.SensitiveEntity{
			Text:     span.Text,
			Type:     entityType,
			Level:    models.P3Confidential,
			StartPos: span.Start,
			EndPos:   span.End,
			Score:    span.Score,
		})
	}
	return entities
}

// IsAvailable 返回底层 ONNX NER 引擎是否已就绪。
func (d *ONNXNERDetector) IsAvailable() bool {
	return d.engine != nil && d.engine.IsNERAvailable()
}

// mapNERLabel 将 hugot BIO 标签归一化为中文实体类型。
// 当前支持 PER(人名)、LOC(地点)、ORG(机构名)；
// 其他标签（如 MISC）返回空字符串，表示不参与脱敏替换。
func mapNERLabel(label string) string {
	switch label {
	case "PER":
		return "姓名"
	case "LOC":
		return "地点"
	case "ORG":
		return "机构名"
	default:
		return ""
	}
}

// ONNXNERSet 供 Wire 使用的 ProviderSet。
var ONNXNERSet = wire.NewSet(
	NewONNXNERDetector,
	NewStrictONNXNERDetector,
)
