package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustWriteRules 将规则库 JSON 写入临时文件并返回路径。
func mustWriteRules(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "rules.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

// defaultRules 返回包含全部四级规则的测试规则库。
func defaultRules() string {
	return `{
  "version": "test-1.0.0",
  "updated_at": "2026-05-18",
  "rules": [
    {
      "id": "l1-diag-definite-001",
      "level": "L1",
      "name": "确诊性诊断结论",
      "patterns": ["你患有[一-龥]+病", "确诊为[一-龥]+"],
      "action": "block",
      "replacement": "BLOCKED_TEXT"
    },
    {
      "id": "l1-drug-dose-001",
      "level": "L1",
      "name": "药物剂量处方",
      "patterns": ["服用\\s*\\d+\\s*毫克"],
      "action": "block",
      "replacement": "DRUG_BLOCKED"
    },
    {
      "id": "l1-surgery-001",
      "level": "L1",
      "name": "手术建议",
      "patterns": ["建议手术", "需要开刀"],
      "action": "block"
    },
    {
      "id": "l2-diag-implied-001",
      "level": "L2",
      "name": "暗示性诊断",
      "patterns": ["可能是[一-龥]+病", "不排除[一-龥]+"],
      "action": "warn",
      "warning": "WARN_IMPLIED"
    },
    {
      "id": "l2-drug-otc-001",
      "level": "L2",
      "name": "非处方药推荐",
      "patterns": ["可以买[一-龥]+药", "建议服用[一-龥]+"],
      "action": "warn",
      "warning": "WARN_DRUG"
    },
    {
      "id": "l2-check-suggest-001",
      "level": "L2",
      "name": "检查项目建议",
      "patterns": ["建议查一下[一-龥]+", "最好做个[一-龥]+检查"],
      "action": "warn",
      "warning": "WARN_CHECK"
    },
    {
      "id": "l3-disease-severe-001",
      "level": "L3",
      "name": "严重疾病科普",
      "patterns": ["癌症", "白血病", "艾滋病"],
      "action": "notice",
      "notice": "NOTICE_SEVERE"
    },
    {
      "id": "l3-disease-chronic-001",
      "level": "L3",
      "name": "重大慢性病科普",
      "patterns": ["糖尿病并发症", "高血压危象", "冠心病"],
      "action": "notice",
      "notice": "NOTICE_CHRONIC"
    }
  ]
}`
}

// TestL1Blocked_DefiniteDiagnosis 验证确诊性诊断被阻断。
func TestL1Blocked_DefiniteDiagnosis(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, defaultRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.Evaluate(context.Background(), "你患有糖尿病，需要长期服药控制。")
	require.NoError(t, err)
	assert.True(t, res.Blocked)
	assert.Equal(t, L1Blocked.String(), res.Level)
	assert.Equal(t, "l1-diag-definite-001", res.MatchedRule)
	assert.Equal(t, "BLOCKED_TEXT", res.SafeText)
}

// TestL1Blocked_DrugDose 验证药物剂量被阻断。
func TestL1Blocked_DrugDose(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, defaultRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.Evaluate(context.Background(), "建议服用 500 毫克")
	require.NoError(t, err)
	assert.True(t, res.Blocked)
	assert.Equal(t, L1Blocked.String(), res.Level)
	assert.Equal(t, "l1-drug-dose-001", res.MatchedRule)
	assert.Equal(t, "DRUG_BLOCKED", res.SafeText)
}

// TestL1Blocked_Surgery 验证手术建议被阻断。
func TestL1Blocked_Surgery(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, defaultRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.Evaluate(context.Background(), "根据情况，建议手术切除。")
	require.NoError(t, err)
	assert.True(t, res.Blocked)
	assert.Equal(t, L1Blocked.String(), res.Level)
	assert.Equal(t, "l1-surgery-001", res.MatchedRule)
}

// TestL1Blocked_DefaultReplacement 验证无 replacement 时使用默认阻断文案。
func TestL1Blocked_DefaultReplacement(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, defaultRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.Evaluate(context.Background(), "需要开刀治疗")
	require.NoError(t, err)
	assert.True(t, res.Blocked)
	assert.Contains(t, res.SafeText, "无法提供医疗诊断")
}

// TestL2Warning_ImpliedDiagnosis 验证暗示性诊断触发警告。
func TestL2Warning_ImpliedDiagnosis(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, defaultRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.Evaluate(context.Background(), "你可能是感冒病了，多注意休息。")
	require.NoError(t, err)
	assert.False(t, res.Blocked)
	assert.Equal(t, L2Warning.String(), res.Level)
	assert.Equal(t, "l2-diag-implied-001", res.MatchedRule)
	assert.Equal(t, "WARN_IMPLIED", res.Warning)
	assert.Equal(t, "你可能是感冒病了，多注意休息。", res.SafeText)
}

// TestL2Warning_DrugOTC 验证非处方药推荐触发警告。
func TestL2Warning_DrugOTC(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, defaultRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.Evaluate(context.Background(), "建议服用布洛芬缓解疼痛。")
	require.NoError(t, err)
	assert.False(t, res.Blocked)
	assert.Equal(t, L2Warning.String(), res.Level)
	assert.Equal(t, "l2-drug-otc-001", res.MatchedRule)
	assert.Equal(t, "WARN_DRUG", res.Warning)
}

// TestL2Warning_CheckSuggest 验证检查项目建议触发警告。
func TestL2Warning_CheckSuggest(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, defaultRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.Evaluate(context.Background(), "最好做个血常规检查看看。")
	require.NoError(t, err)
	assert.False(t, res.Blocked)
	assert.Equal(t, L2Warning.String(), res.Level)
	assert.Equal(t, "l2-check-suggest-001", res.MatchedRule)
	assert.Equal(t, "WARN_CHECK", res.Warning)
}

// TestL3Notice_SevereDisease 验证严重疾病科普触发提示。
func TestL3Notice_SevereDisease(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, defaultRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.Evaluate(context.Background(), "白血病的早期症状包括持续发热和出血倾向。")
	require.NoError(t, err)
	assert.False(t, res.Blocked)
	assert.Equal(t, L3Notice.String(), res.Level)
	assert.Equal(t, "l3-disease-severe-001", res.MatchedRule)
	assert.Equal(t, "NOTICE_SEVERE", res.Notice)
}

// TestL3Notice_ChronicDisease 验证重大慢性病科普触发提示。
func TestL3Notice_ChronicDisease(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, defaultRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.Evaluate(context.Background(), "糖尿病并发症包括视网膜病变和肾病。")
	require.NoError(t, err)
	assert.False(t, res.Blocked)
	assert.Equal(t, L3Notice.String(), res.Level)
	assert.Equal(t, "l3-disease-chronic-001", res.MatchedRule)
	assert.Equal(t, "NOTICE_CHRONIC", res.Notice)
}

// TestL4Normal_HealthTips 验证一般健康科普正常放行。
func TestL4Normal_HealthTips(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, defaultRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	text := "保持规律作息和适度运动有助于提升免疫力。"
	res, err := ci.Evaluate(context.Background(), text)
	require.NoError(t, err)
	assert.False(t, res.Blocked)
	assert.Equal(t, L4Normal.String(), res.Level)
	assert.Equal(t, "", res.MatchedRule)
	assert.Equal(t, text, res.SafeText)
}

// TestPriority_L1BeforeL2 验证 L1 优先级高于 L2，先命中 L1 则短路返回。
func TestPriority_L1BeforeL2(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, defaultRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	// 同时包含 L1 和 L2 关键词，应返回 L1
	res, err := ci.Evaluate(context.Background(), "你患有高血压病，可能是遗传因素导致。")
	require.NoError(t, err)
	assert.True(t, res.Blocked)
	assert.Equal(t, L1Blocked.String(), res.Level)
	assert.Equal(t, "l1-diag-definite-001", res.MatchedRule)
}

// TestFallback_SafeOnLoadError 验证规则库加载失败时返回 nil + error。
func TestFallback_SafeOnLoadError(t *testing.T) {
	t.Parallel()
	ci, err := NewComplianceInterceptor("/nonexistent/path/rules.json")
	// 加载失败强制返回 nil，避免调用者忽略 error 后使用空规则集
	require.Error(t, err)
	require.Nil(t, ci)
}

// TestConfigError_L1InlineEmptyReplacement 验证 L1 inline 规则 replacement 为空时拒绝加载。
func TestConfigError_L1InlineEmptyReplacement(t *testing.T) {
	t.Parallel()
	rules := `{
  "version": "test-invalid",
  "rules": [
    {
      "id": "l1-inline-empty",
      "level": "L1",
      "name": "空替换测试",
      "patterns": ["你患有([一-龥]+)"],
      "action": "block",
      "replace_mode": "inline",
      "replacement": ""
    }
  ]
}`
	path := mustWriteRules(t, rules)
	ci, err := NewComplianceInterceptor(path)
	require.Error(t, err)
	require.Nil(t, ci)
	assert.Contains(t, err.Error(), "L1 inline replacement cannot be empty")
}

// TestConfigError_L1InlineWhitespaceReplacement 验证 L1 inline 规则 replacement 仅含空白时拒绝加载。
func TestConfigError_L1InlineWhitespaceReplacement(t *testing.T) {
	t.Parallel()
	rules := `{
  "version": "test-invalid",
  "rules": [
    {
      "id": "l1-inline-ws",
      "level": "L1",
      "name": "空白替换测试",
      "patterns": ["你患有([一-龥]+)"],
      "action": "block",
      "replace_mode": "inline",
      "replacement": "   "
    }
  ]
}`
	path := mustWriteRules(t, rules)
	ci, err := NewComplianceInterceptor(path)
	require.Error(t, err)
	require.Nil(t, ci)
	assert.Contains(t, err.Error(), "L1 inline replacement cannot be empty")
}

// TestConfigError_L1BlockEmptyReplacementAllowed 验证非 inline 的 L1 规则允许空 replacement（使用默认文案）。
func TestConfigError_L1BlockEmptyReplacementAllowed(t *testing.T) {
	t.Parallel()
	rules := `{
  "version": "test-ok",
  "rules": [
    {
      "id": "l1-block-empty",
      "level": "L1",
      "name": "整段阻断测试",
      "patterns": ["建议手术"],
      "action": "block"
    }
  ]
}`
	path := mustWriteRules(t, rules)
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)
	require.NotNil(t, ci)

	res, err := ci.Evaluate(context.Background(), "建议手术切除")
	require.NoError(t, err)
	assert.True(t, res.Blocked)
	assert.Contains(t, res.SafeText, "无法提供医疗诊断")
}

