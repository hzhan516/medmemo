package updater

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DetectInstallKind 根据当前二进制路径、环境变量和标记文件检测 Linux 安装方式。
// 在非 Linux 平台始终返回 unknown。
// 检测优先级：
//  1. 当前可执行文件路径以 .AppImage 结尾 → appimage
//  2. 环境变量 MEDMEMO_INSTALL_KIND 为 deb/rpm → deb/rpm
//  3. 可执行文件同级目录下的 .install_kind 标记文件内容为 deb/rpm → deb/rpm
//  4. legacy package 探测：dpkg-query -S / rpm -qf → deb/rpm
//  5. 其他 → unknown
func DetectInstallKind(currentPath string) string {
	if runtime.GOOS != "linux" {
		return "unknown"
	}
	if isAppImage(currentPath) {
		return "appimage"
	}
	if kind := os.Getenv("MEDMEMO_INSTALL_KIND"); kind == "deb" || kind == "rpm" {
		return kind
	}
	markerPath := filepath.Join(filepath.Dir(currentPath), ".install_kind")
	if data, err := os.ReadFile(markerPath); err == nil {
		kind := strings.TrimSpace(string(data))
		if kind == "deb" || kind == "rpm" {
			return kind
		}
	}
	if kind := packageDetector(currentPath); kind == "deb" || kind == "rpm" {
		return kind
	}
	return "unknown"
}

// packageDetector 探测当前二进制是否由系统包管理器安装。
// 使用包级变量便于单测注入固定结果。
var packageDetector = func(currentPath string) string {
	return detectPackageKindWith(currentPath, exec.LookPath, exec.Command)
}

// detectPackageKindWith 依赖注入版本的包管理器探测。
func detectPackageKindWith(currentPath string, lookPath func(string) (string, error), command func(string, ...string) *exec.Cmd) string {
	if currentPath == "" {
		return "unknown"
	}

	// Debian/Ubuntu: dpkg-query -S <path> 成功则路径属于某个 deb 包
	if _, err := lookPath("dpkg-query"); err == nil {
		out, err := command("dpkg-query", "-S", currentPath).CombinedOutput()
		if err == nil && strings.Contains(string(out), ":") {
			return "deb"
		}
	}

	// Fedora/openSUSE: rpm -qf <path> 成功则路径属于某个 rpm 包
	if _, err := lookPath("rpm"); err == nil {
		out, err := command("rpm", "-qf", currentPath).CombinedOutput()
		if err == nil && !strings.Contains(strings.ToLower(string(out)), "not owned") {
			return "rpm"
		}
	}

	return "unknown"
}

// isAppImage 判断路径是否为 AppImage 格式。
func isAppImage(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".appimage")
}
