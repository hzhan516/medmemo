// Package desensitizer 提供公共的脱敏算法工具包。
// 不依赖任何 internal/ 子包，可被外部模块引用。
//
// 当前实现基于 regexp 顺序匹配，覆盖常见 PII 类型：
//   - 身份证号（15/18 位）
//   - 大陆手机号（11 位）
//   - 银行卡号（16-19 位）
//   - 邮箱地址
//   - URL
//
// 性能预算：单条文本 <1ms（regexp 实现，待后续升级为 Aho-Corasick）。
package desensitizer

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

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
	name        string
	pattern     *regexp.Regexp
	entityType  string
	level       models.SensitivityLevel
	placeholder string
}

// NewRuleEngine 初始化规则脱敏引擎，内置常用正则规则。
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules: []rule{
			{
				name:        "id_card",
				pattern:     regexp.MustCompile(`\b(\d{15}|\d{17}[\dXx])\b`),
				entityType:  "身份证号",
				level:       models.P3Confidential,
				placeholder: "ID_CARD",
			},
			{
				name:        "phone",
				pattern:     regexp.MustCompile(`(?:(?:\+?86)?1[3-9]\d{9}|\b1[3-9]\d{9}\b)`),
				entityType:  "手机号",
				level:       models.P3Confidential,
				placeholder: "PHONE",
			},
			{
				name:        "bank_card",
				pattern:     regexp.MustCompile(`\b(\d{16}|\d{17}|\d{18}|\d{19})\b`),
				entityType:  "银行卡号",
				level:       models.P3Confidential,
				placeholder: "BANK_CARD",
			},
			{
				name:        "email",
				pattern:     regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
				entityType:  "邮箱",
				level:       models.P2Internal,
				placeholder: "EMAIL",
			},
			{
				name:        "url",
				pattern:     regexp.MustCompile(`https?://[^\s<>"{}|\^\[\]]+`),
				entityType:  "URL",
				level:       models.P2Internal,
				placeholder: "URL",
			},
		},
	}
}

// Process 执行规则匹配与替换。
// 顺序遍历规则，对每个匹配项生成占位符并记录映射关系。
func (e *RuleEngine) Process(text string) (models.DeidentifyResult, error) {
	result := models.DeidentifyResult{
		OriginalText: text,
		SafeText:     text,
		Entities:     make([]models.SensitiveEntity, 0),
		Placeholder:  make(map[string]string),
	}

	// 由于多个规则可能重叠，采用逐规则处理、维护偏移量的策略
	offset := 0
	for _, r := range e.rules {
		matches := r.pattern.FindAllStringIndex(result.SafeText, -1)
		if matches == nil {
			continue
		}

		// 从后向前替换，避免偏移量混乱
		for i := len(matches) - 1; i >= 0; i-- {
			m := matches[i]
			matchedText := result.SafeText[m[0]:m[1]]

			// 生成唯一占位符：{{TYPE_HASH}}
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(matchedText+fmt.Sprintf("_%d_%d", m[0], m[1]))))[:8]
			placeholder := fmt.Sprintf("{{%s_%s}}", r.placeholder, hash)

			// 记录实体信息
			entity := models.SensitiveEntity{
				Text:     matchedText,
				Type:     r.entityType,
				Level:    r.level,
				StartPos: m[0] + offset,
				EndPos:   m[1] + offset,
			}
			result.Entities = append(result.Entities, entity)

			// P2 级记录占位符映射（可逆）；P3 级不记录（不可逆）
			if r.level == models.P2Internal {
				result.Placeholder[placeholder] = matchedText
			}

			// 执行替换
			result.SafeText = result.SafeText[:m[0]] + placeholder + result.SafeText[m[1]:]
		}
	}

	return result, nil
}

// Restore 将脱敏后的文本还原为原始文本（仅支持 P2 级占位符）。
func Restore(deid models.DeidentifyResult) string {
	text := deid.SafeText
	for placeholder, original := range deid.Placeholder {
		text = strings.ReplaceAll(text, placeholder, original)
	}
	return text
}
