// Package updater 提供更新检测与应用服务。
// 封装版本比较、更新决策、下载安装编排等用例逻辑。
package updater

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/wire"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
)

// CheckInterval 默认更新检测间隔（24 小时）。
const CheckInterval = 24 * time.Hour

// Service 更新检测与安装编排服务。
type Service struct {
	updater   port.Updater
	installer port.Installer
	settings  *entity.UpdateSettings
}

// NewService 构造函数，供 Wire 注入。
// defaultChannel 由构建版本自动推导，保证正式版与测试版默认通道一致。
func NewService(u port.Updater, i port.Installer, defaultChannel models.UpdateChannel) *Service {
	settings := entity.DefaultUpdateSettings()
	settings.Channel = defaultChannel
	return &Service{
		updater:   u,
		installer: i,
		settings:  settings,
	}
}

// CheckUpdate 检测是否存在可用更新。
// currentVersion 为当前应用版本号（如 "v0.5.0"）。
// 若用户设置中 SkipVersion 与远程版本一致，则视为已跳过，返回 nil。
func (s *Service) CheckUpdate(ctx context.Context, currentVersion string) (*entity.UpdateInfo, error) {
	info, err := s.updater.FetchLatest(ctx, s.settings.Channel)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}

	// 用户已跳过此版本
	if s.settings.SkipVersion == info.Version {
		return nil, nil
	}

	hasUpdate, err := entity.HasUpdate(currentVersion, info.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to compare versions: %w", err)
	}
	if !hasUpdate {
		return nil, nil
	}

	s.settings.LastChecked = time.Now()
	return info, nil
}

// GetUpdateInfoByVersion 根据请求版本获取更新信息。
// version 为空时回退到最新版本检测；否则直接查询指定 tag，跳过跳过版本与版本比较逻辑。
func (s *Service) GetUpdateInfoByVersion(ctx context.Context, currentVersion, version string) (*entity.UpdateInfo, error) {
	if version == "" {
		return s.CheckUpdate(ctx, currentVersion)
	}

	info, err := s.updater.FetchByTag(ctx, version)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release %s: %w", version, err)
	}
	return info, nil
}

// DownloadUpdate 下载指定版本的更新包到本地。
// 下载路径按平台区分：非 Windows 为 ~/.medmemo/updates；Windows 为当前 exe 所在目录下的
// data\updates，与安装目录保持一致。
// 文件名格式：MedMemo-<version>-<os>-<arch>.<ext>
// 下载过程中通过 progress 回调推送字节进度。
func (s *Service) DownloadUpdate(ctx context.Context, info *entity.UpdateInfo, progress func(downloaded, total int64)) (string, error) {
	updateDir, err := updateDownloadDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve update directory: %w", err)
	}
	if err := os.MkdirAll(updateDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create update directory: %w", err)
	}

	ext := assetExt(runtime.GOOS)
	destName := fmt.Sprintf("MedMemo-%s-%s-%s%s", info.Version, runtime.GOOS, runtime.GOARCH, ext)
	destPath := filepath.Join(updateDir, destName)

	if err := s.updater.Download(ctx, info.DownloadURL, destPath, progress); err != nil {
		return "", fmt.Errorf("failed to download update: %w", err)
	}

	if err := s.updater.VerifyChecksum(destPath, info.Checksum); err != nil {
		// 校验失败则删除已下载文件
		_ = os.Remove(destPath)
		return "", fmt.Errorf("checksum verification failed: %w", err)
	}

	return destPath, nil
}

// updateDownloadDirFor 返回指定平台更新包下载目录，依赖注入便于测试。
func updateDownloadDirFor(goos string, exeFunc func() (string, error), homeFunc func() (string, error)) (string, error) {
	if goos == "windows" {
		exe, err := exeFunc()
		if err == nil {
			return filepath.Join(filepath.Dir(exe), "data", "updates"), nil
		}
	}
	home, err := homeFunc()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".medmemo", "updates"), nil
}

// updateDownloadDir 返回当前平台更新包下载目录。
// Windows 下优先使用当前 exe 所在目录的 data\updates，便于安装版统一管理数据；
// 若无法获取 exe 路径则回退到用户主目录。非 Windows 保持 ~/.medmemo/updates。
func updateDownloadDir() (string, error) {
	return updateDownloadDirFor(runtime.GOOS, os.Executable, os.UserHomeDir)
}

// ApplyUpdate 应用已下载的更新包。
// 安装前自动备份当前二进制，安装失败时触发回滚。
func (s *Service) ApplyUpdate(assetPath string) (string, error) {
	installedPath, err := s.installer.Install(assetPath)
	if err != nil {
		// 安装失败时尝试回滚
		_ = s.installer.Rollback()
		return "", fmt.Errorf("failed to install update: %w", err)
	}
	return installedPath, nil
}

// GetSettings 返回当前更新设置。
func (s *Service) GetSettings() *entity.UpdateSettings {
	return s.settings
}

// SetSettings 更新设置。
func (s *Service) SetSettings(st *entity.UpdateSettings) {
	s.settings = st
}

// SkipVersion 标记跳过指定版本。
func (s *Service) SkipVersion(v string) {
	s.settings.SkipVersion = v
}

// platformAssetNameFor 根据指定平台返回 GitHub Release 中的产物文件名模式。
func platformAssetNameFor(goos string) string {
	switch goos {
	case "linux":
		return fmt.Sprintf("*%s*.AppImage", runtime.GOARCH)
	case "darwin":
		return "*.dmg"
	case "windows":
		return "*-installer.exe"
	default:
		return ""
	}
}

// PlatformAssetName 根据当前平台返回 GitHub Release 中的产物文件名模式。
func PlatformAssetName() string {
	return platformAssetNameFor(runtime.GOOS)
}

// assetExt 返回当前平台对应的资产文件扩展名。
func assetExt(goos string) string {
	switch goos {
	case "linux":
		return ".AppImage"
	case "darwin":
		return ".dmg"
	case "windows":
		return ".exe"
	default:
		return ""
	}
}

// ProviderSet 供 Wire 使用的 Provider 集合。
var ProviderSet = wire.NewSet(
	NewService,
)
