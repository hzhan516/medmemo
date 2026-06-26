package usecase

import (
	"strings"

	"github.com/hzhan516/medmemo/internal/domain/entity"
)

// factCategory 定义事实类别，用于受控同义词扩展。
// 仅用于向量检索和本地关键词匹配，不污染 UI 展示、不注入 LLM 上下文。
type factCategory struct {
	name       string
	predicates []string // predicate 匹配关键词
	objects    []string // object 匹配关键词
	synonyms   []string // 同义词扩展列表
}

// categoryRegistry 受控分类注册表（按优先级排列，首个匹配即返回）。
// 新增分类时请保持「predicate/object 匹配优先、同义词仅用于检索召回」的约束。
var categoryRegistry = []factCategory{
	{
		name:       "体重",
		predicates: []string{"体重", "重量", "重达"},
		objects:    []string{"公斤", "千克", "kg", "斤"},
		synonyms: []string{
			"体重", "体重值", "重量", "公斤", "千克", "kg", "斤",
			"多重", "多少斤", "几斤", "我现在多重", "我多少斤",
		},
	},
	{
		name:       "身高",
		predicates: []string{"身高", "高度", "身长"},
		objects:    []string{"厘米", "cm", "米", "m"},
		synonyms: []string{
			"身高", "身高值", "高度", "多高", "多少厘米", "几cm",
			"我现在多高", "我身高多少",
		},
	},
	{
		name:       "年龄",
		predicates: []string{"年龄", "岁数", "出生", "生日"},
		objects:    []string{"年龄", "岁数", "出生", "生日"},
		synonyms: []string{
			"年龄", "岁数", "几岁", "多大", "多少岁",
			"我今年几岁", "我多大了",
		},
	},
	{
		name:       "血压",
		predicates: []string{"血压", "收缩压", "舒张压", "高压", "低压"},
		objects:    []string{"mmHg"},
		synonyms: []string{
			"血压", "血压值", "高压", "低压", "收缩压", "舒张压",
			"mmHg", "我血压多少",
		},
	},
	{
		name:       "血糖",
		predicates: []string{"血糖", "血糖值", "空腹血糖", "餐后血糖"},
		objects:    []string{"mmol", "血糖", "血糖值", "空腹血糖", "餐后血糖"},
		synonyms: []string{
			"血糖", "血糖值", "空腹血糖", "餐后血糖", "mmol",
			"我血糖多少", "血糖高不高",
		},
	},
	{
		name:       "过敏",
		predicates: []string{"过敏", "过敏原", "过敏史", "敏感"},
		objects:    []string{"过敏", "过敏原", "过敏史", "敏感"},
		synonyms: []string{
			"过敏", "过敏原", "过敏史", "过敏反应",
			"我对什么过敏", "有没有过敏",
		},
	},
	{
		name:       "用药",
		predicates: []string{"用药", "服药", "服用", "药物", "药品", "处方"},
		objects:    []string{"用药", "服药", "服用", "药物", "药品", "处方"},
		synonyms: []string{
			"用药", "服药", "药物", "药品",
			"在吃什么药", "正在服用", "用药记录",
		},
	},
}

// BuildFactRetrievalText 为 approved fact 生成用于 embedding 和关键词召回的 retrieval text。
//
// 该文本只用于向量检索和本地关键词匹配，不用于 UI 展示、不注入 LLM 上下文。
// 输出格式保持模板化：{baseText}。相关问法：{synonyms}。
func BuildFactRetrievalText(fact *entity.ExtractedFact) string {
	if fact == nil {
		return ""
	}

	baseText := buildBaseText(fact)
	if baseText == "" {
		return ""
	}

	if cat := matchCategory(fact); cat != nil {
		return baseText + "。相关问法：" + strings.Join(cat.synonyms, "、") + "。"
	}

	return baseText
}

// buildBaseText 将三元组拼接为基础文本，过滤空字段并规范化空格。
func buildBaseText(fact *entity.ExtractedFact) string {
	fields := []string{fact.Subject, fact.Predicate, fact.Object}
	var nonEmpty []string
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f != "" {
			nonEmpty = append(nonEmpty, f)
		}
	}
	return strings.Join(nonEmpty, " ")
}

// matchCategory 按 predicate/object 关键词匹配事实类别，首个命中即返回。
func matchCategory(fact *entity.ExtractedFact) *factCategory {
	predLower := strings.ToLower(fact.Predicate)
	objLower := strings.ToLower(fact.Object)

	for _, cat := range categoryRegistry {
		if containsAnyKeyword(predLower, cat.predicates) || containsAnyKeyword(objLower, cat.objects) {
			return &cat
		}
	}
	return nil
}

// containsAnyKeyword 检查文本中是否包含任意关键词。
// 对 1-2 个字符的纯 ASCII 关键词增加边界检查，避免 "m" 误命中 "mmol"、"mmHg"。
func containsAnyKeyword(text string, keywords []string) bool {
	for _, kw := range keywords {
		if containsKeywordBounded(text, kw) {
			return true
		}
	}
	return false
}

// containsKeywordBounded 在 text 中查找 keyword，短 ASCII 词需满足边界条件。
func containsKeywordBounded(text, keyword string) bool {
	lowerText := strings.ToLower(text)
	lowerKw := strings.ToLower(keyword)

	idx := strings.Index(lowerText, lowerKw)
	if idx == -1 {
		return false
	}

	// 关键词长度 > 2 个 rune，或非纯 ASCII（如中文、mmHg），直接子串匹配即可。
	runes := []rune(lowerKw)
	if len(runes) > 2 || !isAllASCII(lowerKw) {
		return true
	}

	// 短 ASCII 关键词需检查前后边界，防止命中更长单词内部。
	if idx > 0 {
		b := lowerText[idx-1]
		if (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') {
			return false
		}
	}
	end := idx + len(lowerKw)
	if end < len(lowerText) {
		a := lowerText[end]
		if (a >= 'a' && a <= 'z') || (a >= '0' && a <= '9') {
			return false
		}
	}
	return true
}

// isAllASCII 判断字符串是否仅含 ASCII 字符。
func isAllASCII(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool { return r > 127 }) == -1
}
