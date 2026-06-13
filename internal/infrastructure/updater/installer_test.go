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

	installer := &LinuxInstaller{currentPath: currentBinary}
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

	installer := &LinuxInstaller{
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
	installer := &LinuxInstaller{}
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

func TestGetCurrentBinary(t *testing.T) {
	path := getCurrentBinary()
	// 当前进程为 go test，应返回非空路径
	assert.NotEmpty(t, path)
}

func TestNewLinuxInstaller(t *testing.T) {
	inst := newLinuxInstaller()
	assert.NotNil(t, inst)
	assert.NotEmpty(t, inst.currentPath)
	assert.Equal(t, inst.currentPath, inst.CurrentBinaryPath())
}

func TestLinuxInstallerInstall_EmptyCurrentPath(t *testing.T) {
	installer := &LinuxInstaller{currentPath: ""}
	_, err := installer.Install("/tmp/fake.AppImage")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to determine current binary path")
}

func TestLinuxInstallerInstall_CopyFileFails(t *testing.T) {
	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "MedMemo")
	require.NoError(t, os.WriteFile(currentBinary, []byte("old"), 0755))

	installer := &LinuxInstaller{currentPath: currentBinary}
	// 传入不存在的 source 路径，使 copyFile 失败
	_, err := installer.Install("/nonexistent/AppImage")
	assert.Error(t, err)
}

func TestLinuxInstallerInstall_RenameFails(t *testing.T) {
	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "MedMemo")
	require.NoError(t, os.WriteFile(currentBinary, []byte("old"), 0755))
	// 创建一个目录，使 os.Rename 无法覆盖
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "new.AppImage"), 0755))

	installer := &LinuxInstaller{currentPath: currentBinary}
	_, err := installer.Install(filepath.Join(tmpDir, "new.AppImage"))
	assert.Error(t, err)
}

func TestLinuxInstallerRollback_BackupNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	installer := &LinuxInstaller{
		currentPath: filepath.Join(tmpDir, "MedMemo"),
		backupPath:  filepath.Join(tmpDir, "nonexistent.backup"),
	}
	err := installer.Rollback()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "backup file not found")
}

func TestLinuxInstallerRollback_RenameFails(t *testing.T) {
	tmpDir := t.TempDir()
	// backup 存在，currentPath 是目录导致 rename 失败
	backupPath := filepath.Join(tmpDir, "backup")
	currentPath := filepath.Join(tmpDir, "current_dir")
	require.NoError(t, os.WriteFile(backupPath, []byte("good"), 0644))
	require.NoError(t, os.Mkdir(currentPath, 0755))

	installer := &LinuxInstaller{
		currentPath: currentPath,
		backupPath:  backupPath,
	}
	err := installer.Rollback()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to restore backup")
}

func TestCopyFile_ReadFails(t *testing.T) {
	err := copyFile("/nonexistent/file", "/tmp/dest")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read source file")
}
