// Package updater 封装跨平台更新安装器实现。
// 使用 go:build 标签隔离各平台逻辑，编译期仅包含当前平台代码。
package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/google/wire"
)

// InstallerSet 供 Wire 使用的 Provider 集合。
var InstallerSet = wire.NewSet(
	NewInstaller,
)

// noopInstaller 为不支持平台的空实现，避免编译错误。
type noopInstaller struct{}

func (n *noopInstaller) Install(assetPath string) (string, error) {
	return "", fmt.Errorf("auto-update not supported on %s", runtime.GOOS)
}

func (n *noopInstaller) Rollback() error {
	return fmt.Errorf("rollback not supported on %s", runtime.GOOS)
}

func (n *noopInstaller) CurrentBinaryPath() string {
	exe, _ := os.Executable()
	return exe
}

// getCurrentBinary 获取当前运行二进制文件的绝对路径。
func getCurrentBinary() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		return exe
	}
	return abs
}
