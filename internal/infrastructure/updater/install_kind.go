package updater

import (
	"os"
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
//  4. 其他 → unknown
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
	return "unknown"
}

// isAppImage 判断路径是否为 AppImage 格式。
func isAppImage(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".appimage")
}
