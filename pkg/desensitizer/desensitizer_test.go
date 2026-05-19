package desensitizer

import (
	"testing"

	"github.com/medmemo/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleEngine_Process_IDCard(t *testing.T) {
	engine := NewRuleEngine()
	result, err := engine.Process("我的身份证号是 110105199001011234，请帮我看看")
	require.NoError(t, err)
	assert.NotEqual(t, result.OriginalText, result.SafeText)
	assert.Contains(t, result.SafeText, "{{ID_CARD_")
	assert.NotContains(t, result.SafeText, "110105199001011234")
	assert.Len(t, result.Entities, 1)
	assert.Equal(t, "身份证号", result.Entities[0].Type)
	assert.Equal(t, models.P3Confidential, result.Entities[0].Level)
}

func TestRuleEngine_Process_Phone(t *testing.T) {
	engine := NewRuleEngine()
	result, err := engine.Process("联系我请拨打 13800138000")
	require.NoError(t, err)
	assert.Contains(t, result.SafeText, "{{PHONE_")
	assert.NotContains(t, result.SafeText, "13800138000")
	assert.Len(t, result.Entities, 1)
	assert.Equal(t, "手机号", result.Entities[0].Type)
}

func TestRuleEngine_Process_BankCard(t *testing.T) {
	engine := NewRuleEngine()
	result, err := engine.Process("银行卡号 6222021234567890123 已绑定")
	require.NoError(t, err)
	assert.Contains(t, result.SafeText, "{{BANK_CARD_")
	assert.NotContains(t, result.SafeText, "6222021234567890123")
	assert.Len(t, result.Entities, 1)
	assert.Equal(t, "银行卡号", result.Entities[0].Type)
}

func TestRuleEngine_Process_Email(t *testing.T) {
	engine := NewRuleEngine()
	result, err := engine.Process("请发送邮件到 user@example.com 谢谢")
	require.NoError(t, err)
	assert.Contains(t, result.SafeText, "{{EMAIL_")
	assert.NotContains(t, result.SafeText, "user@example.com")
	assert.Len(t, result.Entities, 1)
	assert.Equal(t, "邮箱", result.Entities[0].Type)
	assert.Equal(t, models.P2Internal, result.Entities[0].Level)
	// P2 级应有占位符映射，可还原
	assert.NotEmpty(t, result.Placeholder)
}

func TestRuleEngine_Process_URL(t *testing.T) {
	engine := NewRuleEngine()
	result, err := engine.Process("参考链接 https://example.com/article 了解更多")
	require.NoError(t, err)
	assert.Contains(t, result.SafeText, "{{URL_")
	assert.NotContains(t, result.SafeText, "https://example.com/article")
	assert.Len(t, result.Entities, 1)
	assert.Equal(t, "URL", result.Entities[0].Type)
}

func TestRuleEngine_Process_Multiple(t *testing.T) {
	engine := NewRuleEngine()
	text := "电话 13800138000，邮箱 test@test.com，身份证 110105199001011234"
	result, err := engine.Process(text)
	require.NoError(t, err)
	assert.Len(t, result.Entities, 3)
	assert.NotContains(t, result.SafeText, "13800138000")
	assert.NotContains(t, result.SafeText, "test@test.com")
	assert.NotContains(t, result.SafeText, "110105199001011234")
}

func TestRuleEngine_Process_NoSensitive(t *testing.T) {
	engine := NewRuleEngine()
	text := "今天天气不错，适合出去散步。"
	result, err := engine.Process(text)
	require.NoError(t, err)
	assert.Equal(t, text, result.SafeText)
	assert.Empty(t, result.Entities)
}

func TestRuleEngine_Process_Empty(t *testing.T) {
	engine := NewRuleEngine()
	result, err := engine.Process("")
	require.NoError(t, err)
	assert.Equal(t, "", result.SafeText)
	assert.Empty(t, result.Entities)
}

func TestRestore(t *testing.T) {
	engine := NewRuleEngine()
	original := "请发送邮件到 user@example.com 谢谢"
	result, err := engine.Process(original)
	require.NoError(t, err)
	restored := Restore(result)
	assert.Equal(t, original, restored)
}

func TestRestore_P3Irreversible(t *testing.T) {
	engine := NewRuleEngine()
	original := "我的身份证号是 110105199001011234"
	result, err := engine.Process(original)
	require.NoError(t, err)
	// P3 级不记录占位符映射，无法还原
	restored := Restore(result)
	assert.Contains(t, restored, "{{ID_CARD_")
	assert.NotContains(t, restored, "110105199001011234")
}
