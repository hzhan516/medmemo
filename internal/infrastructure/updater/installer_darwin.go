//go:build darwin

package updater

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// NewInstaller 根据当前运行平台创建对应的 Installer 实例。
func NewInstaller() *DarwinInstaller {
	return newDarwinInstaller()
}

// DarwinInstaller 实现 macOS 平台的 DMG 自动安装与回滚。
// 授权失败时保留 DMG 并返回 ManualInstallRequired，引导用户手动安装。
type DarwinInstaller struct {
	currentPath  string
	backupPath   string
	targetBundle string
	runner       cmdRunner
}

// cmdRunner 抽象命令执行，便于单测注入。
type cmdRunner interface {
	Run(name string, args ...string) ([]byte, error)
}

// execCmdRunner 使用 os/exec 执行真实命令。
type execCmdRunner struct{}

func (execCmdRunner) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

func newDarwinInstaller() *DarwinInstaller {
	return &DarwinInstaller{
		currentPath: resolveAppBundlePath(),
		runner:      execCmdRunner{},
	}
}

// ManualInstallRequired 表示需要用户手动安装 DMG。
type ManualInstallRequired struct {
	DMGPath string
}

func (e *ManualInstallRequired) Error() string {
	return fmt.Sprintf("manual install required: open %s", e.DMGPath)
}

// Install 自动挂载 DMG 并替换 Applications 中的 .app。
func (d *DarwinInstaller) Install(assetPath string) (string, error) {
	target, err := d.resolveTargetBundle()
	if err != nil {
		return "", err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	backupDir := filepath.Join(home, ".medmemo", "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// 备份现有 .app
	if _, statErr := os.Stat(target); statErr == nil {
		d.backupPath = filepath.Join(backupDir, filepath.Base(target)+".backup")
		if err := os.Rename(target, d.backupPath); err != nil {
			return "", fmt.Errorf("failed to backup current app bundle: %w", err)
		}
	}

	mountPoint, err := d.attachDMG(assetPath)
	if err != nil {
		_ = d.Rollback()
		return "", fmt.Errorf("failed to attach dmg: %w", err)
	}
	defer d.detachDMG(mountPoint)

	srcApp, err := findAppInMountedDMG(mountPoint)
	if err != nil {
		_ = d.Rollback()
		return "", fmt.Errorf("failed to find app in dmg: %w", err)
	}

	// 优先直接复制；目标目录不可写则尝试管理员授权
	if err := d.copyAppBundle(srcApp, target); err != nil {
		if errors.Is(err, errAuthorizationCanceled) {
			fallbackDMG, copyErr := d.stageDMGForManualInstall(assetPath, home)
			if copyErr != nil {
				_ = d.Rollback()
				return "", fmt.Errorf("authorization canceled and failed to stage dmg: %w", copyErr)
			}
			_ = d.Rollback()
			return "", &ManualInstallRequired{DMGPath: fallbackDMG}
		}
		_ = d.Rollback()
		return "", fmt.Errorf("failed to replace app bundle: %w", err)
	}

	return target, nil
}

// Rollback 从备份恢复 .app。
func (d *DarwinInstaller) Rollback() error {
	if d.backupPath == "" || d.currentPath == "" {
		return fmt.Errorf("no backup available for rollback")
	}
	if _, err := os.Stat(d.backupPath); err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}
	target := resolveAppBundlePath()
	if target == "" {
		target = d.currentPath
	}
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("failed to remove broken bundle: %w", err)
	}
	if err := os.Rename(d.backupPath, target); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	return nil
}

// CurrentBinaryPath 返回当前二进制路径（通常是 .app/Contents/MacOS/MedMemo）。
func (d *DarwinInstaller) CurrentBinaryPath() string {
	return d.currentPath
}

// resolveAppBundlePath 从当前二进制向上查找 .app 目录。
func resolveAppBundlePath() string {
	return resolveAppBundlePathFrom(getCurrentBinary())
}

