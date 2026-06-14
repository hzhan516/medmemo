package usecase

import (
	"strings"
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestBuildFactRetrievalText(t *testing.T) {
		t.Parallel()
	tests := []struct {
		name     string
		fact     *entity.ExtractedFact
		wantSub  []string // 期望包含的子串
		wantNot  []string // 期望不包含的子串
		category string
	}{
		{
			name:     "体重类 predicate 命中",
			fact:     &entity.ExtractedFact{Subject: "用户", Predicate: "体重是", Object: "70公斤"},
			wantSub:  []string{"用户 体重是 70公斤", "多重", "多少斤", "我现在多重"},
			category: "体重",
		},
		{
			name:     "体重类 object 命中",
			fact:     &entity.ExtractedFact{Subject: "用户", Predicate: "重达", Object: "140斤"},
			wantSub:  []string{"用户 重达 140斤", "斤", "我多少斤"},
			category: "体重",
		},
		{
			name:     "身高类 predicate 命中",
			fact:     &entity.ExtractedFact{Subject: "用户", Predicate: "身高是", Object: "175厘米"},
			wantSub:  []string{"用户 身高是 175厘米", "多高", "我身高多少"},
			category: "身高",
		},
		{
			name:     "身高类 object 命中 cm",
			fact:     &entity.ExtractedFact{Subject: "用户", Predicate: "高度为", Object: "180cm"},
			wantSub:  []string{"用户 高度为 180cm", "几cm"},
			category: "身高",
		},
		{
			name:     "年龄类命中",
			fact:     &entity.ExtractedFact{Subject: "用户", Predicate: "年龄是", Object: "30岁"},
			wantSub:  []string{"用户 年龄是 30岁", "几岁", "多大", "我今年几岁"},
			category: "年龄",
		},
		{
			name:     "血压类命中",
			fact:     &entity.ExtractedFact{Subject: "用户", Predicate: "血压是", Object: "120/80 mmHg"},
			wantSub:  []string{"用户 血压是 120/80 mmHg", "高压", "低压", "我血压多少"},
			category: "血压",
		},
		{
			name:     "血糖类命中",
			fact:     &entity.ExtractedFact{Subject: "用户", Predicate: "空腹血糖是", Object: "5.6 mmol"},
			wantSub:  []string{"用户 空腹血糖是 5.6 mmol", "餐后血糖", "血糖高不高"},
			category: "血糖",
		},
		{
			name:     "过敏类命中",
			fact:     &entity.ExtractedFact{Subject: "用户", Predicate: "过敏史为", Object: "青霉素过敏"},
			wantSub:  []string{"用户 过敏史为 青霉素过敏", "过敏原", "有没有过敏"},
			category: "过敏",
		},
		{
			name:     "用药类命中",
			fact:     &entity.ExtractedFact{Subject: "用户", Predicate: "正在服用", Object: "阿司匹林"},
			wantSub:  []string{"用户 正在服用 阿司匹林", "用药", "在吃什么药"},
			category: "用药",
		},
		{
			name:     "未匹配分类返回基础文本",
			fact:     &entity.ExtractedFact{Subject: "用户", Predicate: "职业是", Object: "工程师"},
			wantSub:  []string{"用户 职业是 工程师"},
			wantNot:  []string{"相关问法"},
			category: "",
		},
		{
			name:     "nil fact 返回空串",
			fact:     nil,
			wantSub:  []string{},
			category: "",
		},
		{
			name:     "空字段返回空串",
			fact:     &entity.ExtractedFact{Subject: "", Predicate: "", Object: ""},
			wantSub:  []string{},
			category: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFactRetrievalText(tt.fact)
			for _, sub := range tt.wantSub {
				assert.Contains(t, got, sub, "期望包含子串: %s", sub)
			}
			for _, sub := range tt.wantNot {
				assert.NotContains(t, got, sub, "不应包含子串: %s", sub)
			}
			if tt.category != "" && tt.fact != nil {
				cat := matchCategory(tt.fact)
				assert.NotNil(t, cat)
				assert.Equal(t, tt.category, cat.name)
			}
		})
	}
}

func TestBuildFactRetrievalText_NoLLMOrUIExposure(t *testing.T) {
		t.Parallel()
	// 合规红线：同义词扩展文本不注入 LLM，此测试仅验证输出格式稳定。
	fact := &entity.ExtractedFact{Subject: "用户", Predicate: "体重是", Object: "70公斤"}
	text := BuildFactRetrievalText(fact)
	assert.True(t, strings.HasPrefix(text, "用户 体重是 70公斤"))
	assert.Contains(t, text, "。相关问法：")
	assert.True(t, strings.HasSuffix(text, "。"))
}

func TestMatchCategory_Priority(t *testing.T) {
		t.Parallel()
	// 体重在注册表中排第一，事实同时含体重和身高关键词时应命中体重。
	fact := &entity.ExtractedFact{Subject: "用户", Predicate: "体重是", Object: "70公斤且身高175"}
	cat := matchCategory(fact)
	assert.NotNil(t, cat)
	assert.Equal(t, "体重", cat.name)
}
