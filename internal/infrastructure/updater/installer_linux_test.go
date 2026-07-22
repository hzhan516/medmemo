//go:build linux

package updater

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAppImagePathWith_ARGV0(t *testing.T) {
	tmpDir := t.TempDir()
	appImage := filepath.Join(tmpDir, "MedMemo.AppImage")
	require.NoError(t, os.WriteFile(appImage, []byte("x"), 0755))

	got := resolveAppImagePathWith(appImage, nil, func() string { return "/fallback" })
	assert.Equal(t, appImage, got)
}

func TestResolveAppImagePathWith_ProcCmdline(t *testing.T) {
	tmpDir := t.TempDir()
	appImage := filepath.Join(tmpDir, "MedMemo.AppImage")
	require.NoError(t, os.WriteFile(appImage, []byte("x"), 0755))

	cmdline := []byte(appImage + "\x00arg1\x00arg2\x00")
	got := resolveAppImagePathWith("/not-an-appimage", cmdline, func() string { return "/fallback" })
	assert.Equal(t, appImage, got)
}

func TestResolveAppImagePathWith_FallbackExecutable(t *testing.T) {
	fallback := "/some/exe"
	got := resolveAppImagePathWith("/not-an-appimage", []byte("/usr/bin/go\x00test\x00"), func() string { return fallback })
	assert.Equal(t, fallback, got)
}

func TestResolveAppImagePathWith_NilFallback(t *testing.T) {
	got := resolveAppImagePathWith("/not-an-appimage", []byte("/usr/bin/go\x00test\x00"), nil)
	assert.Empty(t, got)
}

func TestReadProcSelfCmdline(t *testing.T) {
	data := readProcSelfCmdline()
	// 在 Linux 上通常能读取到当前进程命令行
	assert.NotNil(t, data)
}

// TestNewInstaller 验证 Linux Installer 工厂函数。
func TestNewInstaller(t *testing.T) {
	inst := NewInstaller()
	require.NotNil(t, inst)
	assert.NotEmpty(t, inst.CurrentBinaryPath())
}

// TestManualPackageInstallRequired_Error 验证错误信息包含建议命令。
func TestManualPackageInstallRequired_Error(t *testing.T) {
	err := &ManualPackageInstallRequired{
		PackagePath: "/tmp/MedMemo.deb",
		Kind:        "deb",
		Command:     "sudo dpkg -i /tmp/MedMemo.deb",
	}
	assert.Contains(t, err.Error(), "manual install required")
	assert.Contains(t, err.Error(), "dpkg")
}

func TestLinuxInstaller_Install_AppImageReplacesOriginal(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", t.TempDir())

	currentBinary := filepath.Join(tmpDir, "MedMemo.AppImage")
	require.NoError(t, os.WriteFile(currentBinary, []byte("old binary"), 0755))

	newAppImage := filepath.Join(tmpDir, "MedMemo-v0.2.0.AppImage")
	require.NoError(t, os.WriteFile(newAppImage, []byte("new binary"), 0644))

	installer := &LinuxInstaller{currentPath: currentBinary}
	path, err := installer.Install(newAppImage)
	require.NoError(t, err)
	assert.Equal(t, currentBinary, path)

	content, err := os.ReadFile(currentBinary)
	require.NoError(t, err)
	assert.Equal(t, "new binary", string(content))

	info, err := os.Stat(currentBinary)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())

	assert.NotEmpty(t, installer.backupPath)
}

func TestLinuxInstaller_Install_NotWritableFallback(t *testing.T) {
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.Mkdir(readOnlyDir, 0755))

	currentBinary := filepath.Join(readOnlyDir, "MedMemo.AppImage")
	require.NoError(t, os.WriteFile(currentBinary, []byte("old"), 0755))
	require.NoError(t, os.Chmod(readOnlyDir, 0555))
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0755) })

	installer := &LinuxInstaller{currentPath: currentBinary}
	_, err := installer.Install(filepath.Join(tmpDir, "new.AppImage"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AppImage directory is not writable")
}

func TestLinuxInstaller_Install_NotAppImageReturnsManualError(t *testing.T) {
	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "MedMemo")
	require.NoError(t, os.WriteFile(currentBinary, []byte("old"), 0755))

	installer := &LinuxInstaller{currentPath: currentBinary}
	assetPath := filepath.Join(tmpDir, "new.AppImage")
	require.NoError(t, os.WriteFile(assetPath, []byte("update"), 0644))

	_, err := installer.Install(assetPath)
	require.Error(t, err)

	var manual *ManualPackageInstallRequired
	require.ErrorAs(t, err, &manual)
	assert.Equal(t, "unknown", manual.Kind)
	assert.Equal(t, assetPath, manual.PackagePath)
}

