//go:build windows

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/registry"
)

func TestSelectWindowsDataDir_Portable(t *testing.T) {
	legacyDir := t.TempDir()
	got := selectWindowsDataDir("", legacyDir, func(string) bool { return false }, func(string) bool { return true })
	assert.Equal(t, legacyDir, got)
}

func TestSelectWindowsDataDir_FreshWritable(t *testing.T) {
	legacyDir := t.TempDir()
	installDir := t.TempDir()

	got := selectWindowsDataDir(installDir, legacyDir,
		func(string) bool { return false },
		func(string) bool { return true },
	)
	assert.Equal(t, filepath.Join(installDir, "data"), got)
}

func TestSelectWindowsDataDir_LegacyExists(t *testing.T) {
	legacyDir := t.TempDir()
	legacyDB := filepath.Join(legacyDir, "medmemo.db")
	require.NoError(t, os.WriteFile(legacyDB, []byte("legacy"), 0600))

	installDir := t.TempDir()
	got := selectWindowsDataDir(installDir, legacyDir,
		func(path string) bool { return path == legacyDB },
		func(string) bool { return true },
	)
	assert.Equal(t, legacyDir, got)
}

func TestSelectWindowsDataDir_DualLegacyPreferred(t *testing.T) {
	legacyDir := t.TempDir()
	legacyDB := filepath.Join(legacyDir, "medmemo.db")
	require.NoError(t, os.WriteFile(legacyDB, []byte("legacy"), 0600))

	installDir := t.TempDir()
	installData := filepath.Join(installDir, "data")
	require.NoError(t, os.MkdirAll(installData, 0755))

	got := selectWindowsDataDir(installDir, legacyDir,
		func(path string) bool { return path == legacyDB },
		func(string) bool { return true },
	)
	assert.Equal(t, legacyDir, got)
}

func TestSelectWindowsDataDir_InstallUnwritable(t *testing.T) {
	legacyDir := t.TempDir()
	installDir := t.TempDir()

	got := selectWindowsDataDir(installDir, legacyDir,
		func(string) bool { return false },
		func(string) bool { return false },
	)
	assert.Equal(t, legacyDir, got)
}

func TestDefaultDataDirPath_EnvOverride(t *testing.T) {
	home := t.TempDir()
	envDir := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("MEDMEMO_DATA_DIR", envDir)

	loader := NewLoader("", models.ChannelStable)
	cfg, err := loader.Load()
	require.NoError(t, err)
	assert.Equal(t, envDir, cfg.DataDir)
}

func TestDefaultDataDirPath_ConfigOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)

	configDir := t.TempDir()
	configPath := filepath.Join(home, ".medmemo", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0755))
	require.NoError(t, os.WriteFile(configPath, []byte("data_dir: "+configDir+"\n"), 0600))

	loader := NewLoader("", models.ChannelStable)
	cfg, err := loader.Load()
	require.NoError(t, err)
	assert.Equal(t, configDir, cfg.DataDir)
}

func TestDefaultDataDirPath_RegistryInstallDir(t *testing.T) {
	tmpDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)

	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\MedMemo`, registry.SET_VALUE)
	require.NoError(t, err)
	require.NoError(t, k.SetStringValue("InstallPath", tmpDir))
	require.NoError(t, k.Close())

	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\MedMemo`)
	})

	got := defaultDataDirPath()
	assert.Equal(t, filepath.Join(tmpDir, "data"), got)
}

func TestDefaultDataDirPath_RegistryLegacyExePath(t *testing.T) {
	tmpDir := t.TempDir()
	exePath := filepath.Join(tmpDir, "MedMemo.exe")
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)

	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\MedMemo`, registry.SET_VALUE)
	require.NoError(t, err)
	require.NoError(t, k.SetStringValue("InstallPath", exePath))
	require.NoError(t, k.Close())

	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\MedMemo`)
	})

	got := defaultDataDirPath()
	assert.Equal(t, filepath.Join(tmpDir, "data"), got)
}

func TestDefaultDataDirPath_RegistryMissing(t *testing.T) {
	_ = registry.DeleteKey(registry.CURRENT_USER, `Software\MedMemo`)

	home := t.TempDir()
	t.Setenv("USERPROFILE", home)

	got := defaultDataDirPath()
	assert.Equal(t, filepath.Join(home, ".medmemo", "data"), got)
}

func TestDefaultDataDirPath_RegistryLegacyPreferred(t *testing.T) {
	tmpDir := t.TempDir()
	home := t.TempDir()
	legacyDir := filepath.Join(home, ".medmemo", "data")
	require.NoError(t, os.MkdirAll(legacyDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(legacyDir, "medmemo.db"), []byte("legacy"), 0600))
	t.Setenv("USERPROFILE", home)

	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\MedMemo`, registry.SET_VALUE)
	require.NoError(t, err)
	require.NoError(t, k.SetStringValue("InstallPath", tmpDir))
	require.NoError(t, k.Close())

	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\MedMemo`)
	})

	got := defaultDataDirPath()
	assert.Equal(t, legacyDir, got)
}
