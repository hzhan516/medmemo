package pipeline

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"sort"

	"github.com/hzhan516/medmemo/pkg/models"
)

// L1ExtendedRuleStage 是严格级（strict）专属的 L1.5 兜底规则阶段。
// 仅当 input.Level == models.DesensitizationStrict 时激活，否则原样透传，
// 不影响标准级（standard）与关闭级（off）行为。
//
// 该阶段以确定性正则覆盖 L1 基础规则与 L2 NER 之外常见的可标识信息：
// 出生日期、年龄+姓名、地址、病历/病案号、车牌号、护照号、IP 地址。
// 作为流水线最后一道兜底，运行在 L1+L2 已处理后的文本上，
// 对残留的可标识信息做硬遮蔽，从而使严格级的可见 PII 不多于标准级。
type L1ExtendedRuleStage struct {
	rules []extendedRule
}

// extendedRule 单条兜底正则规则。
type extendedRule struct {
	name    string
	prefix  string // 占位符前缀，如 DOB / ADDR
	level   models.SensitivityLevel
	pattern *regexp.Regexp
}

// NewL1ExtendedRuleStage 创建严格级兜底规则阶段。
// 规则顺序即优先级：靠前的规则在重叠时优先保留，避免同一区域被多次替换。
func NewL1ExtendedRuleStage() *L1ExtendedRuleStage {
	return &L1ExtendedRuleStage{rules: defaultExtendedRules()}
}

// defaultExtendedRules 返回严格级兜底规则集合。
// 正则均为确定性匹配（RE2，无回溯），保证延迟可控且行为可预测。
func defaultExtendedRules() []extendedRule {
	return []extendedRule{
		{
			name:   "birth_date",
			prefix: "DOB",
			level:  models.P3Confidential,
			// 出生日期：1900-2099 年，支持 - / . 年月日 分隔。
			pattern: regexp.MustCompile(`(?:19|20)\d{2}\s*[-/.年]\s*(?:0?[1-9]|1[0-2])\s*[-/.月]\s*(?:0?[1-9]|[12]\d|3[01])\s*日?`),
		},
		{
			name:   "ip_address",
			prefix: "IP",
			level:  models.P3Confidential,
			// IPv4 地址：严格限定每段 0-255，避免误伤普通小数串。
			pattern: regexp.MustCompile(`(?:(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)\.){3}(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)`),
		},
		{
			name:   "license_plate",
			prefix: "PLATE",
			level:  models.P3Confidential,
			// 车牌号：省份简称 + 字母 + 5~6 位（含新能源）。
			pattern: regexp.MustCompile(`[京津沪渝冀豫云辽黑湘皖鲁新苏浙赣鄂桂甘晋蒙陕吉闽贵粤青藏川宁琼][A-Z][A-HJ-NP-Z0-9]{5,6}`),
		},
		{
			name:   "medical_record_no",
			prefix: "MRN",
			level:  models.P3Confidential,
			// 病历/病案/门诊/住院/就诊/医保 号。
			pattern: regexp.MustCompile(`(?:病历号|病案号|门诊号|住院号|就诊卡号|就诊号|医保卡号|社保号)[:：]?\s*[A-Za-z0-9]{4,}`),
		},
		{
			name:   "passport_no",
			prefix: "PASSPORT",
			level:  models.P3Confidential,
			// 护照号：1 位字母 + 8 位数字（中国护照 E/G/等开头）。
			pattern: regexp.MustCompile(`\b[A-Za-z][0-9]{8}\b`),
		},
		{
			name:   "address",
			prefix: "ADDR",
			level:  models.P3Confidential,
			// 地址：省/市/区县 + 路/街/巷 + 号/室/楼 等，允许中间夹带汉字与数字。
			pattern: regexp.MustCompile(`[\p{Han}]{2,}(?:省|市|区|县|自治区|自治州)[\p{Han}\d]*(?:路|街|道|巷|号|栋|幢|单元|室|楼|村|镇|乡)[\p{Han}\d]*(?:号|室|栋|楼|单元)?`),
		},
		{
			name:   "age_name",
			prefix: "AGE_NAME",
			level:  models.P3Confidential,
			// 姓名 + 年龄：如“张三，35岁”“李四 42 周岁”。
			pattern: regexp.MustCompile(`[\p{Han}]{2,4}(?:先生|女士|同学|同志|老师)?[，,、\s]*\d{1,3}\s*(?:岁|周岁)`),
		},
		{
			name:   "age",
			prefix: "AGE",
			level:  models.P2Internal,
			// 独立年龄：如“35岁”“42 周岁”。
			pattern: regexp.MustCompile(`\d{1,3}\s*(?:岁|周岁)`),
		},
	}
}

// match 内部结构：一次正则命中的区间与来源规则。
type extMatch struct {
	start int
	end   int
	rule  extendedRule
}

// Process 在严格级下执行兜底正则遮蔽；非严格级原样透传。
func (s *L1ExtendedRuleStage) Process(_ context.Context, input Input) (Output, error) {
	if input.Level != models.DesensitizationStrict {
		return passthroughOutput(input), nil
	}

	text := input.Text

	// 1. 收集所有规则命中区间。
	var matches []extMatch
	for _, r := range s.rules {
		for _, loc := range r.pattern.FindAllStringIndex(text, -1) {
			matches = append(matches, extMatch{start: loc[0], end: loc[1], rule: r})
		}
	}
	if len(matches) == 0 {
		return passthroughOutput(input), nil
	}

	// 2. 过滤重叠：按 start 升序、更长区间优先，规则顺序作为次级优先级。
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].start != matches[j].start {
			return matches[i].start < matches[j].start
		}
		return (matches[i].end - matches[i].start) > (matches[j].end - matches[j].start)
	})
	kept := make([]extMatch, 0, len(matches))
	lastEnd := -1
	for _, m := range matches {
		if m.start < lastEnd {
			continue // 与已保留区间重叠，跳过
		}
		kept = append(kept, m)
		lastEnd = m.end
	}

	// 3. 从后向前替换，避免偏移量错乱；同时记录实体与占位符映射。
	sort.Slice(kept, func(i, j int) bool { return kept[i].start > kept[j].start })

	if input.Metadata == nil {
		input.Metadata = make(map[string]any)
	}
	newEntities := make([]models.SensitiveEntity, 0, len(kept))
	// 合并已有占位符映射，追加本阶段新增映射（供输出还原使用）。
	placeholders := map[string]string{}
	if existing, ok := input.Metadata["l1_placeholders"].(map[string]string); ok {
		for k, v := range existing {
			placeholders[k] = v
		}
	}

	for i, m := range kept {
		original := text[m.start:m.end]
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(original+fmt.Sprintf("_%s_%d", m.rule.name, i))))[:8]
		placeholder := fmt.Sprintf("{{%s_%s}}", m.rule.prefix, hash)
		text = text[:m.start] + placeholder + text[m.end:]
		// 占位符映射仅收录 P2：P3 走 LocalRestore，避免可还原映射外泄。
		if m.rule.level == models.P2Internal {
			placeholders[placeholder] = original
		}
		newEntities = append(newEntities, models.SensitiveEntity{
			Text:        original,
			Type:        m.rule.name,
			Level:       m.rule.level,
			StartPos:    m.start,
			EndPos:      m.end,
			Placeholder: placeholder,
		})
	}

	// 仅将本阶段新增实体写入 l1_entities（流水线按阶段收集，避免与前序阶段重复计数）。
	input.Metadata["l1_entities"] = newEntities
	input.Metadata["l1_placeholders"] = placeholders
	return Output{Text: text, Metadata: input.Metadata}, nil
}
