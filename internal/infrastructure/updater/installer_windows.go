//go:build windows

package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// NewInstaller 根据当前运行平台创建对应的 Installer 实例。
func NewInstaller() *WindowsInstaller {
	return newWindowsInstaller()
}

// WindowsInstaller 实现 Windows 平台的更新安装。
type WindowsInstaller struct {
	currentPath string
	backupPath  string
}

func newWindowsInstaller() *WindowsInstaller {
	return &WindowsInstaller{
		currentPath: getCurrentBinary(),
	}
}

// Install 备份当前 exe 并启动安装程序，由 NSIS 在原安装目录完成静默升级。
func (w *WindowsInstaller) Install(assetPath string) (string, error) {
	if w.currentPath == "" {
		return "", fmt.Errorf("failed to determine current binary path")
	}

	installDir, err := resolveInstallDir(w.currentPath, defaultReadInstallPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve install directory: %w", err)
	}

	backupDir := filepath.Join(installDir, "data", "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	w.backupPath = filepath.Join(backupDir, filepath.Base(w.currentPath)+".backup")
	if err := copyFileWindows(w.currentPath, w.backupPath); err != nil {
		return "", fmt.Errorf("failed to backup current binary: %w", err)
	}

	tempDir := os.TempDir()
	installerPath := filepath.Join(tempDir, filepath.Base(assetPath))
	if err := os.Rename(assetPath, installerPath); err != nil {
		if err := copyFileWindows(assetPath, installerPath); err != nil {
			return "", fmt.Errorf("failed to stage installer: %w", err)
		}
		_ = os.Remove(assetPath)
	}

	// /D= 必须是最后一个参数
	cmd := exec.Command(installerPath, "/S", "/D="+installDir)
	if err := cmd.Start(); err != nil {
		_ = w.Rollback()
		return "", fmt.Errorf("failed to start installer: %w", err)
	}

	return filepath.Join(installDir, "MedMemo.exe"), nil
}

// Rollback 恢复到备份的二进制。
func (w *WindowsInstaller) Rollback() error {
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
func (w *WindowsInstaller) CurrentBinaryPath() string {
	return w.currentPath
}

// readInstallPathFunc 读取注册表中 InstallPath 值的函数签名。
type readInstallPathFunc func(key registry.Key, path string) (string, error)

// defaultReadInstallPath 从 Windows 注册表读取 InstallPath。
func defaultReadInstallPath(key registry.Key, path string) (string, error) {
	k, err := registry.OpenKey(key, path, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	value, _, err := k.GetStringValue("InstallPath")
	if err != nil {
		return "", err
	}
	return value, nil
}

// resolveInstallDir 根据当前 exe 路径与注册表记录确定升级目标目录。
// 注册表 InstallPath 既可能是目录也可能是旧版写的 exe 路径，统一归一化为目录。
func resolveInstallDir(currentExe string, readPath readInstallPathFunc) (string, error) {
	currentDir := filepath.Dir(currentExe)

	hkcuRaw, _ := readPath(registry.CURRENT_USER, `Software\MedMemo`)
	hklmRaw, _ := readPath(registry.LOCAL_MACHINE, `Software\MedMemo`)

	hkcu := normalizeInstallPath(hkcuRaw)
	hklm := normalizeInstallPath(hklmRaw)

	// 优先匹配当前运行 exe 所在目录，防止 per-user/all-users 混淆
	if hkcu != "" && hkcu == currentDir {
		return currentDir, nil
	}
	if hklm != "" && hklm == currentDir {
		return currentDir, nil
	}

	// fallback：按 HKCU → HKLM → 当前 exe 目录
	if hkcu != "" {
		return hkcu, nil
	}
	if hklm != "" {
		return hklm, nil
	}
	return currentDir, nil
}

// normalizeInstallPath 将注册表 InstallPath 值归一化为安装目录。
// 旧版本可能写入 exe 路径，新版本写入目录，两者都需要兼容。
func normalizeInstallPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.EqualFold(filepath.Base(value), "MedMemo.exe") {
		return filepath.Dir(value)
	}
	return value
}

// copyFileWindows 复制文件。
func copyFileWindows(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file %s: %w", src, err)
	}
	return os.WriteFile(dst, input, 0755)
}