// TestFallback_SafeOnEmptyRules 验证空规则库时直接放行。
func TestFallback_SafeOnEmptyRules(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, `{"version":"empty","rules":[]}`)
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.Evaluate(context.Background(), "任何内容")
	require.NoError(t, err)
	assert.False(t, res.Blocked)
	assert.Equal(t, L4Normal.String(), res.Level)
}

// TestRuleHotReload 验证运行时热更新规则库。
func TestRuleHotReload(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "rules.json")

	// 初始规则库：无 L1 规则
	v1 := `{"version":"1.0","rules":[]}`
	require.NoError(t, os.WriteFile(path, []byte(v1), 0644))

	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.Evaluate(context.Background(), "你患有糖尿病。")
	require.NoError(t, err)
	assert.Equal(t, L4Normal.String(), res.Level)

	// 热更新：加入 L1 规则
	v2 := `{"version":"2.0","rules":[{"id":"l1-test","level":"L1","name":"测试","patterns":["你患有"],"action":"block","replacement":"RELOADED"}]}`
	require.NoError(t, os.WriteFile(path, []byte(v2), 0644))
	require.NoError(t, ci.ReloadRules())

	res, err = ci.Evaluate(context.Background(), "你患有糖尿病。")
	require.NoError(t, err)
	assert.True(t, res.Blocked)
	assert.Equal(t, "RELOADED", res.SafeText)
	assert.Equal(t, "2.0", ci.Version())
}

