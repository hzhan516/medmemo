package usecase

import (
	"strings"
)

// MemoryIntent 表示可本地短路的个人事实查询意图。
type MemoryIntent string

const (
	IntentPersonalWeight    MemoryIntent = "personal_weight"
	IntentPersonalHeight    MemoryIntent = "personal_height"
	IntentPersonalAge       MemoryIntent = "personal_age"
	IntentAllergyHistory    MemoryIntent = "allergy_history"
	IntentMedicationHistory MemoryIntent = "medication_history"
)

// IntentConfidence 表示意图匹配的置信度级别。
type IntentConfidence int

const (
	ConfidenceHigh   IntentConfidence = iota + 1 // 可本地短路
	ConfidenceMedium                             // 仅用于 query expansion，不直接本地回答
	ConfidenceLow                                // 不走短路，避免误判为医疗建议
)

// intentConfig 保存单个意图的匹配配置。
type intentConfig struct {
	Predicates      []string // 映射到 fact.predicate 的值（支持多 predicate）
	HighPatterns    []string // 高置信匹配模式（必须是明确"查本人记录"的完整句式）
	MediumKeywords  []string // 中置信关键词（仅用于 query expansion，不触发本地回答）
	BlockedSuffixes []string // 阻断后缀（命中后强制降为 Low）
}

// intentAliasConfig 是意图匹配的内置配置表。
// 只有 ConfidenceHigh 才允许本地短路回答。
var intentAliasConfig = map[MemoryIntent]intentConfig{
	IntentPersonalWeight: {
		Predicates:      []string{"体重是"},
		HighPatterns:    []string{"我现在多重", "我多少斤", "我多少公斤", "我的体重是多少", "我体重多少", "我重量多少"},
		MediumKeywords:  []string{"多重", "重量", "几斤", "多少斤", "多少公斤", "体重"},
		BlockedSuffixes: []string{"怎么办", "怎么减", "正常吗", "是不是太", "怎么变"},
	},
	IntentPersonalHeight: {
		Predicates:      []string{"身高是"},
		HighPatterns:    []string{"我多高", "我的身高是多少", "我身高多少"},
		MediumKeywords:  []string{"多高", "身高", "身长"},
		BlockedSuffixes: []string{"怎么办", "正常吗", "是不是太"},
	},
	IntentPersonalAge: {
		Predicates:      []string{"年龄是"},
		HighPatterns:    []string{"我几岁", "我多大", "我的年龄是多少", "我年龄多少"},
		MediumKeywords:  []string{"几岁", "多大", "年龄"},
		BlockedSuffixes: []string{"怎么办", "正常吗"},
	},
	IntentAllergyHistory: {
		Predicates:      []string{"过敏史为", "对...过敏"},
		HighPatterns:    []string{"我对什么过敏", "我过敏史是什么", "我对哪些东西过敏"},
		MediumKeywords:  []string{"过敏史", "过敏"},
		BlockedSuffixes: []string{"怎么办", "怎么治", "该吃什么药", "能吃什么"},
	},
	IntentMedicationHistory: {
		Predicates:      []string{"服用", "正在服用"},
		HighPatterns:    []string{"我正在吃什么药", "我服用什么药", "我吃什么药", "我的用药是什么"},
		MediumKeywords:  []string{"吃什么药", "用药", "服用"},
		BlockedSuffixes: []string{"怎么办", "该吃什么药", "应该吃什么药", "怎么吃", "有什么副作用"},
	},
}

// IntentResult 保存意图解析结果。
type IntentResult struct {
	Intent     MemoryIntent
	Confidence IntentConfidence
	Predicates []string // 映射到的 fact predicate 列表
}

// IntentResolver 负责从用户 query 中解析意图。
type IntentResolver struct {
	expansion *QueryExpansionService
}

// NewIntentResolver 创建新的意图解析器。
func NewIntentResolver(expansion *QueryExpansionService) *IntentResolver {
	return &IntentResolver{expansion: expansion}
}

// Resolve 解析用户 query，返回意图匹配结果。
// 未命中任何意图时返回 nil。
func (r *IntentResolver) Resolve(query string) *IntentResult {
	normalized := r.expansion.Normalize(query)
	if normalized == "" {
		return nil
	}

	// 先检查阻断后缀：任何含阻断后缀的 query 直接降为 Low
	for intent, cfg := range intentAliasConfig {
		if r.hasBlockedSuffix(normalized, cfg.BlockedSuffixes) {
			// 继续检查是否命中了该 intent 的 medium，用于后续 query expansion
			if r.matchesMedium(normalized, cfg.MediumKeywords) {
				return &IntentResult{
					Intent:     intent,
					Confidence: ConfidenceLow,
					Predicates: cfg.Predicates,
				}
			}
			continue
		}

		// High：必须匹配完整模式（完全相等或以"我"开头且包含核心词）
		if r.matchesHigh(normalized, cfg.HighPatterns) {
			return &IntentResult{
				Intent:     intent,
				Confidence: ConfidenceHigh,
				Predicates: cfg.Predicates,
			}
		}

		// Medium：包含关键词即可
		if r.matchesMedium(normalized, cfg.MediumKeywords) {
			return &IntentResult{
				Intent:     intent,
				Confidence: ConfidenceMedium,
				Predicates: cfg.Predicates,
			}
		}
	}

	return nil
}

// matchesHigh 检查 normalized 是否匹配 HighPatterns。
// 规则：完全相等，或以"我"开头且包含 pattern 中的核心词（去掉前缀"我"后的部分）。
func (r *IntentResolver) matchesHigh(normalized string, patterns []string) bool {
	for _, p := range patterns {
		if normalized == p {
			return true
		}
		// 以"我"开头且包含 pattern 核心部分（去掉开头的"我"）
		if strings.HasPrefix(normalized, "我") && len(p) > 3 {
			core := strings.TrimPrefix(p, "我")
			if core != p && strings.Contains(normalized, core) {
				return true
			}
		}
	}
	return false
}

// matchesMedium 检查 normalized 是否包含任意 MediumKeyword。
func (r *IntentResolver) matchesMedium(normalized string, keywords []string) bool {
	return containsAny(normalized, keywords)
}

// hasBlockedSuffix 检查 normalized 是否包含任意阻断后缀。
func (r *IntentResolver) hasBlockedSuffix(normalized string, suffixes []string) bool {
	return containsAny(normalized, suffixes)
}

// containsAny 检查 s 是否包含 candidates 中任意子串。
func containsAny(s string, candidates []string) bool {
	for _, c := range candidates {
		if strings.Contains(s, c) {
			return true
		}
	}
	return false
}
