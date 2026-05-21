package application

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComplianceLogger_Log 验证拦截日志记录。
func TestComplianceLogger_Log(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewComplianceLogger(tmpDir)

	ctx := context.Background()
	err := logger.Log(ctx, "l1-diag-001", "你患有糖尿病", "建议咨询医生", "L1_BLOCKED")
	require.NoError(t, err)

	// 读取日志文件验证内容
	logPath := filepath.Join(tmpDir, "compliance_logs.jsonl")
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)

	var entry ComplianceLogEntry
	require.NoError(t, json.Unmarshal(data, &entry))

	assert.Equal(t, "l1-diag-001", entry.RuleID)
	assert.Equal(t, "L1_BLOCKED", entry.Level)
	assert.Equal(t, 18, entry.OriginalLen) // "你患有糖尿病" 为 18 字节（UTF-8，6 个汉字 × 3）
	assert.Equal(t, 18, entry.ReplacedLen) // "建议咨询医生" 为 18 字节（6 个汉字 × 3）
	assert.NotEmpty(t, entry.TextHash)
	assert.Len(t, entry.TextHash, 6)
	assert.NotEmpty(t, entry.Timestamp)
}

// TestComplianceLogger_LogFeedback 验证用户申诉反馈记录。
func TestComplianceLogger_LogFeedback(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewComplianceLogger(tmpDir)

	ctx := context.Background()
	err := logger.LogFeedback(ctx, "l2-diag-001", "可能是感冒", "false_positive")
	require.NoError(t, err)

	feedbackPath := filepath.Join(tmpDir, "compliance_feedback.jsonl")
	data, err := os.ReadFile(feedbackPath)
	require.NoError(t, err)

	var entry ComplianceFeedbackEntry
	require.NoError(t, json.Unmarshal(data, &entry))

	assert.Equal(t, "l2-diag-001", entry.RuleID)
	assert.Equal(t, "false_positive", entry.Feedback)
	assert.NotEmpty(t, entry.TextHash)
	assert.Len(t, entry.TextHash, 6)
}

// TestComplianceLogger_PrivacyProtection 验证日志不包含原始文本内容。
func TestComplianceLogger_PrivacyProtection(t *testing.T) {
	tmpDir := t.TempDir()
	logger := NewComplianceLogger(tmpDir)

	ctx := context.Background()
	sensitiveText := "我叫张三，身份证号 110101199001011234，电话 13800138000"
	err := logger.Log(ctx, "l1-pii-001", sensitiveText, "已替换", "L1_BLOCKED")
	require.NoError(t, err)

	logPath := filepath.Join(tmpDir, "compliance_logs.jsonl")
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)

	// 日志内容中不应出现原始敏感文本
	logContent := string(data)
	assert.NotContains(t, logContent, "张三")
	assert.NotContains(t, logContent, "110101")
	assert.NotContains(t, logContent, "13800138000")
	// 但应包含长度和哈希信息
	assert.Contains(t, logContent, "hash")
}

// TestComplianceLogger_Rotation 验证日志文件自动轮转。
func TestComplianceLogger_Rotation(t *testing.T) {
	tmpDir := t.TempDir()
	logger := &ComplianceLogger{
		logPath:      filepath.Join(tmpDir, "compliance_logs.jsonl"),
		feedbackPath: filepath.Join(tmpDir, "compliance_feedback.jsonl"),
		maxSize:      100, // 设置很小的阈值以便触发轮转
	}

	ctx := context.Background()
	// 写入超过阈值的日志
	for i := 0; i < 10; i++ {
		err := logger.Log(ctx, "l1-test", strings.Repeat("x", 20), strings.Repeat("y", 20), "L1_BLOCKED")
		require.NoError(t, err)
	}

	// 验证原文件被轮转
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)

	var hasBackup bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "compliance_logs_") && strings.HasSuffix(e.Name(), ".jsonl") {
			hasBackup = true
		}
	}
	assert.True(t, hasBackup, "log file should be rotated")
}

// TestComplianceLogger_DefaultPath 验证默认日志路径。
func TestComplianceLogger_DefaultPath(t *testing.T) {
	logger := NewComplianceLogger("")
	assert.Contains(t, logger.logPath, "data/compliance_logs.jsonl")
	assert.Contains(t, logger.feedbackPath, "data/compliance_feedback.jsonl")
}

// TestShortHash 验证短哈希函数。
func TestShortHash(t *testing.T) {
	h1 := shortHash("hello")
	h2 := shortHash("hello")
	h3 := shortHash("world")

	assert.Len(t, h1, 6)
	assert.Equal(t, h1, h2)
	assert.NotEqual(t, h1, h3)
}
