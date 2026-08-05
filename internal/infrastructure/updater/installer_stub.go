//go:build !linux && !darwin && !windows

package updater

import (
	"fmt"
	"os"
	"runtime"
)

// NoopInstaller 为不支持平台的空实现，避免编译错误。
type NoopInstaller struct{}

func (n *NoopInstaller) Install(assetPath string) (string, error) {
	return "", fmt.Errorf("auto-update not supported on %s", runtime.GOOS)
}

func (n *NoopInstaller) Rollback() error {
	return fmt.Errorf("rollback not supported on %s", runtime.GOOS)
}

func (n *NoopInstaller) CurrentBinaryPath() string {
	exe, _ := os.Executable()
	return exe
}

func (n *NoopInstaller) InstallKind() string {
	return "unknown"
}

// NewInstaller 为不支持的平台返回 NoopInstaller。
func NewInstaller() *NoopInstaller {
	return &NoopInstaller{}
}