// TestPerformance_SingleDetection 验证单条检测延迟 <10ms。
func TestPerformance_SingleDetection(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, defaultRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	text := "你患有糖尿病，建议服用 500 毫克二甲双胍，可能需要手术。"

	// 预热
	_, _ = ci.Evaluate(context.Background(), text)

	start := time.Now()
	for i := 0; i < 100; i++ {
		_, err := ci.Evaluate(context.Background(), text)
		require.NoError(t, err)
	}
	elapsed := time.Since(start)
	avg := elapsed / 100

	t.Logf("Average detection latency: %v", avg)
	assert.Less(t, avg, 10*time.Millisecond, "single detection should be <10ms")
}

// TestEvaluateWithTimeout 验证带超时的评估在异常场景下安全降级。
func TestEvaluateWithTimeout(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, defaultRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.EvaluateWithTimeout(context.Background(), "一般健康建议", 5*time.Second)
	require.NoError(t, err)
	assert.Equal(t, L4Normal.String(), res.Level)
}

// TestVersion 验证规则库版本号读取。
func TestVersion(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, defaultRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)
	assert.Equal(t, "test-1.0.0", ci.Version())
}

// ========== EvaluateWithInlineReplace 测试 ==========

// inlineRules 返回包含 inline 替换规则的测试规则库。
func inlineRules() string {
	return `{
  "version": "test-inline-1.0.0",
  "updated_at": "2026-05-18",
  "rules": [
    {
      "id": "l1-inline-diag-001",
      "level": "L1",
      "name": "确定性诊断-inline-你患有",
      "patterns": ["你患有([一-龥]+)"],
      "action": "block",
      "replace_mode": "inline",
      "replacement": "您的情况可能与${1}有关，建议咨询医生确认"
    },
    {
      "id": "l1-inline-diag-002",
      "level": "L1",
      "name": "确定性诊断-inline-确诊为",
      "patterns": ["确诊为([一-龥]+)"],
      "action": "block",
      "replace_mode": "inline",
      "replacement": "检查结果可能与${1}有关，需由医生面诊后确认"
    },
    {
      "id": "l1-block-drug-001",
      "level": "L1",
      "name": "药物剂量处方",
      "patterns": ["服用\\s*\\d+\\s*毫克"],
      "action": "block",
      "replacement": "DRUG_BLOCKED"
    },
    {
      "id": "l2-diag-implied-001",
      "level": "L2",
      "name": "暗示性诊断",
      "patterns": ["可能是[一-龥]+病"],
      "action": "warn",
      "warning": "WARN_IMPLIED"
    },
    {
      "id": "l3-disease-001",
      "level": "L3",
      "name": "严重疾病",
      "patterns": ["癌症"],
      "action": "notice",
      "notice": "NOTICE_SEVERE"
    }
  ]
}`
}

