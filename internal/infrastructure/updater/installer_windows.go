//go:build windows

package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/medmemo/medmemo/internal/application/port"
)

// NewInstaller 根据当前运行平台创建对应的 Installer 实例。
func NewInstaller() port.Installer {
	return newWindowsInstaller()
}

// windowsInstaller 实现 Windows 平台的更新安装。
type windowsInstaller struct {
	currentPath string
	backupPath  string
}

func newWindowsInstaller() *windowsInstaller {
	return &windowsInstaller{
		currentPath: getCurrentBinary(),
	}
}

// Install Windows 安装策略：
// 1. 备份当前 .exe
// 2. 启动新安装程序（带静默安装参数 /S）
// 3. 提示用户当前应用将在安装完成后关闭
// 注意：Windows 不允许直接替换正在运行的 .exe，必须通过安装程序完成。
func (w *windowsInstaller) Install(assetPath string) (string, error) {
	if w.currentPath == "" {
		return "", fmt.Errorf("failed to determine current binary path")
	}

	// 创建备份
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	backupDir := filepath.Join(home, ".medmemo", "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	w.backupPath = filepath.Join(backupDir, filepath.Base(w.currentPath)+".backup")
	if err := copyFileWindows(w.currentPath, w.backupPath); err != nil {
		return "", fmt.Errorf("failed to backup current binary: %w", err)
	}

	// 将安装包移动到临时目录并启动
	tempDir := os.TempDir()
	installerPath := filepath.Join(tempDir, filepath.Base(assetPath))
	if err := os.Rename(assetPath, installerPath); err != nil {
		if err := copyFileWindows(assetPath, installerPath); err != nil {
			return "", fmt.Errorf("failed to stage installer: %w", err)
		}
		_ = os.Remove(assetPath)
	}

	// 启动安装程序（NSIS 静默安装参数 /S）
	cmd := exec.Command(installerPath, "/S")
	if err := cmd.Start(); err != nil {
		_ = w.Rollback()
		return "", fmt.Errorf("failed to start installer: %w", err)
	}

	return installerPath, nil
}

// Rollback 恢复到备份的二进制。
func (w *windowsInstaller) Rollback() error {
	if w.backupPath == "" || w.currentPath == "" {
		return fmt.Errorf("no backup available for rollback")
	}
	if _, err := os.Stat(w.backupPath); err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}
	if err := os.Rename(w.backupPath, w.currentPath); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	return nil
}

// CurrentBinaryPath 返回当前二进制路径。
func (w *windowsInstaller) CurrentBinaryPath() string {
	return w.currentPath
}

// copyFileWindows 复制文件。
func copyFileWindows(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0755)
}
