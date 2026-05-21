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

func TestRuleEngine_Process_PhoneWithPrefix(t *testing.T) {
	engine := NewRuleEngine()
	result, err := engine.Process("拨打 +8613800138000 联系")
	require.NoError(t, err)
	assert.Contains(t, result.SafeText, "{{PHONE_")
	assert.NotContains(t, result.SafeText, "+8613800138000")
	assert.Len(t, result.Entities, 1)
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

func TestRuleEngine_Process_OverlapPhoneAndBankCard(t *testing.T) {
	// 手机号前11位与银行卡16位重叠时，优先保留先命中的规则（按 rule 顺序）
	engine := NewRuleEngine()
	// 1380013800012345：手机号匹配 13800138000（前11位），银行卡匹配全部16位
	// 由于去重保留先出现的（按 start 排序后先加入的），取决于 rule 加载顺序
	result, err := engine.Process("卡号 1380013800012345")
	require.NoError(t, err)
	// 不应 panic，且至少命中一条规则
	assert.True(t, len(result.Entities) >= 1)
}

func TestRuleEngine_Process_ACSkipEmail(t *testing.T) {
	// 文本中没有 @ 时，AC 预筛选应跳过 email 规则，Regexp 不应执行
	engine := NewRuleEngine()
	text := "请拨打电话 13800138000 联系"
	result, err := engine.Process(text)
	require.NoError(t, err)
	assert.Len(t, result.Entities, 1)
	assert.Equal(t, "手机号", result.Entities[0].Type)
}

func TestRuleEngine_Process_ACSkipURL(t *testing.T) {
	// 文本中没有 http:// 或 https:// 时，AC 预筛选应跳过 URL 规则
	engine := NewRuleEngine()
	text := "访问 example.com 首页"
	result, err := engine.Process(text)
	require.NoError(t, err)
	// example.com 不含协议头，不应被识别为 URL
	assert.Empty(t, result.Entities)
}

func TestRuleEngine_Process_DigitScanActivatesIDCard(t *testing.T) {
	// 文本中有 >=15 位连续数字时，数字扫描应激活身份证规则
	engine := NewRuleEngine()
	text := "编号 110105199001011234 已登记"
	result, err := engine.Process(text)
	require.NoError(t, err)
	assert.Len(t, result.Entities, 1)
	assert.Equal(t, "身份证号", result.Entities[0].Type)
}

func TestRuleEngine_Process_NoLongDigitsSkipRules(t *testing.T) {
	// 文本中无长数字序列时，身份证/手机号/银行卡规则应被跳过
	engine := NewRuleEngine()
	text := "今天去公园散步，看到一只小狗。"
	result, err := engine.Process(text)
	require.NoError(t, err)
	assert.Empty(t, result.Entities)
}
