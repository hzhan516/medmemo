package usecase

import (
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestBuildFactRetrievalText_NilFactReturnsEmpty(t *testing.T) {
	result := BuildFactRetrievalText(nil)
	assert.Empty(t, result)
}

func TestBuildFactRetrievalText_WeightFactIncludesSynonyms(t *testing.T) {
	fact := &entity.ExtractedFact{
		Subject:   "用户",
		Predicate: "体重是",
		Object:    "110公斤",
	}
	result := BuildFactRetrievalText(fact)

	assert.Contains(t, result, "用户 体重是 110公斤")
	assert.Contains(t, result, "相关问法：")
	assert.Contains(t, result, "多少斤")
	assert.Contains(t, result, "几斤")
	assert.Contains(t, result, "多重")
	assert.Contains(t, result, "斤")
	assert.Contains(t, result, "公斤")
	assert.Contains(t, result, "千克")
	assert.Contains(t, result, "kg")
	assert.Contains(t, result, "体重值")
	assert.Contains(t, result, "我现在多重")
	assert.Contains(t, result, "我多少斤")
}

func TestBuildFactRetrievalText_WeightFactByPredicateZhongda(t *testing.T) {
	fact := &entity.ExtractedFact{
		Subject:   "用户",
		Predicate: "重达",
		Object:    "110",
	}
	result := BuildFactRetrievalText(fact)
	assert.Contains(t, result, "相关问法：")
	assert.Contains(t, result, "多重")
}

func TestBuildFactRetrievalText_WeightFactByObjectKG(t *testing.T) {
	fact := &entity.ExtractedFact{
		Subject:   "用户",
		Predicate: "是",
		Object:    "55kg",
	}
	result := BuildFactRetrievalText(fact)
	assert.Contains(t, result, "相关问法：")
	assert.Contains(t, result, "kg")
	assert.Contains(t, result, "多重")
}

func TestBuildFactRetrievalText_WeightFactByObjectUpperKG(t *testing.T) {
	fact := &entity.ExtractedFact{
		Subject:   "用户",
		Predicate: "是",
		Object:    "55KG",
	}
	result := BuildFactRetrievalText(fact)
	assert.Contains(t, result, "相关问法：")
	assert.Contains(t, result, "kg")
}

func TestBuildFactRetrievalText_NonWeightFactWithZhongNotMatched(t *testing.T) {
	fact := &entity.ExtractedFact{
		Subject:   "用户",
		Predicate: "严重",
		Object:    "头痛",
	}
	result := BuildFactRetrievalText(fact)
	assert.NotContains(t, result, "相关问法")
	assert.Equal(t, "用户 严重 头痛", result)
}

func TestBuildFactRetrievalText_NormalFactKeepsBaseText(t *testing.T) {
	fact := &entity.ExtractedFact{
		Subject:   "用户",
		Predicate: "患有",
		Object:    "高血压",
	}
	result := BuildFactRetrievalText(fact)
	assert.Equal(t, "用户 患有 高血压", result)
	assert.NotContains(t, result, "相关问法")
}

func TestBuildFactRetrievalText_EmptyFieldsHandled(t *testing.T) {
	fact := &entity.ExtractedFact{
		Subject:   "用户",
		Predicate: "",
		Object:    "高血压",
	}
	result := BuildFactRetrievalText(fact)
	assert.Equal(t, "用户 高血压", result)
	assert.NotContains(t, result, "  ")
}

func TestBuildFactRetrievalText_AllEmptyFields(t *testing.T) {
	fact := &entity.ExtractedFact{
		Subject:   "",
		Predicate: "",
		Object:    "",
	}
	result := BuildFactRetrievalText(fact)
	assert.Empty(t, result)
}
