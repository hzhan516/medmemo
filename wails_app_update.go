package main

import (
	"fmt"
	"time"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// UpdateInfoResponse 前端更新信息响应。
type UpdateInfoResponse struct {
	Version         string `json:"version"`
	DisplayVersion  string `json:"display_version"`
	Name            string `json:"name"`
	Body            string `json:"body"`
	PublishedAt     string `json:"published_at"`
	Mandatory       bool   `json:"mandatory"`
	Channel         string `json:"channel"`
	Prerelease      bool   `json:"prerelease"`
	PrereleaseLabel string `json:"prerelease_label"`
	BuildNumber     string `json:"build_number"`
}

// CheckUpdate 检测是否存在可用更新，供前端主动调用。
func (a *WailsApp) CheckUpdate() (*UpdateInfoResponse, error) {
	if a.updaterSvc == nil {
		return nil, fmt.Errorf("updater service not initialized")
	}

	info, err := a.updaterSvc.CheckUpdate(a.ctx, version)
	if err != nil {
		return nil, fmt.Errorf("failed to check update: %w", err)
	}
	if info == nil {
		return nil, nil
	}

	return &UpdateInfoResponse{
		Version:         info.Version,
		DisplayVersion:  info.DisplayVersion,
		Name:            info.Name,
		Body:            info.Body,
		PublishedAt:     info.PublishedAt.Format(time.RFC3339),
		Mandatory:       info.Mandatory,
		Channel:         string(info.Channel),
		Prerelease:      info.Prerelease,
		PrereleaseLabel: info.PreReleaseLabel,
		BuildNumber:     info.BuildNumber,
	}, nil
}

// DownloadUpdateRequest 下载更新请求。
type DownloadUpdateRequest struct {
	Version string `json:"version"`
}

// DownloadUpdate 下载指定版本的更新包。
// 下载进度通过 Wails Events "update:progress" 推送。
func (a *WailsApp) DownloadUpdate(req DownloadUpdateRequest) (string, error) {
	if a.updaterSvc == nil {
		return "", fmt.Errorf("updater service not initialized")
	}

	// 先重新获取 UpdateInfo（包含正确的下载 URL）
	info, err := a.updaterSvc.GetUpdateInfoByVersion(a.ctx, version, req.Version)
	if err != nil {
		return "", fmt.Errorf("failed to get update info for version %s: %w", req.Version, err)
	}
	if info == nil {
		return "", fmt.Errorf("no update available for version %s", req.Version)
	}

	progressCb := func(downloaded, total int64) {
		runtime.EventsEmit(a.ctx, "update:progress", map[string]int64{
			"downloaded": downloaded,
			"total":      total,
		})
	}

	path, err := a.updaterSvc.DownloadUpdate(a.ctx, info, progressCb)
	if err != nil {
		return "", fmt.Errorf("failed to download update: %w", err)
	}

	return path, nil
}

// ApplyUpdate 应用已下载的更新并启动新版本。
func (a *WailsApp) ApplyUpdate(assetPath string) error {
	if a.updaterSvc == nil {
		return fmt.Errorf("updater service not initialized")
	}

	installedPath, err := a.updaterSvc.ApplyUpdate(assetPath)
	if err != nil {
		return fmt.Errorf("failed to apply update: %w", err)
	}

	if err := restartAfterUpdate(a.ctx, installedPath); err != nil {
		return fmt.Errorf("update installed but failed to restart: %w", err)
	}

	runtime.Quit(a.ctx)
	return nil
}

// UpdateSettingsResponse 更新设置响应。
type UpdateSettingsResponse struct {
	CheckEnabled bool   `json:"check_enabled"`
	Channel      string `json:"channel"`
	SkipVersion  string `json:"skip_version"`
}

// GetUpdateSettings 获取当前更新设置。
func (a *WailsApp) GetUpdateSettings() (*UpdateSettingsResponse, error) {
	if a.updaterSvc == nil {
		return nil, fmt.Errorf("updater service not initialized")
	}

	s := a.updaterSvc.GetSettings()
	return &UpdateSettingsResponse{
		CheckEnabled: s.CheckEnabled,
		Channel:      string(s.Channel),
		SkipVersion:  s.SkipVersion,
	}, nil
}

// SetUpdateSettings 保存更新设置。
func (a *WailsApp) SetUpdateSettings(req UpdateSettingsResponse) error {
	if a.updaterSvc == nil {
		return fmt.Errorf("updater service not initialized")
	}

	s := &entity.UpdateSettings{
		CheckEnabled: req.CheckEnabled,
		Channel:      models.UpdateChannel(req.Channel),
		SkipVersion:  req.SkipVersion,
	}
	a.updaterSvc.SetSettings(s)
	return nil
}

// SkipUpdateVersion 标记跳过指定版本。
func (a *WailsApp) SkipUpdateVersion(v string) error {
	if a.updaterSvc == nil {
		return fmt.Errorf("updater service not initialized")
	}
	a.updaterSvc.SkipVersion(v)
	return nil
}

// OpenDownloadURL 通过系统浏览器打开指定 URL。
func (a *WailsApp) OpenDownloadURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}
