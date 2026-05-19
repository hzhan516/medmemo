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

// TestFallback_SafeOnLoadError 验证规则库加载失败时降级放行。
func TestFallback_SafeOnLoadError(t *testing.T) {
	ci, err := NewComplianceInterceptor("/nonexistent/path/rules.json")
	// 加载失败时返回 error，但仍返回可用的拦截器实例（降级策略）
	require.Error(t, err)
	require.NotNil(t, ci)

	res, err := ci.Evaluate(context.Background(), "你患有糖尿病。")
	require.NoError(t, err)
	assert.False(t, res.Blocked)
	assert.Equal(t, L4Normal.String(), res.Level)
}

// TestFallback_SafeOnEmptyRules 验证空规则库时直接放行。
func TestFallback_SafeOnEmptyRules(t *testing.T) {
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
	path := mustWriteRules(t, defaultRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)

	res, err := ci.EvaluateWithTimeout(context.Background(), "一般健康建议", 5*time.Second)
	require.NoError(t, err)
	assert.Equal(t, L4Normal.String(), res.Level)
}

// TestVersion 验证规则库版本号读取。
func TestVersion(t *testing.T) {
	path := mustWriteRules(t, defaultRules())
	ci, err := NewComplianceInterceptor(path)
	require.NoError(t, err)
	assert.Equal(t, "test-1.0.0", ci.Version())
}
