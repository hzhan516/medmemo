package updater

import (
	"github.com/google/wire"
	"github.com/hzhan516/medmemo/internal/application/port"
	infraUpdater "github.com/hzhan516/medmemo/internal/infrastructure/updater"
)

// InstallerAdapter 将基础设施层的具体 Installer 实现适配为 application/port.Installer 接口。
// 遵循 Clean Architecture：infrastructure 层不直接引用 application 层接口，
// 由 adapter 层负责类型转换与接口绑定。
type InstallerAdapter struct {
	installer port.Installer
}

// NewInstallerAdapter 创建 InstallerAdapter 实例。
func NewInstallerAdapter() *InstallerAdapter {
	return &InstallerAdapter{installer: infraUpdater.NewInstaller()}
}

// Install 委托给底层 installer。
func (a *InstallerAdapter) Install(assetPath string) (string, error) {
	return a.installer.Install(assetPath)
}

// Rollback 委托给底层 installer。
func (a *InstallerAdapter) Rollback() error {
	return a.installer.Rollback()
}

// CurrentBinaryPath 委托给底层 installer。
func (a *InstallerAdapter) CurrentBinaryPath() string {
	return a.installer.CurrentBinaryPath()
}

// InstallKind 委托给底层 installer。
func (a *InstallerAdapter) InstallKind() string {
	return a.installer.InstallKind()
}

// InstallerAdapterSet 供 Wire 使用的 ProviderSet。
//
//goland:noinspection GoUnusedGlobalVariable
var InstallerAdapterSet = wire.NewSet(
	NewInstallerAdapter,
	wire.Bind(new(port.Installer), new(*InstallerAdapter)),
)
