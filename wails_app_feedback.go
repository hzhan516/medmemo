package main

import (
	"github.com/hzhan516/medmemo/internal/application/feedback"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// GetVersion 返回当前应用版本号（构建时通过 -ldflags 注入）。
func (a *WailsApp) GetVersion() string {
	return version
}

// VersionInfoResponse 当前应用版本结构化信息，供 About 页与状态栏展示。
type VersionInfoResponse struct {
	Version         string `json:"version"`
	DisplayVersion  string `json:"display_version"`
	BuildNumber     string `json:"build_number"`
	Channel         string `json:"channel"`
	PrereleaseLabel string `json:"prerelease_label"`
	Prerelease      bool   `json:"prerelease"`
}

// GetVersionInfo 解析当前版本号为结构化信息。
func (a *WailsApp) GetVersionInfo() (*VersionInfoResponse, error) {
	v := models.ParseAppVersion(version)
	return &VersionInfoResponse{
		Version:         v.Version,
		DisplayVersion:  v.DisplayVersion,
		BuildNumber:     v.BuildNumber,
		Channel:         string(v.Channel),
		PrereleaseLabel: v.PrereleaseLabel,
		Prerelease:      v.Prerelease,
	}, nil
}

// CollectSystemInfo 收集当前运行环境信息，供前端展示。
func (a *WailsApp) CollectSystemInfo() (*feedback.SystemInfo, error) {
	reporter := feedback.NewReporter(version, buildTime)
	return reporter.Collect(), nil
}

// OpenGitHubIssue 打开系统浏览器到 GitHub Issue 创建页面，预填日志内容。
// 前端调用后，用户只需在浏览器中点击 Submit 即可创建 Issue。
func (a *WailsApp) OpenGitHubIssue(userDescription string, errorLog string) error {
	reporter := feedback.NewReporter(version, buildTime)
	info := reporter.Collect()

	logContent, err := feedback.ReadAppLogFile("")
	if err != nil {
		// 日志读取失败不影响主流程，仅记录
		logContent = ""
	}

	// 合并显式传入的错误日志与本地日志文件
	combinedLog := errorLog
	if logContent != "" {
		if combinedLog != "" {
			combinedLog += "\n\n--- 本地日志 ---\n" + logContent
		} else {
			combinedLog = logContent
		}
	}

	issueURL := reporter.BuildIssueURL(info, userDescription, combinedLog)
	runtime.BrowserOpenURL(a.ctx, issueURL)
	return nil
}

// GetVersionNotes 返回全部版本提示数据，按版本降序排列（最新在前）。
func (a *WailsApp) GetVersionNotes() []entity.VersionNote {
	notes := make([]entity.VersionNote, len(entity.AllVersionNotes))
	copy(notes, entity.AllVersionNotes)
	// 倒序：最新版本在前
	for i, j := 0, len(notes)-1; i < j; i, j = i+1, j-1 {
		notes[i], notes[j] = notes[j], notes[i]
	}
	return notes
}
