//go:build !linux && !darwin && !windows

package updater

import "github.com/medmemo/medmemo/internal/application/port"

// NewInstaller 为不支持的平台返回 noopInstaller。
func NewInstaller() port.Installer {
	return &noopInstaller{}
}
