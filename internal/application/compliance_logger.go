// Package application 实现应用用例层，编排领域对象完成完整业务流程。
package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ComplianceLogEntry 单条合规拦截日志条目。
// 不记录原始文本内容，仅记录长度和哈希，保护用户隐私。
type ComplianceLogEntry struct {
	Timestamp   string `json:"ts"`
	RuleID      string `json:"rule_id"`
	Level       string `json:"level"`
	OriginalLen int    `json:"orig_len"`
	ReplacedLen int    `json:"repl_len"`
	TextHash    string `json:"hash"` // SHA-256 前 6 位
}

// ComplianceFeedbackEntry 单条用户申诉反馈日志条目。
type ComplianceFeedbackEntry struct {
	Timestamp string `json:"ts"`
	RuleID    string `json:"rule_id"`
	TextHash  string `json:"text_hash"`
	Feedback  string `json:"feedback_type"`
}

// ComplianceLogger 合规拦截脱敏日志记录器。
// 以 JSON Lines 格式追加写入本地文件，单文件超过阈值时自动轮转。
type ComplianceLogger struct {
	logPath      string
	feedbackPath string
	mu           sync.Mutex
	maxSize      int64 // 单文件大小阈值（字节）
}

// NewComplianceLogger 创建合规日志记录器。
// logDir 为日志目录，若不存在则自动创建。
// 当 logDir 为相对路径且 MEDMEMO_DATA_DIR 环境变量已设置时，以前者为前缀。
func NewComplianceLogger(logDir string) *ComplianceLogger {
	if logDir == "" {
		logDir = "data"
	}
	if !filepath.IsAbs(logDir) {
		if baseDir := os.Getenv("MEDMEMO_DATA_DIR"); baseDir != "" {
			logDir = filepath.Join(baseDir, logDir)
		}
	}
	return &ComplianceLogger{
		logPath:      filepath.Join(logDir, "compliance_logs.jsonl"),
		feedbackPath: filepath.Join(logDir, "compliance_feedback.jsonl"),
		maxSize:      10 * 1024 * 1024, // 10MB
	}
}

// Log 记录一次合规拦截事件。
// 原始文本仅计算长度和哈希，不保存内容。
func (l *ComplianceLogger) Log(ctx context.Context, ruleID string, originalText, replacedText, level string) error {
	entry := ComplianceLogEntry{
		Timestamp:   time.Now().Format(time.RFC3339),
		RuleID:      ruleID,
		Level:       level,
		OriginalLen: len(originalText),
		ReplacedLen: len(replacedText),
		TextHash:    shortHash(originalText),
	}
	return l.appendLog(l.logPath, entry)
}

// LogFeedback 记录一次用户申诉反馈。
func (l *ComplianceLogger) LogFeedback(ctx context.Context, ruleID, originalText, feedbackType string) error {
	entry := ComplianceFeedbackEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		RuleID:    ruleID,
		TextHash:  shortHash(originalText),
		Feedback:  feedbackType,
	}
	return l.appendLog(l.feedbackPath, entry)
}

// appendLog 将日志条目追加到指定文件，并在超过大小阈值时自动轮转。
func (l *ComplianceLogger) appendLog(path string, entry any) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory %s: %w", dir, err)
	}

	// 检查是否需要轮转
	if info, err := os.Stat(path); err == nil && info.Size() > l.maxSize {
		if err := l.rotate(path); err != nil {
			return fmt.Errorf("failed to rotate log file: %w", err)
		}
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", path, err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}
	return nil
}

// rotate 将当前日志文件重命名为带时间戳的备份文件。
func (l *ComplianceLogger) rotate(path string) error {
	timestamp := time.Now().Format("20060102_150405")
	ext := filepath.Ext(path)
	base := path[:len(path)-len(ext)]
	backupPath := fmt.Sprintf("%s_%s%s", base, timestamp, ext)
	return os.Rename(path, backupPath)
}

// shortHash 计算文本的 SHA-256 摘要并取前 6 位十六进制字符。
func shortHash(text string) string {
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:])[:6]
}
