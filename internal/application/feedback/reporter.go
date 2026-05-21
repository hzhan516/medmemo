// Package feedback 实现问题反馈与日志上报功能。
// 遵循隐私优先原则：不收集对话内容、健康信息或敏感凭据。
package feedback

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	githubIssueBaseURL = "https://github.com/medmemo/medmemo/issues/new"
	maxLogLines        = 100
)

// SystemInfo 系统与运行环境信息。
// 所有字段均为非敏感的技术信息。
type SystemInfo struct {
	AppVersion string `json:"app_version"`
	GoVersion  string `json:"go_version"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	BuildTime  string `json:"build_time,omitempty"`
}

// Reporter 日志反馈报告生成器。
type Reporter struct {
	appVersion string
	buildTime  string
}

// NewReporter 创建报告生成器。
// appVersion 和 buildTime 由构建时通过 -ldflags 注入的主包变量传入。
func NewReporter(appVersion, buildTime string) *Reporter {
	return &Reporter{
		appVersion: appVersion,
		buildTime:  buildTime,
	}
}

// Collect 收集当前系统信息。
func (r *Reporter) Collect() *SystemInfo {
	return &SystemInfo{
		AppVersion: r.appVersion,
		GoVersion:  runtime.Version(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		BuildTime:  r.buildTime,
	}
}

// BuildIssueURL 根据系统信息和用户描述构建 GitHub Issue 预填 URL。
// 返回的 URL 可直接用浏览器打开，title 和 body 已预填。
// 用户只需在浏览器中点击 Submit 即可创建 Issue。
func (r *Reporter) BuildIssueURL(info *SystemInfo, userDesc, errorLog string) string {
	body := r.buildIssueBody(info, userDesc, errorLog)

	params := url.Values{}
	params.Set("title", fmt.Sprintf("[Bug] MedMemo %s on %s/%s", info.AppVersion, info.OS, info.Arch))
	params.Set("body", body)

	return githubIssueBaseURL + "?" + params.Encode()
}

// buildIssueBody 生成 Issue Markdown 正文。
func (r *Reporter) buildIssueBody(info *SystemInfo, userDesc, errorLog string) string {
	var b strings.Builder

	b.WriteString("## 问题描述\n\n")
	if userDesc != "" {
		b.WriteString(userDesc)
	} else {
		b.WriteString("（用户未提供描述）")
	}
	b.WriteString("\n\n")

	b.WriteString("## 系统信息\n\n")
	b.WriteString(fmt.Sprintf("- **App 版本**: %s\n", info.AppVersion))
	b.WriteString(fmt.Sprintf("- **Go 版本**: %s\n", info.GoVersion))
	b.WriteString(fmt.Sprintf("- **操作系统**: %s\n", info.OS))
	b.WriteString(fmt.Sprintf("- **系统架构**: %s\n", info.Arch))
	if info.BuildTime != "" {
		b.WriteString(fmt.Sprintf("- **构建时间**: %s\n", info.BuildTime))
	}
	b.WriteString("\n")

	if errorLog != "" {
		b.WriteString("## 错误日志\n\n")
		b.WriteString("```\n")
		b.WriteString(sanitizeLog(errorLog))
		b.WriteString("\n```\n\n")
	}

	b.WriteString("---\n")
	b.WriteString("*此 Issue 由 MedMemo 自动生成的反馈链接创建，不包含任何对话内容或个人健康信息。*")

	return b.String()
}

// ReadAppLogFile 读取最近的本地应用日志文件内容（最多 maxLogLines 行）。
// 如果日志文件不存在，返回空字符串，不报错。
func ReadAppLogFile(logDir string) (string, error) {
	if logDir == "" {
		logDir = "data"
	}
	logPath := filepath.Join(logDir, "app.log")

	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read log file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) > maxLogLines {
		lines = lines[len(lines)-maxLogLines:]
	}

	return sanitizeLog(strings.Join(lines, "\n")), nil
}

// sanitizeLog 对日志内容进行脱敏处理：
// 1. 替换用户主目录路径为 ~
// 2. 移除潜在的 API key 模式（简单启发式）
func sanitizeLog(input string) string {
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		input = strings.ReplaceAll(input, homeDir, "~")
	}

	// 简单启发式：替换看起来像 Bearer token 或 api_key 的值
	// 保留键名，替换值为 [REDACTED]
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "api_key") || strings.Contains(lower, "apikey") ||
			strings.Contains(lower, "bearer") || strings.Contains(lower, "authorization") ||
			strings.Contains(lower, "token") || strings.Contains(lower, "password") ||
			strings.Contains(lower, "secret") {
			// 对包含敏感关键词的行进行整体脱敏
			lines[i] = "[REDACTED: potential credential]"
		}
	}

	return strings.Join(lines, "\n")
}