// TestInlineReplace_SingleTerm 验证单条 inline 替换在文中局部生效。
func TestInlineReplace_SingleTerm(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, inlineRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.EvaluateWithInlineReplace(context.Background(), "根据描述，你患有高血压，建议定期监测血压。")
	require.NoError(t, err)
	// inline 替换后文本不再命中其他规则，正常放行
	assert.Equal(t, L4Normal.String(), res.Level)
	assert.False(t, res.Blocked)
	assert.Contains(t, res.SafeText, "您的情况可能与高血压有关，建议咨询医生确认")
	assert.Contains(t, res.SafeText, "建议定期监测血压")
	assert.Contains(t, res.ReplacedTerms, "l1-inline-diag-001")
}

// TestInlineReplace_MultipleTerms 验证多条 inline 规则依次替换。
func TestInlineReplace_MultipleTerms(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, inlineRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.EvaluateWithInlineReplace(context.Background(), "你患有糖尿病，确诊为二型糖尿病。")
	require.NoError(t, err)
	// inline 替换后文本不再命中其他规则，正常放行
	assert.Equal(t, L4Normal.String(), res.Level)
	assert.False(t, res.Blocked)
	// 两条 inline 规则都应该被应用
	assert.Contains(t, res.SafeText, "您的情况可能与糖尿病有关，建议咨询医生确认")
	assert.Contains(t, res.SafeText, "检查结果可能与二型糖尿病有关，需由医生面诊后确认")
	assert.Len(t, res.ReplacedTerms, 2)
}

