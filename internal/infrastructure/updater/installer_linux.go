//go:build linux

package updater

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/medmemo/medmemo/internal/application/port"
)

// NewInstaller 根据当前运行平台创建对应的 Installer 实例。
func NewInstaller() port.Installer {
	return newLinuxInstaller()
}

// linuxInstaller 实现 Linux 平台（AppImage）的更新安装与回滚。
type linuxInstaller struct {
	currentPath string
	backupPath  string
}

func newLinuxInstaller() *linuxInstaller {
	return &linuxInstaller{
		currentPath: getCurrentBinary(),
	}
}

// Install 安装更新：备份当前 AppImage → 替换为新版本 → 设置可执行权限。
func (l *linuxInstaller) Install(assetPath string) (string, error) {
	if l.currentPath == "" {
		return "", fmt.Errorf("failed to determine current binary path")
	}

	// 创建备份目录
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	backupDir := filepath.Join(home, ".medmemo", "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// 备份当前二进制
	l.backupPath = filepath.Join(backupDir, filepath.Base(l.currentPath)+".backup")
	if err := copyFile(l.currentPath, l.backupPath); err != nil {
		return "", fmt.Errorf("failed to backup current binary: %w", err)
	}

	// 设置新文件可执行权限
	if err := os.Chmod(assetPath, 0755); err != nil {
		_ = l.Rollback()
		return "", fmt.Errorf("failed to set executable permission: %w", err)
	}

	// 替换当前二进制
	// 注意：Linux 允许替换正在运行的可执行文件（inode 级替换）
	if err := os.Rename(assetPath, l.currentPath); err != nil {
		_ = l.Rollback()
		return "", fmt.Errorf("failed to replace binary: %w", err)
	}

	return l.currentPath, nil
}

// Rollback 恢复到备份的二进制版本。
func (l *linuxInstaller) Rollback() error {
	if l.backupPath == "" || l.currentPath == "" {
		return fmt.Errorf("no backup available for rollback")
	}
	if _, err := os.Stat(l.backupPath); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}
	if err := os.Rename(l.backupPath, l.currentPath); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	return nil
}

// CurrentBinaryPath 返回当前二进制路径。
func (l *linuxInstaller) CurrentBinaryPath() string {
	return l.currentPath
}

// copyFile 复制文件，用于备份操作。
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0755)
}
