// Package detector 实现敏感信息检测适配器，
// 将基础设施层的检测能力封装为 application/port 定义的接口。
package detector

import (
	"context"

	"github.com/google/wire"
	"github.com/medmemo/medmemo/pkg/models"
)

// RuleDetector 基于规则的敏感信息检测器（占位实现）。
// TODO(作者): 接入 pkg/desensitizer 规则引擎与 ONNX NER 模型 [Issue#030]
type RuleDetector struct{}

// NewRuleDetector 创建规则检测器。
func NewRuleDetector() *RuleDetector {
	return &RuleDetector{}
}

// Detect 检测文本中的敏感实体，当前返回空结果。
func (d *RuleDetector) Detect(ctx context.Context, text string) ([]models.SensitiveEntity, error) {
	return []models.SensitiveEntity{}, nil
}

// ProviderSet 供 Wire 使用的 ProviderSet。
var ProviderSet = wire.NewSet(
	NewRuleDetector,
)
