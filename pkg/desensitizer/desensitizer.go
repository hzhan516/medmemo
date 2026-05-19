// Package desensitizer 提供公共的脱敏算法工具包。
// 不依赖任何 internal/ 子包，可被外部模块引用。
package desensitizer

import (
	"fmt"
	"regexp"

	"github.com/medmemo/medmemo/pkg/models"
)

// Stage 定义脱敏流水线中的单个处理阶段接口。
type Stage interface {
	Process(text string) (models.DeidentifyResult, error)
}

// Pipeline 串联多个脱敏阶段，按顺序执行。
type Pipeline struct {
	stages []Stage
}

// NewPipeline 创建脱敏流水线。
func NewPipeline(stages ...Stage) *Pipeline {
	return &Pipeline{stages: stages}
}

// Execute 依次执行各脱敏阶段，任一阶段出错即短路返回。
func (p *Pipeline) Execute(text string) (models.DeidentifyResult, error) {
	result := models.DeidentifyResult{OriginalText: text, SafeText: text}
	for _, stage := range p.stages {
		r, err := stage.Process(result.SafeText)
		if err != nil {
			return models.DeidentifyResult{}, fmt.Errorf("deidentify stage %T failed: %w", stage, err)
		}
		result = r
	}
	return result, nil
}

// RuleEngine 基于正则规则的一级脱敏引擎（L1）。
// 延迟 <1ms，覆盖：身份证、手机号、银行卡、邮箱、URL。
type RuleEngine struct {
	rules []rule
}

type rule struct {
	pattern     *regexp.Regexp
	entityType  string
	level       models.SensitivityLevel
	placeholder string
}

// NewRuleEngine 初始化规则脱敏引擎，内置常用正则规则。
func NewRuleEngine() *RuleEngine {
	// TODO(作者): 补充完整的正则规则与占位符映射 [Issue#001]
	return &RuleEngine{}
}

// Process 执行规则匹配与替换。
func (e *RuleEngine) Process(text string) (models.DeidentifyResult, error) {
	// TODO(作者): 实现 Aho-Corasick 多模式匹配或 regexp 批量替换 [Issue#029]
	return models.DeidentifyResult{OriginalText: text, SafeText: text}, nil
}
