package pipeline

import (
	"context"
	"regexp"
	"testing"

	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestL1ExtendedRuleStage_PassthroughNonStrict 验证非严格级原样透传，不做任何遮蔽。
func TestL1ExtendedRuleStage_PassthroughNonStrict(t *testing.T) {
	t.Parallel()
	stage := NewL1ExtendedRuleStage()
	ctx := context.Background()

	for _, lvl := range []models.DesensitizationLevel{models.DesensitizationStandard, models.DesensitizationOff, ""} {
		in := Input{Text: "我的IP是192.168.1.1，出生于1990-01-01", Level: lvl, Metadata: map[string]any{}}
		out, err := stage.Process(ctx, in)
		require.NoError(t, err)
		assert.Equal(t, in.Text, out.Text, "level=%q 应原样透传", lvl)
	}
}

// TestL1ExtendedRuleStage_PerRule 验证各兜底正则在严格级下均能遮蔽目标 PII。
func TestL1ExtendedRuleStage_PerRule(t *testing.T) {
	t.Parallel()
	stage := NewL1ExtendedRuleStage()
	ctx := context.Background()

	cases := []struct {
		name    string
		text    string
		leaked  string // 遮蔽后不应再出现的原文片段
		wantTag string // 期望出现的占位符前缀
	}{
		{"birth_date", "出生日期：1990-01-01。", "1990-01-01", "{{DOB_"},
		{"birth_date_cn", "出生于1985年12月3日", "1985年12月3日", "{{DOB_"},
		{"ip_address", "服务器地址 10.0.0.255 已连接", "10.0.0.255", "{{IP_"},
		{"license_plate", "车牌 京A12345 违章", "京A12345", "{{PLATE_"},
		{"medical_record", "病历号: A1234567", "A1234567", "{{MRN_"},
		{"passport", "护照号 E12345678 有效", "E12345678", "{{PASSPORT_"},
		{"address", "住址：北京市朝阳区建国路88号", "建国路88号", "{{ADDR_"},
		{"age_name", "患者张三，45岁", "45岁", "{{AGE"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := Input{Text: c.text, Level: models.DesensitizationStrict, Metadata: map[string]any{}}
			out, err := stage.Process(ctx, in)
			require.NoError(t, err)
			assert.NotContains(t, out.Text, c.leaked, "原文 PII 应被遮蔽")
			assert.Contains(t, out.Text, c.wantTag, "应产生对应占位符")
		})
	}
}

// TestL1ExtendedRuleStage_RecordsPlaceholders 验证占位符映射仅收录 P2，P3 仅进入实体。
func TestL1ExtendedRuleStage_RecordsPlaceholders(t *testing.T) {
	t.Parallel()
	stage := NewL1ExtendedRuleStage()
	ctx := context.Background()

	in := Input{
		// 逗号隔断，避免 age_name 规则误匹配“年龄 35岁”，仅触发独立的 age 规则（P2）。
		Text:     "IP 192.168.0.1，35岁",
		Level:    models.DesensitizationStrict,
		Metadata: map[string]any{"l1_placeholders": map[string]string{"{{EMAIL_x}}": "a@b.com"}},
	}
	out, err := stage.Process(ctx, in)
	require.NoError(t, err)

	ph, ok := out.Metadata["l1_placeholders"].(map[string]string)
	require.True(t, ok)
	// 保留既有 L1 占位符
	assert.Equal(t, "a@b.com", ph["{{EMAIL_x}}"])
	// P2（年龄）占位符应被收录
	foundP2 := false
	for k, v := range ph {
		if v == "35岁" && len(k) > 0 {
			foundP2 = true
		}
	}
	assert.True(t, foundP2, "P2 年龄占位符应记录在 l1_placeholders")
	// P3（IP）占位符不应出现在映射中
	for _, v := range ph {
		assert.NotEqual(t, "192.168.0.1", v, "P3 IP 占位符不应进入 l1_placeholders")
	}

	ents, ok := out.Metadata["l1_entities"].([]models.SensitiveEntity)
	require.True(t, ok)
	require.Len(t, ents, 2)
	var gotIP, gotAge bool
	for _, e := range ents {
		switch e.Type {
		case "ip_address":
			gotIP = true
			assert.Equal(t, models.P3Confidential, e.Level)
			assert.Equal(t, "192.168.0.1", e.Text)
		case "age":
			gotAge = true
			assert.Equal(t, models.P2Internal, e.Level)
			assert.Equal(t, "35岁", e.Text)
		}
	}
	assert.True(t, gotIP, "实体中应包含 IP")
	assert.True(t, gotAge, "实体中应包含年龄")
}

// TestDeidentifyPipeline_StrictMasksMoreThanStandard 验证 NER 关闭时，
// 严格级遮蔽标准级会泄露的实体，且严格级可见 PII 不多于标准级。
func TestDeidentifyPipeline_StrictMasksMoreThanStandard(t *testing.T) {
	t.Parallel()
	// NER 不可用（透传），仅比较 L1 与 L1.5 的差异。
	l2 := NewL2NERStage(&mockNERDetector{available: false}, &mockNERDetector{available: false})
	p := NewDefaultDeidentifyPipeline(NewL1RuleStage(), l2, NewL1ExtendedRuleStage())
	ctx := context.Background()

	// 该文本含 L1 基础规则未覆盖的 PII：IP、出生日期、车牌、地址。
	text := "我叫王五，家住上海市浦东新区世纪大道100号，车牌沪B88888，出生1988-08-08，服务器 172.16.0.1"

	std, err := p.Execute(ctx, text, models.DesensitizationStandard)
	require.NoError(t, err)
	strict, err := p.Execute(ctx, text, models.DesensitizationStrict)
	require.NoError(t, err)

	// 标准级泄露的具体 PII，在严格级下应被遮蔽。
	for _, leaked := range []string{"172.16.0.1", "1988-08-08", "沪B88888"} {
		assert.Contains(t, std.SafeText, leaked, "标准级应保留（泄露）%s", leaked)
		assert.NotContains(t, strict.SafeText, leaked, "严格级应遮蔽 %s", leaked)
	}

	// 严格级可见 PII 数量不多于标准级。
	assert.LessOrEqual(t, visiblePIICount(strict.SafeText), visiblePIICount(std.SafeText))
}

// visiblePIICount 统计文本中残留的可标识信息片段数量（测试辅助）。
func visiblePIICount(text string) int {
	patterns := []string{
		`(?:19|20)\d{2}[-/.年](?:0?[1-9]|1[0-2])`,
		`(?:(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)\.){3}(?:25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)`,
		`[京津沪渝冀豫云辽黑湘皖鲁新苏浙赣鄂桂甘晋蒙陕吉闽贵粤青藏川宁琼][A-Z][A-HJ-NP-Z0-9]{5,6}`,
	}
	count := 0
	for _, pat := range patterns {
		count += len(regexp.MustCompile(pat).FindAllString(text, -1))
	}
	return count
}
