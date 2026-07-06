// Package detector 实现敏感信息检测适配器，
// 将基础设施层的检测能力封装为 application/port 定义的接口。
package detector

import (
	"context"
	"fmt"

	"github.com/google/wire"
	"github.com/hzhan516/medmemo/pkg/desensitizer"
	"github.com/hzhan516/medmemo/pkg/models"
)

// RuleDetector 基于规则的敏感信息检测器。
// 当前接入 pkg/desensitizer L1 规则引擎，L2 NER 模型待后续引入 [Issue#030]。
type RuleDetector struct {
	engine *desensitizer.RuleEngine
}

// NewRuleDetector 创建规则检测器。
func NewRuleDetector() *RuleDetector {
	return &RuleDetector{engine: desensitizer.NewRuleEngine()}
}

// Detect 检测文本中的敏感实体，返回分级标记结果。
func (d *RuleDetector) Detect(_ context.Context, text string) ([]models.SensitiveEntity, error) {
	result, err := d.engine.Process(text)
	if err != nil {
		return nil, fmt.Errorf("rule detection failed: %w", err)
	}
	return result.Entities, nil
}

// ProviderSet 供 Wire 使用的 ProviderSet。
var ProviderSet = wire.NewSet(
	NewRuleDetector,
)
