package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckEmergency_ALevel(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{"chest pain", "我胸痛伴呼吸困难，很难受"},
		{"unconscious", "患者意识丧失，需要急救"},
		{"severe allergy", "严重过敏反应，喉咙肿胀"},
		{"bleeding", "大出血，血流不止"},
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
		{"high fever", "持续高热三天不退"},
		{"abdominal pain", "剧烈腹痛，难以忍受"},
		{"blood in urine", "发现血尿，尿液带血"},
		{"vision loss", "视力突然下降，看不清东西"},
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

func TestContainsAll(t *testing.T) {
	assert.True(t, containsAll("胸痛伴呼吸困难", "胸痛 呼吸困难"))
	assert.True(t, containsAll("我胸痛并且呼吸困难", "胸痛 呼吸困难"))
	assert.False(t, containsAll("只有胸痛", "胸痛 呼吸困难"))
	assert.True(t, containsAll("Hello World", "hello world"))
}
