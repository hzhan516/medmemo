//go:build darwin

package updater

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRunner 模拟 hdiutil / cp / osascript 命令。
type fakeRunner struct {
	t          *testing.T
	mountPoint string
	directCpOK bool
	adminCpOK  bool
	canceled   bool
	cpTarget   string
	cpSource   string
	calls      []string
}

func (f *fakeRunner) Run(name string, args ...string) ([]byte, error) {
	cmd := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, cmd)

	switch name {
	case "hdiutil":
		if len(args) > 0 && args[0] == "attach" {
			return []byte(fmt.Sprintf("/dev/disk2s1\tApple_HFS\t%s\n", f.mountPoint)), nil
		}
		return []byte("ok\n"), nil
	case "cp":
		if f.directCpOK {
			f.cpSource = args[len(args)-2]
			f.cpTarget = filepath.Join(args[len(args)-1], filepath.Base(args[len(args)-2]))
			require.NoError(f.t, copyApp(f.cpSource, f.cpTarget))
			return []byte(""), nil
		}
		return []byte("cp: permission denied"), fmt.Errorf("exit status 1")
	case "osascript":
		if f.adminCpOK {
			f.cpSource = extractSource(args)
			f.cpTarget = filepath.Join(extractTargetParent(args), filepath.Base(f.cpSource))
			require.NoError(f.t, copyApp(f.cpSource, f.cpTarget))
			return []byte(""), nil
		}
		if f.canceled {
			return []byte("User canceled"), fmt.Errorf("exit status 1")
		}
		return []byte("admin copy failed"), fmt.Errorf("exit status 1")
	}
	return []byte(""), nil
}

func extractSource(args []string) string {
	script := args[len(args)-1]
	// 简单解析 cp -R "src" "dst"
	parts := strings.Split(script, `"`)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func extractTargetParent(args []string) string {
	script := args[len(args)-1]
	parts := strings.Split(script, `"`)
	if len(parts) >= 4 {
		return parts[3]
	}
	return ""
}

// copyApp 模拟 cp -R 复制 .app 目录。
func copyApp(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

func TestResolveAppBundlePath(t *testing.T) {
	tmpDir := t.TempDir()
	bundle := filepath.Join(tmpDir, "MedMemo.app", "Contents", "MacOS")
	require.NoError(t, os.MkdirAll(bundle, 0755))
	exe := filepath.Join(bundle, "MedMemo")
	require.NoError(t, os.WriteFile(exe, []byte("binary"), 0755))

	assert.Equal(t, filepath.Join(tmpDir, "MedMemo.app"), resolveAppBundlePathFrom(exe))
}

func TestFindAppInMountedDMG(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "MedMemo.app", "Contents")
	require.NoError(t, os.MkdirAll(appDir, 0755))

	appPath, err := findAppInMountedDMG(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, "MedMemo.app"), appPath)
}

func TestDarwinInstaller_WritableTargetAutoReplace(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", t.TempDir())

	// 构造挂载点内的源 .app
	mountDir := filepath.Join(tmpDir, "mount")
	srcApp := filepath.Join(mountDir, "MedMemo.app")
	require.NoError(t, os.MkdirAll(filepath.Join(srcApp, "Contents"), 0755))

	// 构造目标 .app（旧版本）
	target := filepath.Join(tmpDir, "MedMemo.app")
	require.NoError(t, os.MkdirAll(filepath.Join(target, "Contents"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(target, "Contents", "old.txt"), []byte("old"), 0644))

	dmgPath := filepath.Join(tmpDir, "MedMemo.dmg")
	require.NoError(t, os.WriteFile(dmgPath, []byte("dmg"), 0644))

	inst := &DarwinInstaller{
		targetBundle: target,
		runner:       &fakeRunner{t: t, mountPoint: mountDir, directCpOK: true},
	}

	path, err := inst.Install(dmgPath)
	require.NoError(t, err)
	assert.Equal(t, target, path)

	data, err := os.ReadFile(filepath.Join(target, "Contents", "old.txt"))
	require.NoError(t, err)
	assert.Equal(t, "old", string(data))

	assert.NotEmpty(t, inst.backupPath)
}

func TestDarwinInstaller_AdminAuthorizationFailureFallback(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	mountDir := filepath.Join(tmpDir, "mount")
	srcApp := filepath.Join(mountDir, "MedMemo.app")
	require.NoError(t, os.MkdirAll(filepath.Join(srcApp, "Contents"), 0755))

	target := filepath.Join(tmpDir, "MedMemo.app")
	require.NoError(t, os.MkdirAll(filepath.Join(target, "Contents"), 0755))

	dmgPath := filepath.Join(tmpDir, "MedMemo.dmg")
	require.NoError(t, os.WriteFile(dmgPath, []byte("dmg"), 0644))

	inst := &DarwinInstaller{
		targetBundle: target,
		runner:       &fakeRunner{t: t, mountPoint: mountDir, canceled: true},
	}

	_, err := inst.Install(dmgPath)
	require.Error(t, err)

	var manual *ManualInstallRequired
	require.True(t, errors.As(err, &manual), "expected ManualInstallRequired error")
	assert.FileExists(t, manual.DMGPath)
}

func TestDarwinInstaller_RollbackRestoresAppBundle(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "MedMemo.app")
	backup := filepath.Join(tmpDir, "MedMemo.app.backup")

	require.NoError(t, os.MkdirAll(filepath.Join(target, "Contents"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(target, "Contents", "flag"), []byte("broken"), 0644))

	require.NoError(t, os.MkdirAll(filepath.Join(backup, "Contents"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(backup, "Contents", "flag"), []byte("good"), 0644))

	inst := &DarwinInstaller{
		currentPath:  target,
		backupPath:   backup,
		targetBundle: target,
	}

	require.NoError(t, inst.Rollback())

	data, err := os.ReadFile(filepath.Join(target, "Contents", "flag"))
	require.NoError(t, err)
	assert.Equal(t, "good", string(data))
}
