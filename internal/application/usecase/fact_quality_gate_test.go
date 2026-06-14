package usecase

import (
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestApplyFactQualityGate_RejectsQuestionIntent(t *testing.T) {
		t.Parallel()
	facts := []*entity.ExtractedFact{
		entity.NewExtractedFact("用户", "询问", "体重", 0.8, []string{"msg_1"}),
		entity.NewExtractedFact("用户", "想知道", "血压", 0.8, []string{"msg_1"}),
		entity.NewExtractedFact("用户", "咨询", "病情", 0.8, []string{"msg_1"}),
	}
	result := ApplyFactQualityGate(facts)
	assert.Empty(t, result, "询问意图应被全部拒绝")
}

func TestApplyFactQualityGate_RejectsAICapabilityLimit(t *testing.T) {
		t.Parallel()
	facts := []*entity.ExtractedFact{
		entity.NewExtractedFact("AI", "无法告知", "用户体重", 0.8, []string{"msg_1"}),
		entity.NewExtractedFact("助手", "不知道", "年龄", 0.8, []string{"msg_1"}),
		entity.NewExtractedFact("模型", "不能判断", "病情", 0.8, []string{"msg_1"}),
	}
	result := ApplyFactQualityGate(facts)
	assert.Empty(t, result, "AI 能力限制应被全部拒绝")
}

func TestApplyFactQualityGate_RejectsToolAdvice(t *testing.T) {
		t.Parallel()
	facts := []*entity.ExtractedFact{
		entity.NewExtractedFact("用户", "需要使用", "体重秤", 0.8, []string{"msg_1"}),
		entity.NewExtractedFact("用户", "应该使用", "血压计", 0.8, []string{"msg_1"}),
		entity.NewExtractedFact("用户", "建议检查", "血糖", 0.8, []string{"msg_1"}),
	}
	result := ApplyFactQualityGate(facts)
	assert.Empty(t, result, "工具建议应被全部拒绝")
}

func TestApplyFactQualityGate_AcceptsConcreteWeight(t *testing.T) {
		t.Parallel()
	facts := []*entity.ExtractedFact{
		entity.NewExtractedFact("用户", "体重是", "110公斤", 0.95, []string{"msg_1"}),
	}
	result := ApplyFactQualityGate(facts)
	assert.Len(t, result, 1)
	assert.Equal(t, "110公斤", result[0].Object)
}

func TestApplyFactQualityGate_RejectsPersonalAttrWithoutValue(t *testing.T) {
		t.Parallel()
	facts := []*entity.ExtractedFact{
		entity.NewExtractedFact("用户", "体重", "", 0.8, []string{"msg_1"}),
		entity.NewExtractedFact("用户", "身高是", "很高", 0.8, []string{"msg_1"}),
		entity.NewExtractedFact("用户", "年龄", "未知", 0.8, []string{"msg_1"}),
	}
	result := ApplyFactQualityGate(facts)
	assert.Empty(t, result, "缺少具体数值的个人属性应被拒绝")
}

func TestApplyFactQualityGate_PassesNormalFact(t *testing.T) {
		t.Parallel()
	facts := []*entity.ExtractedFact{
		entity.NewExtractedFact("用户", "患有", "偏头痛", 0.9, []string{"msg_1"}),
		entity.NewExtractedFact("用户", "服用", "阿司匹林", 0.8, []string{"msg_1"}),
	}
	result := ApplyFactQualityGate(facts)
	assert.Len(t, result, 2)
}

func TestApplyFactQualityGate_EmptyInput(t *testing.T) {
		t.Parallel()
	var facts []*entity.ExtractedFact
	result := ApplyFactQualityGate(facts)
	assert.Empty(t, result)
}
