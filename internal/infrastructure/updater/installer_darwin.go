//go:build darwin

package updater

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hzhan516/medmemo/internal/application/port"
)

// NewInstaller 根据当前运行平台创建对应的 Installer 实例。
func NewInstaller() port.Installer {
	return newDarwinInstaller()
}

// darwinInstaller 实现 macOS 平台的更新安装。
// 当前阶段使用浏览器引导下载，后续可通过 Sparkle 框架替换。
// TODO(Sparkle): 在 macOS 环境下集成 Sparkle 框架 CGO 绑定 [Issue#033]
type darwinInstaller struct {
	currentPath string
	backupPath  string
}

func newDarwinInstaller() *darwinInstaller {
	return &darwinInstaller{
		currentPath: getCurrentBinary(),
	}
}

// Install macOS 安装策略：
// 1. 下载 .dmg 到 ~/Downloads/
// 2. 通过 BrowserOpenURL 提示用户手动打开 .dmg 安装
// 3. 不自动替换 .app（需要用户拖拽到 Applications）
// 后续 Sparkle 集成后可实现后台静默下载 + 前台提醒安装。
func (d *darwinInstaller) Install(assetPath string) (string, error) {
	// 将 .dmg 移动到用户下载目录，方便查找
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	downloadsDir := filepath.Join(home, "Downloads")
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create downloads directory: %w", err)
	}

	destPath := filepath.Join(downloadsDir, filepath.Base(assetPath))
	if err := os.Rename(assetPath, destPath); err != nil {
		// 如果移动失败（跨设备），尝试复制
		if err := copyFileDarwin(assetPath, destPath); err != nil {
			return "", fmt.Errorf("failed to move dmg to Downloads: %w", err)
		}
		_ = os.Remove(assetPath)
	}

	return destPath, nil
}

// Rollback macOS 回滚：从备份恢复 .app。
func (d *darwinInstaller) Rollback() error {
	if d.backupPath == "" || d.currentPath == "" {
		return fmt.Errorf("no backup available for rollback")
	}
	if _, err := os.Stat(d.backupPath); err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}
	if err := os.Rename(d.backupPath, d.currentPath); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	return nil
}

// CurrentBinaryPath 返回当前二进制路径（通常是 .app/Contents/MacOS/MedMemo）。
func (d *darwinInstaller) CurrentBinaryPath() string {
	return d.currentPath
}

// copyFileDarwin 复制文件。
func copyFileDarwin(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file %s: %w", src, err)
	}
	return os.WriteFile(dst, input, 0644)
}