// TestInlineReplace_ThenL1Block 验证 inline 替换后仍命中整段阻断规则。
func TestInlineReplace_ThenL1Block(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, inlineRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	// 文本同时包含 inline 替换内容和整段阻断内容
	res, err := ci.EvaluateWithInlineReplace(context.Background(), "你患有高血压，建议服用 500 毫克降压药。")
	require.NoError(t, err)
	// 药物剂量整段阻断优先级高于 inline 替换结果
	assert.Equal(t, L1Blocked.String(), res.Level)
	assert.True(t, res.Blocked)
	assert.Equal(t, "DRUG_BLOCKED", res.SafeText)
	// 但 inline 替换仍记录在 ReplacedTerms 中
	assert.Contains(t, res.ReplacedTerms, "l1-inline-diag-001")
}

// TestInlineReplace_NoMatch 验证未命中 inline 规则时正常放行。
func TestInlineReplace_NoMatch(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, inlineRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.EvaluateWithInlineReplace(context.Background(), "保持规律作息有助于健康。")
	require.NoError(t, err)
	assert.Equal(t, L4Normal.String(), res.Level)
	assert.False(t, res.Blocked)
	assert.Len(t, res.ReplacedTerms, 0)
}

// TestInlineReplace_L2AfterReplace 验证 inline 替换后无其他规则命中时正常放行。
func TestInlineReplace_L2AfterReplace(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, inlineRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	// inline 替换消除了 L1 命中，替换后文本无 L2/L3 内容，应返回 L4
	res, err := ci.EvaluateWithInlineReplace(context.Background(), "你患有高血压的可能性不大。")
	require.NoError(t, err)
	assert.Equal(t, L4Normal.String(), res.Level)
	assert.False(t, res.Blocked)
}

// TestInlineReplace_L3StillWorks 验证 inline 替换不影响 L3 提示。
func TestInlineReplace_L3StillWorks(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, inlineRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.EvaluateWithInlineReplace(context.Background(), "癌症的早期筛查很重要。")
	require.NoError(t, err)
	assert.Equal(t, L3Notice.String(), res.Level)
	assert.False(t, res.Blocked)
	assert.Equal(t, "NOTICE_SEVERE", res.Notice)
}

// TestInlineReplace_EmptyRules 验证空规则库时 inline 替换直接放行。
func TestInlineReplace_EmptyRules(t *testing.T) {
	t.Parallel()
	path := mustWriteRules(t, `{"version":"empty","rules":[]}`)
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.EvaluateWithInlineReplace(context.Background(), "你患有糖尿病。")
	require.NoError(t, err)
	assert.Equal(t, L4Normal.String(), res.Level)
	assert.Len(t, res.ReplacedTerms, 0)
}

// TestInlineReplace_CaptureGroup 验证正则捕获组在替换中正确引用。
func TestInlineReplace_CaptureGroup(t *testing.T) {
	t.Parallel()
	rules := `{
  "version": "test-capture",
  "rules": [
    {
      "id": "l1-capture-test",
      "level": "L1",
      "name": "捕获组测试",
      "patterns": ["确诊为([一-龥]+)"],
      "action": "block",
      "replace_mode": "inline",
      "replacement": "可能为${1}，建议咨询医生"
    }
  ]
}`
	path := mustWriteRules(t, rules)
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.EvaluateWithInlineReplace(context.Background(), "确诊为糖尿病")
	require.NoError(t, err)
	assert.Equal(t, "可能为糖尿病，建议咨询医生", res.SafeText)
}
