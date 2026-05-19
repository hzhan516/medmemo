package main

import (
	"testing"

	"github.com/medmemo/medmemo/internal/application"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckEmergency_ALevel(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"chest_pain", "我胸痛伴呼吸困难，很难受"},
		{"unconscious", "患者意识丧失，需要急救"},
		{"severe_allergy", "严重过敏反应，喉咙肿胀"},
		{"bleeding", "大出血，血流不止"},
		{"stroke", "突发偏瘫，口角歪斜，可能是脑卒中"},
		{"poisoning", "误食农药中毒，快帮忙"},
		{"drowning", "孩子溺水了，没有呼吸"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &WailsApp{}
			result, err := app.CheckEmergency(tt.text)
			require.NoError(t, err)
			assert.Equal(t, "A", result.Level)
			assert.NotEmpty(t, result.Message)
			assert.Contains(t, result.Action, "120")
		})
	}
}

func TestCheckEmergency_BLevel(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"high_fever", "持续高热三天不退"},
		{"abdominal_pain", "剧烈腹痛，难以忍受"},
		{"blood_in_urine", "发现血尿，尿液带血"},
		{"vision_loss", "视力突然下降，看不清东西"},
		{"palpitation", "心悸胸闷，心跳过快"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &WailsApp{}
			result, err := app.CheckEmergency(tt.text)
			require.NoError(t, err)
			assert.Equal(t, "B", result.Level)
			assert.NotEmpty(t, result.Message)
		})
	}
}

func TestCheckEmergency_None(t *testing.T) {
	app := &WailsApp{}
	result, err := app.CheckEmergency("今天天气不错，想了解一下健康饮食")
	require.NoError(t, err)
	assert.Equal(t, "none", result.Level)
	assert.Empty(t, result.Message)
}

func TestCheckEmergency_Empty(t *testing.T) {
	app := &WailsApp{}
	result, err := app.CheckEmergency("")
	require.NoError(t, err)
	assert.Equal(t, "none", result.Level)
}

// TestCheckEmergency_Delegation 验证 wails_app.go 的 CheckEmergency 正确委托到 application 层。
func TestCheckEmergency_Delegation(t *testing.T) {
	app := &WailsApp{}

	// A 级
	result, err := app.CheckEmergency("胸痛 呼吸困难")
	require.NoError(t, err)
	assert.Equal(t, "A", result.Level)

	// B 级
	result, err = app.CheckEmergency("持续高热")
	require.NoError(t, err)
	assert.Equal(t, "B", result.Level)

	// 无命中
	result, err = app.CheckEmergency("普通感冒吃什么好")
	require.NoError(t, err)
	assert.Equal(t, "none", result.Level)
}

// TestEvaluateEmergency_Integration 直接测试 application 层紧急检测引擎。
func TestEvaluateEmergency_Integration(t *testing.T) {
	// A 级覆盖更多关键词
	cases := []string{
		"心跳骤停怎么办",
		"过敏性休克紧急处理",
		"孕妇破水了",
		"新生儿发热不退",
		"一氧化碳中毒",
		"电击伤后昏迷",
		"窒息喘不过气",
	}
	for _, text := range cases {
		result := application.EvaluateEmergency(text)
		assert.Equal(t, application.LevelA, result.Level, "文本: %s", text)
	}

	// B 级覆盖更多关键词
	cases = []string{
		"黑便三天了",
		"便血伴有腹痛",
		"黄疸加重",
		"咯血痰中带血",
		"关节红肿热痛",
		"不明原因发热消瘦",
	}
	for _, text := range cases {
		result := application.EvaluateEmergency(text)
		assert.Equal(t, application.LevelB, result.Level, "文本: %s", text)
	}
}
