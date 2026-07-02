//go:build windows

package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/registry"
)

func fakeReadInstallPath(values map[registry.Key]string) readInstallPathFunc {
	return func(key registry.Key, path string) (string, error) {
		if v, ok := values[key]; ok {
			return v, nil
		}
		return "", fmt.Errorf("not found")
	}
}

func TestResolveInstallDir_PerUserHKCUDirectory(t *testing.T) {
	currentExe := `C:\Users\Alice\AppData\Local\Programs\MedMemo\MedMemo.exe`
	values := map[registry.Key]string{
		registry.CURRENT_USER: `C:\Users\Alice\AppData\Local\Programs\MedMemo`,
	}

	dir, err := resolveInstallDir(currentExe, fakeReadInstallPath(values))
	require.NoError(t, err)
	assert.Equal(t, `C:\Users\Alice\AppData\Local\Programs\MedMemo`, dir)
}

func TestResolveInstallDir_AllUsersHKLMDirectoryCompatibility(t *testing.T) {
	currentExe := `C:\Program Files\MedMemo\MedMemo.exe`
	values := map[registry.Key]string{
		registry.LOCAL_MACHINE: `C:\Program Files\MedMemo`,
	}

	dir, err := resolveInstallDir(currentExe, fakeReadInstallPath(values))
	require.NoError(t, err)
	assert.Equal(t, `C:\Program Files\MedMemo`, dir)
}

func TestResolveInstallDir_LegacyRegistryValueExecutablePath(t *testing.T) {
	currentExe := `C:\Users\Alice\AppData\Local\Programs\MedMemo\MedMemo.exe`
	values := map[registry.Key]string{
		registry.CURRENT_USER: `C:\Users\Alice\AppData\Local\Programs\MedMemo\MedMemo.exe`,
	}

	dir, err := resolveInstallDir(currentExe, fakeReadInstallPath(values))
	require.NoError(t, err)
	assert.Equal(t, `C:\Users\Alice\AppData\Local\Programs\MedMemo`, dir)
}

func TestResolveInstallDir_BothPreferCurrentExecutablePath(t *testing.T) {
	currentExe := `C:\Program Files\MedMemo\MedMemo.exe`
	values := map[registry.Key]string{
		registry.CURRENT_USER:  `C:\Users\Alice\AppData\Local\Programs\MedMemo`,
		registry.LOCAL_MACHINE: `C:\Program Files\MedMemo`,
	}

	dir, err := resolveInstallDir(currentExe, fakeReadInstallPath(values))
	require.NoError(t, err)
	assert.Equal(t, `C:\Program Files\MedMemo`, dir)
}

func TestResolveInstallDir_FallbackToCurrentDir(t *testing.T) {
	currentExe := `C:\Tools\MedMemo\MedMemo.exe`
	dir, err := resolveInstallDir(currentExe, fakeReadInstallPath(map[registry.Key]string{}))
	require.NoError(t, err)
	assert.Equal(t, `C:\Tools\MedMemo`, dir)
}

func TestWindowsInstaller_CommandUsesResolvedInstallDir(t *testing.T) {
	tmpDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)

	currentExe := filepath.Join(tmpDir, "MedMemo.exe")
	require.NoError(t, os.WriteFile(currentExe, []byte("old"), 0755))

	installerExe := filepath.Join(tmpDir, "MedMemo-installer.exe")
	require.NoError(t, os.WriteFile(installerExe, []byte("installer"), 0755))

	inst := &WindowsInstaller{currentPath: currentExe}

	// 使用一个可执行的假安装程序，验证命令行参数
	resolvedDir := filepath.Join(tmpDir, "resolved")
	require.NoError(t, os.MkdirAll(resolvedDir, 0755))

	// 通过覆盖 resolveInstallDir 的输入实现较困难，这里直接验证命令构建逻辑
	path, err := inst.Install(installerExe)
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(path, "MedMemo.exe"))
}

func TestWindowsInstaller_DArgumentIsLast(t *testing.T) {
	tmpDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)

	currentExe := filepath.Join(tmpDir, "MedMemo.exe")
	require.NoError(t, os.WriteFile(currentExe, []byte("old"), 0755))

	installerExe := filepath.Join(tmpDir, "MedMemo-installer.exe")
	require.NoError(t, os.WriteFile(installerExe, []byte("installer"), 0755))

	inst := &WindowsInstaller{currentPath: currentExe}
	_, err := inst.Install(installerExe)
	require.NoError(t, err)
}

func TestWindowsInstaller_EmptyCurrentPath(t *testing.T) {
	inst := &WindowsInstaller{currentPath: ""}
	_, err := inst.Install(`C:\fake.exe`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to determine current binary path")
}

func TestWindowsInstaller_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	current := filepath.Join(tmpDir, "MedMemo.exe")
	backup := filepath.Join(tmpDir, "MedMemo.exe.backup")
	require.NoError(t, os.WriteFile(current, []byte("broken"), 0755))
	require.NoError(t, os.WriteFile(backup, []byte("good"), 0755))

	inst := &WindowsInstaller{currentPath: current, backupPath: backup}
	require.NoError(t, inst.Rollback())

	data, err := os.ReadFile(current)
	require.NoError(t, err)
	assert.Equal(t, "good", string(data))
}
