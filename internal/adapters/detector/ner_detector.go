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

// ONNXNERDetector 基于 ONNX NER 模型的敏感信息检测器。
// 将 hugot/DistilBERT 推理结果适配为 application/port.NERDetector 接口。
type ONNXNERDetector struct {
	engine    *onnx.Engine
	threshold float32
}

// NewONNXNERDetector 创建 ONNX NER 检测器。
func NewONNXNERDetector(engine *onnx.Engine) *ONNXNERDetector {
	return &ONNXNERDetector{
		engine:    engine,
		threshold: defaultConfidenceThreshold,
	}
}

// Predict 执行 NER 推理并返回标准化实体列表。
// 引擎不可用或推理出错时降级返回空列表，不阻断调用方流水线。
func (d *ONNXNERDetector) Predict(ctx context.Context, text string) ([]models.SensitiveEntity, error) {
	if d.engine == nil || !d.engine.IsAvailable() {
		return nil, nil
	}

	spans, err := d.engine.Predict(ctx, text)
	if err != nil {
		// 降级：推理失败时不报错，返回空列表让上层继续执行
		return nil, nil
	}

	var entities []models.SensitiveEntity
	for _, span := range spans {
		if span.Score < d.threshold {
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
	return entities, nil
}

// IsAvailable 返回底层 ONNX 引擎是否已就绪。
func (d *ONNXNERDetector) IsAvailable() bool {
	return d.engine != nil && d.engine.IsAvailable()
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
)
