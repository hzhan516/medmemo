//go:build !linux && !darwin && !windows

package updater

import (
	"fmt"
	"os"
	"runtime"

	"github.com/medmemo/medmemo/internal/application/port"
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

// NewInstaller 为不支持的平台返回 noopInstaller。
func NewInstaller() port.Installer {
	return &noopInstaller{}
}
