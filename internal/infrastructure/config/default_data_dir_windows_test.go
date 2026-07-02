//go:build windows

package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/registry"
)

func TestDefaultDataDirPath_RegistryInstallDir(t *testing.T) {
	tmpDir := t.TempDir()

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
