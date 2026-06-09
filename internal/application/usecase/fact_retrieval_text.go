package usecase

import (
	"strings"

	"github.com/hzhan516/medmemo/internal/domain/entity"
)

// BuildFactRetrievalText 为 approved fact 生成用于 embedding 和关键词召回的 retrieval text。
//
// 该文本只用于向量检索和本地关键词匹配，不用于 UI 展示、不注入 LLM 上下文。
// 当前版本仅对体重类事实做同义词扩展，后续可扩展身高/年龄/血压/血糖等类别。
func BuildFactRetrievalText(fact *entity.ExtractedFact) string {
	if fact == nil {
		return ""
	}

	baseText := buildBaseText(fact)
	if baseText == "" {
		return ""
	}

	if isWeightFact(fact) {
		synonyms := []string{
			"体重", "体重值", "重量", "公斤", "千克", "kg", "斤",
			"多重", "多少斤", "几斤", "我现在多重", "我多少斤",
		}
		return baseText + "。相关问法：" + strings.Join(synonyms, "、") + "。"
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

// isWeightFact 判断事实是否属于体重类，用于决定是否附加同义词扩展。
func isWeightFact(fact *entity.ExtractedFact) bool {
	predLower := strings.ToLower(fact.Predicate)
	objLower := strings.ToLower(fact.Object)

	weightPredicates := []string{"体重", "重量", "重达"}
	for _, wp := range weightPredicates {
		if strings.Contains(predLower, wp) {
			return true
		}
	}

	weightObjects := []string{"公斤", "千克", "kg", "斤"}
	for _, wo := range weightObjects {
		if strings.Contains(objLower, wo) {
			return true
		}
	}

	return false
}