func TestLinuxInstaller_Rollback(t *testing.T) {
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

func TestLinuxInstaller_RollbackNoBackup(t *testing.T) {
	installer := &LinuxInstaller{}
	err := installer.Rollback()
	require.Error(t, err)
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
	currentBinary := filepath.Join(tmpDir, "MedMemo.AppImage")
	require.NoError(t, os.WriteFile(currentBinary, []byte("old"), 0755))

	installer := &LinuxInstaller{currentPath: currentBinary}
	_, err := installer.Install("/nonexistent/AppImage")
	assert.Error(t, err)
}

func TestLinuxInstallerInstall_CurrentBinaryMissing(t *testing.T) {
	tmpDir := t.TempDir()
	installer := &LinuxInstaller{currentPath: filepath.Join(tmpDir, "missing.AppImage")}
	assetPath := filepath.Join(tmpDir, "asset.AppImage")
	require.NoError(t, os.WriteFile(assetPath, []byte("update"), 0644))

	_, err := installer.Install(assetPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to backup current binary")
}

func TestLinuxInstallerInstall_RenameFails(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", t.TempDir())

	currentBinary := filepath.Join(tmpDir, "MedMemo.AppImage")
	require.NoError(t, os.WriteFile(currentBinary, []byte("old"), 0755))
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backup file not found")
}

func TestLinuxInstallerRollback_RenameFails(t *testing.T) {
	tmpDir := t.TempDir()
	backupPath := filepath.Join(tmpDir, "backup")
	currentPath := filepath.Join(tmpDir, "current_dir")
	require.NoError(t, os.WriteFile(backupPath, []byte("good"), 0644))
	require.NoError(t, os.Mkdir(currentPath, 0755))

	installer := &LinuxInstaller{
		currentPath: currentPath,
		backupPath:  backupPath,
	}
	err := installer.Rollback()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to restore backup")
}

func TestCopyFile_ReadFails(t *testing.T) {
	err := copyFile("/nonexistent/file", "/tmp/dest")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read source file")
}

func TestAssertDirWritable(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, assertDirWritable(tmpDir))

	readOnly := filepath.Join(tmpDir, "ro")
	require.NoError(t, os.Mkdir(readOnly, 0555))
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0755) })
	assert.Error(t, assertDirWritable(readOnly))
}

func TestLinuxInstaller_Install_DebReturnsManualInstall(t *testing.T) {
	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "MedMemo")
	require.NoError(t, os.WriteFile(currentBinary, []byte("old"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".install_kind"), []byte("deb"), 0644))

	assetPath := filepath.Join(tmpDir, "MedMemo_1.1.10_amd64.deb")
	require.NoError(t, os.WriteFile(assetPath, []byte("pkg"), 0644))

	installer := &LinuxInstaller{currentPath: currentBinary}
	_, err := installer.Install(assetPath)
	require.Error(t, err)

	var manual *ManualPackageInstallRequired
	require.ErrorAs(t, err, &manual)
	assert.Equal(t, "deb", manual.Kind)
	assert.Equal(t, assetPath, manual.PackagePath)
	assert.Contains(t, manual.Command, "dpkg")
}

func TestLinuxInstaller_Install_RPMReturnsManualInstall(t *testing.T) {
	tmpDir := t.TempDir()
	currentBinary := filepath.Join(tmpDir, "MedMemo")
	require.NoError(t, os.WriteFile(currentBinary, []byte("old"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".install_kind"), []byte("rpm"), 0644))

	assetPath := filepath.Join(tmpDir, "MedMemo-1.1.10-1.x86_64.rpm")
	require.NoError(t, os.WriteFile(assetPath, []byte("pkg"), 0644))

	installer := &LinuxInstaller{currentPath: currentBinary}
	_, err := installer.Install(assetPath)
	require.Error(t, err)

	var manual *ManualPackageInstallRequired
	require.ErrorAs(t, err, &manual)
	assert.Equal(t, "rpm", manual.Kind)
	assert.Equal(t, assetPath, manual.PackagePath)
	assert.Contains(t, manual.Command, "rpm")
}

func TestDetectInstallKind(t *testing.T) {
	t.Run("appimage by suffix", func(t *testing.T) {
		assert.Equal(t, "appimage", DetectInstallKind("/opt/MedMemo.AppImage"))
	})

	t.Run("deb by env", func(t *testing.T) {
		t.Setenv("MEDMEMO_INSTALL_KIND", "deb")
		assert.Equal(t, "deb", DetectInstallKind("/opt/MedMemo"))
	})

	t.Run("rpm by marker file", func(t *testing.T) {
		tmpDir := t.TempDir()
		binary := filepath.Join(tmpDir, "MedMemo")
		require.NoError(t, os.WriteFile(binary, []byte("x"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".install_kind"), []byte("rpm"), 0644))

		assert.Equal(t, "rpm", DetectInstallKind(binary))
	})

	t.Run("marker file read error falls back to unknown", func(t *testing.T) {
		tmpDir := t.TempDir()
		binary := filepath.Join(tmpDir, "MedMemo")
		require.NoError(t, os.WriteFile(binary, []byte("x"), 0755))
		// 将标记文件创建为目录，使 ReadFile 失败
		require.NoError(t, os.Mkdir(filepath.Join(tmpDir, ".install_kind"), 0755))

		assert.Equal(t, "unknown", DetectInstallKind(binary))
	})

	t.Run("appimage wins over env", func(t *testing.T) {
		t.Setenv("MEDMEMO_INSTALL_KIND", "deb")
		assert.Equal(t, "appimage", DetectInstallKind("/opt/MedMemo.AppImage"))
	})

	t.Run("unknown", func(t *testing.T) {
		assert.Equal(t, "unknown", DetectInstallKind("/opt/MedMemo"))
	})
}
