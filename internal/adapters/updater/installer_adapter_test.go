package updater

import (
	"testing"

	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 编译时检查 InstallerAdapter 实现了 port.Installer 接口。
var _ port.Installer = (*InstallerAdapter)(nil)

// TestNewInstallerAdapter 验证构造函数返回非空实例。
func TestNewInstallerAdapter(t *testing.T) {
	adapter := NewInstallerAdapter()
	require.NotNil(t, adapter)
	assert.NotNil(t, adapter.installer)
}

// TestInstallerAdapter_Install 验证 Install 委托到底层 installer。
func TestInstallerAdapter_Install(t *testing.T) {
	adapter := NewInstallerAdapter()
	// 底层 installer 在 Linux 上返回错误（非 AppImage 环境），仅验证委托路径可达。
	_, err := adapter.Install("/tmp/nonexistent.AppImage")
	assert.Error(t, err)
}

// TestInstallerAdapter_Rollback 验证 Rollback 委托到底层 installer。
func TestInstallerAdapter_Rollback(t *testing.T) {
	adapter := NewInstallerAdapter()
	// 无备份时 Rollback 应返回错误，仅验证委托路径可达。
	err := adapter.Rollback()
	assert.Error(t, err)
}

// TestInstallerAdapter_CurrentBinaryPath 验证 CurrentBinaryPath 委托到底层 installer。
func TestInstallerAdapter_CurrentBinaryPath(t *testing.T) {
	adapter := NewInstallerAdapter()
	path := adapter.CurrentBinaryPath()
	assert.NotEmpty(t, path)
}

// TestInstallerAdapter_InstallKind 验证 InstallKind 委托到底层 installer。
func TestInstallerAdapter_InstallKind(t *testing.T) {
	adapter := NewInstallerAdapter()
	kind := adapter.InstallKind()
	assert.NotEmpty(t, kind)
}
