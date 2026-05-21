//go:build linux

package updater

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinuxInstallerInstall(t *testing.T) {
	// 创建临时目录模拟当前二进制
	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "MedMemo")
	require.NoError(t, os.WriteFile(currentBinary, []byte("old binary"), 0755))

	// 创建新的 AppImage
	newAppImage := filepath.Join(tmpDir, "MedMemo-v0.2.0.AppImage")
	require.NoError(t, os.WriteFile(newAppImage, []byte("new binary"), 0644))

	installer := &linuxInstaller{currentPath: currentBinary}
	path, err := installer.Install(newAppImage)
	require.NoError(t, err)
	assert.Equal(t, currentBinary, path)

	// 验证已替换
	content, err := os.ReadFile(currentBinary)
	require.NoError(t, err)
	assert.Equal(t, "new binary", string(content))

	// 由于测试使用真实 home 目录，此处简化验证备份路径已设置
	assert.NotEmpty(t, installer.backupPath)
}

func TestLinuxInstallerRollback(t *testing.T) {
	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "MedMemo")
	backupBinary := filepath.Join(tmpDir, "MedMemo.backup")

	require.NoError(t, os.WriteFile(currentBinary, []byte("broken binary"), 0755))
	require.NoError(t, os.WriteFile(backupBinary, []byte("good binary"), 0755))

	installer := &linuxInstaller{
		currentPath: currentBinary,
		backupPath:  backupBinary,
	}

	err := installer.Rollback()
	require.NoError(t, err)

	content, err := os.ReadFile(currentBinary)
	require.NoError(t, err)
	assert.Equal(t, "good binary", string(content))
}

func TestLinuxInstallerRollbackNoBackup(t *testing.T) {
	installer := &linuxInstaller{}
	err := installer.Rollback()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no backup available")
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source")
	dst := filepath.Join(tmpDir, "dest")

	require.NoError(t, os.WriteFile(src, []byte("test content"), 0644))
	require.NoError(t, copyFile(src, dst))

	content, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}
