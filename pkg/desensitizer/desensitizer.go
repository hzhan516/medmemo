// Package desensitizer 提供公共的脱敏算法工具包。
// 不依赖任何 internal/ 子包，可被外部模块引用。
//
// L1 规则引擎采用 Aho-Corasick 预筛选 + Regexp 精确验证的混合架构：
//   - AC 自动机扫描特征关键词（如 @、http://），快速激活相关规则
//   - 数字序列扫描为无特征关键词的规则（身份证、银行卡）做二次激活
//   - 仅在激活的规则上执行 regexp，大幅减少全文本扫描次数
//
// 覆盖实体类型：
//   - 身份证号（15/18 位）
//   - 大陆手机号（11 位）
//   - 银行卡号（16-19 位）
//   - 邮箱地址
//   - URL
//
// 性能预算：单条文本 <1ms（AC 预筛选 + regexp 精确验证）。
package desensitizer

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/hzhan516/medmemo/pkg/models"
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

// RuleEngine 基于规则的一级脱敏引擎（L1）。
// 采用 Aho-Corasick 多模式预筛选 + Regexp 精确验证的混合架构，
// 时间复杂度接近 O(n)，覆盖：身份证、手机号、银行卡、邮箱、URL。
type RuleEngine struct {
	rules []compiledRule
	ac    *AhoCorasick
}

// matchInfo 记录一次 regexp 匹配的结果。
type matchInfo struct {
	rule  compiledRule
	start int
	end   int
	text  string
}

// NewRuleEngine 初始化规则脱敏引擎，加载外置规则并构建 AC 自动机。
// 规则加载失败时降级返回空引擎（所有文本直接放行，不阻断业务）。
func NewRuleEngine() *RuleEngine {
	rules, err := loadDefaultRules()
	if err != nil {
		// 降级：规则加载失败时返回空规则引擎，确保调用方不 panic
		return &RuleEngine{rules: []compiledRule{}}
	}

	var keywords []string
	for _, r := range rules {
		keywords = append(keywords, r.config.Keywords...)
	}
	var ac *AhoCorasick
	if len(keywords) > 0 {
		ac = NewAhoCorasick(keywords)
	}

	return &RuleEngine{rules: rules, ac: ac}
}

// Process 执行规则匹配与替换。
// 流程：AC 预筛选 → 数字扫描 → regexp 精确匹配 → 去重 → 从后向前替换。
func (e *RuleEngine) Process(text string) (models.DeidentifyResult, error) {
	result := models.DeidentifyResult{
		OriginalText: text,
		SafeText:     text,
		Entities:     make([]models.SensitiveEntity, 0),
		Placeholder:  make(map[string]string),
	}

	if len(e.rules) == 0 || text == "" {
		return result, nil
	}

	ruleActive := e.activateRules(text)
	matches := e.collectMatches(text, ruleActive)
	matches = deduplicateMatches(matches)
	if len(matches) == 0 {
		return result, nil
	}

	return e.replaceMatches(result, matches, text), nil
}

// activateRules 通过 AC 预筛选和数字扫描确定需要激活的规则。
func (e *RuleEngine) activateRules(text string) map[string]bool {
	ruleActive := make(map[string]bool)
	for _, r := range e.rules {
		// 无关键词的规则默认激活（后续通过数字扫描二次判断）
		ruleActive[r.config.Name] = len(r.config.Keywords) == 0
	}

	// AC 预筛选：扫描特征关键词，激活对应规则
	if e.ac != nil {
		acMatches := e.ac.Search(text)
		for _, m := range acMatches {
			for _, r := range e.rules {
				if slices.Contains(r.config.Keywords, m.Pattern) {
					ruleActive[r.config.Name] = true
					break
				}
			}
		}
	}

	// 数字长度快速扫描：为 min_digits > 0 但未激活的规则做二次判断
	maxMinDigits := 0
	for _, r := range e.rules {
		if !ruleActive[r.config.Name] && r.config.MinDigits > maxMinDigits {
			maxMinDigits = r.config.MinDigits
		}
	}
	if maxMinDigits > 0 {
		maxFound := maxDigitSequence(text)
		for _, r := range e.rules {
			if !ruleActive[r.config.Name] && r.config.MinDigits > 0 && maxFound >= r.config.MinDigits {
				ruleActive[r.config.Name] = true
			}
		}
	}

	return ruleActive
}

// collectMatches 在原始文本上执行所有激活规则的 regexp 匹配。
func (e *RuleEngine) collectMatches(text string, ruleActive map[string]bool) []matchInfo {
	var matches []matchInfo
	for _, r := range e.rules {
		if !ruleActive[r.config.Name] {
			continue
		}
		for _, re := range r.patterns {
			for _, loc := range re.FindAllStringIndex(text, -1) {
				matches = append(matches, matchInfo{
					rule:  r,
					start: loc[0],
					end:   loc[1],
					text:  text[loc[0]:loc[1]],
				})
			}
		}
	}
	return matches
}

// deduplicateMatches 按 start 升序排序并去除重叠匹配，保留先出现的。
func deduplicateMatches(matches []matchInfo) []matchInfo {
	if len(matches) == 0 {
		return matches
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].start == matches[j].start {
			return matches[i].end < matches[j].end
		}
		return matches[i].start < matches[j].start
	})

	var deduped []matchInfo
	for _, m := range matches {
		overlap := false
		for _, d := range deduped {
			if m.start < d.end && m.end > d.start {
				overlap = true
				break
			}
		}
		if !overlap {
			deduped = append(deduped, m)
		}
	}
	return deduped
}

// replaceMatches 按 start 降序从后向前替换匹配文本，避免偏移量混乱。
func (e *RuleEngine) replaceMatches(result models.DeidentifyResult, matches []matchInfo, text string) models.DeidentifyResult {
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].start > matches[j].start
	})

	safeText := text
	for _, m := range matches {
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(m.text+fmt.Sprintf("_%d_%d", m.start, m.end))))[:8]
		placeholder := fmt.Sprintf("{{%s_%s}}", m.rule.config.Placeholder, hash)

		entity := models.SensitiveEntity{
			Text:        m.text,
			Type:        m.rule.config.EntityType,
			Level:       m.rule.level,
			StartPos:    m.start,
			EndPos:      m.end,
			Placeholder: placeholder,
		}
		result.Entities = append(result.Entities, entity)

		// P2 级记录占位符映射（可逆）；P3 级不记录（不可逆）
		if m.rule.level == models.P2Internal {
			result.Placeholder[placeholder] = m.text
		}

		safeText = safeText[:m.start] + placeholder + safeText[m.end:]
	}

	result.SafeText = safeText
	return result
}

// maxDigitSequence 返回文本中最长连续数字序列的长度。
func maxDigitSequence(text string) int {
	maxCount, count := 0, 0
	for _, r := range text {
		if r >= '0' && r <= '9' {
			count++
			if count > maxCount {
				maxCount = count
			}
		} else {
			count = 0
		}
	}
	return maxCount
}

// Restore 将脱敏后的文本还原为原始文本（仅支持 P2 级占位符）。
func Restore(deid models.DeidentifyResult) string {
	text := deid.SafeText
	for placeholder, original := range deid.Placeholder {
		text = strings.ReplaceAll(text, placeholder, original)
	}
	return text
}
