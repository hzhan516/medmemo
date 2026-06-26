//go:build linux

package updater

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NewInstaller 根据当前运行平台创建对应的 Installer 实例。
func NewInstaller() *LinuxInstaller {
	return newLinuxInstaller()
}

// LinuxInstaller 实现 Linux 平台（AppImage）的更新安装与回滚。
type LinuxInstaller struct {
	currentPath string
	backupPath  string
}

func newLinuxInstaller() *LinuxInstaller {
	return &LinuxInstaller{
		currentPath: resolveAppImagePath(),
	}
}

// Install 安装更新：备份当前 AppImage → 替换为新版本 → 设置可执行权限。
func (l *LinuxInstaller) Install(assetPath string) (string, error) {
	if l.currentPath == "" {
		return "", fmt.Errorf("failed to determine current binary path")
	}

	if !isAppImage(l.currentPath) {
		return "", fmt.Errorf("manual update required: current binary is not an AppImage")
	}

	dir := filepath.Dir(l.currentPath)
	if err := assertDirWritable(dir); err != nil {
		return "", fmt.Errorf("AppImage directory is not writable: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	backupDir := filepath.Join(home, ".medmemo", "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	l.backupPath = filepath.Join(backupDir, filepath.Base(l.currentPath)+".backup")
	if err := copyFile(l.currentPath, l.backupPath); err != nil {
		return "", fmt.Errorf("failed to backup current binary: %w", err)
	}

	if err := os.Chmod(assetPath, 0755); err != nil {
		_ = l.Rollback()
		return "", fmt.Errorf("failed to set executable permission: %w", err)
	}

	// Linux 允许替换正在运行的可执行文件（inode 级替换）
	if err := os.Rename(assetPath, l.currentPath); err != nil {
		_ = l.Rollback()
		return "", fmt.Errorf("failed to replace binary: %w", err)
	}

	return l.currentPath, nil
}

// Rollback 恢复到备份的二进制版本。
func (l *LinuxInstaller) Rollback() error {
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
func (l *LinuxInstaller) CurrentBinaryPath() string {
	return l.currentPath
}

// resolveAppImagePath 解析用户实际启动的 AppImage 路径。
// 优先级：ARGV0 → /proc/self/cmdline → os.Executable() 兜底。
func resolveAppImagePath() string {
	return resolveAppImagePathWith(
		os.Getenv("ARGV0"),
		readProcSelfCmdline(),
		getCurrentBinary,
	)
}

// resolveAppImagePathWith 依赖注入版本，便于单测。
func resolveAppImagePathWith(argv0 string, cmdline []byte, fallback func() string) string {
	if isAppImage(argv0) {
		if abs, err := filepath.Abs(argv0); err == nil {
			return abs
		}
		return argv0
	}

	args := strings.Split(string(bytes.TrimRight(cmdline, "\x00")), "\x00")
	for _, arg := range args {
		if isAppImage(arg) {
			if abs, err := filepath.Abs(arg); err == nil {
				return abs
			}
			return arg
		}
	}

	if fallback != nil {
		return fallback()
	}
	return ""
}

// readProcSelfCmdline 读取当前进程命令行参数。
func readProcSelfCmdline() []byte {
	data, err := os.ReadFile("/proc/self/cmdline")
	if err != nil {
		return nil
	}
	return data
}

// isAppImage 判断路径是否为 AppImage。
func isAppImage(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".appimage")
}

// assertDirWritable 通过创建临时文件验证目录可写。
func assertDirWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".medmemo-write-test-*")
	if err != nil {
		return err
	}
	name := f.Name()
	_ = f.Close()
	return os.Remove(name)
}

// copyFile 复制文件，用于备份操作。
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file %s: %w", src, err)
	}
	return os.WriteFile(dst, input, 0755)
}