// resolveAppBundlePathFrom 从给定二进制路径向上查找 .app 目录。
func resolveAppBundlePathFrom(exe string) string {
	dir := filepath.Dir(exe)
	for {
		if strings.HasSuffix(filepath.Base(dir), ".app") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// resolveTargetBundle 根据当前 bundle 位置决定目标安装路径。
func (d *DarwinInstaller) resolveTargetBundle() (string, error) {
	if d.targetBundle != "" {
		return d.targetBundle, nil
	}

	const appName = "MedMemo.app"
	systemApp := filepath.Join("/Applications", appName)
	userApp := filepath.Join(os.Getenv("HOME"), "Applications", appName)

	current := resolveAppBundlePath()
	if current != "" {
		if current == systemApp {
			return systemApp, nil
		}
		if current == userApp {
			return userApp, nil
		}
	}

	// 默认尝试系统 Applications；无权限时 copyAppBundle 会降级到用户目录
	return systemApp, nil
}

// attachDMG 挂载 DMG 并返回挂载点。
func (d *DarwinInstaller) attachDMG(dmgPath string) (string, error) {
	out, err := d.runner.Run("hdiutil", "attach", "-nobrowse", "-readonly", dmgPath)
	if err != nil {
		return "", fmt.Errorf("hdiutil attach failed: %w\n%s", err, string(out))
	}
	return parseMountPoint(string(out))
}

// detachDMG 卸载 DMG，错误不中断主流程。
func (d *DarwinInstaller) detachDMG(mountPoint string) {
	_, _ = d.runner.Run("hdiutil", "detach", mountPoint)
}

// parseMountPoint 从 hdiutil attach 输出解析挂载点。
func parseMountPoint(output string) (string, error) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		// /dev/diskN Apple_HFS /Volumes/Name
		mount := strings.Join(fields[2:], " ")
		if strings.HasPrefix(mount, "/Volumes/") || strings.HasPrefix(mount, "/private/tmp/") {
			return mount, nil
		}
	}
	return "", fmt.Errorf("failed to parse mount point from hdiutil output")
}

// findAppInMountedDMG 在挂载点查找第一个 .app。
func findAppInMountedDMG(mountPoint string) (string, error) {
	entries, err := os.ReadDir(mountPoint)
	if err != nil {
		return "", fmt.Errorf("failed to read mounted dmg: %w", err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".app") {
			return filepath.Join(mountPoint, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("no .app found in dmg")
}

var errAuthorizationCanceled = errors.New("authorization canceled")

// copyAppBundle 复制 .app 到目标路径，必要时使用管理员授权。
func (d *DarwinInstaller) copyAppBundle(srcApp, target string) error {
	targetParent := filepath.Dir(target)

	// 先尝试直接复制
	if _, err := d.runner.Run("cp", "-R", srcApp, targetParent); err == nil {
		return nil
	}

	// 目标目录不可写，使用 osascript 提权复制
	script := fmt.Sprintf(`do shell script "cp -R %q %q" with administrator privileges`, srcApp, targetParent)
	out, err := d.runner.Run("osascript", "-e", script)
	if err != nil {
		if strings.Contains(string(out), "User canceled") || strings.Contains(string(out), "authorization") {
			return errAuthorizationCanceled
		}
		return fmt.Errorf("admin copy failed: %w\n%s", err, string(out))
	}
	return nil
}

// stageDMGForManualInstall 将 DMG 复制到 Downloads 供用户手动安装。
func (d *DarwinInstaller) stageDMGForManualInstall(dmgPath, home string) (string, error) {
	downloadsDir := filepath.Join(home, "Downloads")
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create Downloads: %w", err)
	}
	dest := filepath.Join(downloadsDir, filepath.Base(dmgPath))
	if err := copyFileDarwin(dmgPath, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// copyFileDarwin 复制文件。
func copyFileDarwin(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file %s: %w", src, err)
	}
	return os.WriteFile(dst, input, 0644)
}
