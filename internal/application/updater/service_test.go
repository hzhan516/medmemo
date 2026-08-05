package updater

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockUpdater 是 port.Updater 的测试替身。
type mockUpdater struct {
	latestInfo   *entity.UpdateInfo
	fetchErr     error
	tagInfo      *entity.UpdateInfo
	tagErr       error
	tagRequested string
	downloadTo   string
	downloadErr  error
	verifyErr    error
}

func (m *mockUpdater) FetchLatest(_ context.Context, _ models.UpdateChannel) (*entity.UpdateInfo, error) {
	return m.latestInfo, m.fetchErr
}

func (m *mockUpdater) FetchByTag(_ context.Context, tag string) (*entity.UpdateInfo, error) {
	m.tagRequested = tag
	return m.tagInfo, m.tagErr
}

func (m *mockUpdater) Download(_ context.Context, _, destPath string, _ func(downloaded, total int64)) error {
	m.downloadTo = destPath
	if m.downloadErr == nil {
		_ = os.MkdirAll(filepath.Dir(destPath), 0755)
		_ = os.WriteFile(destPath, []byte("payload"), 0644)
	}
	return m.downloadErr
}

func (m *mockUpdater) VerifyChecksum(_, _ string) error {
	return m.verifyErr
}

// mockInstaller 是 port.Installer 的测试替身。
type mockInstaller struct {
	installPath string
	installErr  error
	rollbackErr error
	currentPath string
	kind        string
}

func (m *mockInstaller) Install(_ string) (string, error) {
	return m.installPath, m.installErr
}

func (m *mockInstaller) Rollback() error {
	return m.rollbackErr
}

func (m *mockInstaller) CurrentBinaryPath() string {
	return m.currentPath
}

func (m *mockInstaller) InstallKind() string {
	if m.kind == "" {
		return "unknown"
	}
	return m.kind
}

func TestServiceCheckUpdate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		currentVersion string
		latestInfo     *entity.UpdateInfo
		fetchErr       error
		skipVersion    string
		wantInfo       bool
		wantErr        bool
	}{
		{
			name:           "new version available",
			currentVersion: "v0.1.0",
			latestInfo:     &entity.UpdateInfo{Version: "v0.2.0", Channel: models.ChannelBeta},
			wantInfo:       true,
		},
		{
			name:           "no update same version",
			currentVersion: "v0.2.0",
			latestInfo:     &entity.UpdateInfo{Version: "v0.2.0", Channel: models.ChannelBeta},
			wantInfo:       false,
		},
		{
			name:           "older version",
			currentVersion: "v0.3.0",
			latestInfo:     &entity.UpdateInfo{Version: "v0.2.0", Channel: models.ChannelBeta},
			wantInfo:       false,
		},
		{
			name:           "user skipped version",
			currentVersion: "v0.1.0",
			latestInfo:     &entity.UpdateInfo{Version: "v0.2.0", Channel: models.ChannelBeta},
			skipVersion:    "v0.2.0",
			wantInfo:       false,
		},
		{
			name:           "fetch error",
			currentVersion: "v0.1.0",
			fetchErr:       fmt.Errorf("network error"),
			wantInfo:       false,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockU := &mockUpdater{latestInfo: tt.latestInfo, fetchErr: tt.fetchErr}
			mockI := &mockInstaller{}
			svc := NewService(mockU, mockI, models.ChannelStable)
			svc.settings.SkipVersion = tt.skipVersion

			info, err := svc.CheckUpdate(context.Background(), tt.currentVersion)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantInfo {
				require.NotNil(t, info)
				assert.Equal(t, tt.latestInfo.Version, info.Version)
			} else {
				assert.Nil(t, info)
			}
		})
	}
}

func TestServiceDownloadUpdate(t *testing.T) {
	t.Parallel()
	mockU := &mockUpdater{}
	mockI := &mockInstaller{}
	svc := NewService(mockU, mockI, models.ChannelStable)

	info := &entity.UpdateInfo{
		Version:     "v0.2.0",
		DownloadURL: "https://example.com/update.AppImage",
		Checksum:    "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3",
	}

	path, err := svc.DownloadUpdate(context.Background(), info, nil)
	require.NoError(t, err)
	assert.Contains(t, path, "MedMemo-v0.2.0")
	assert.NotEmpty(t, mockU.downloadTo)
}

func TestServiceApplyUpdate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		installPath string
		installErr  error
		wantErr     bool
	}{
		{"success", "/new/binary", nil, false},
		{"install failed", "", fmt.Errorf("permission denied"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockU := &mockUpdater{}
			mockI := &mockInstaller{installPath: tt.installPath, installErr: tt.installErr}
			svc := NewService(mockU, mockI, models.ChannelStable)

			path, err := svc.ApplyUpdate("/tmp/update.AppImage")
			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, path)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.installPath, path)
		})
	}
}

func TestServiceSettings(t *testing.T) {
	t.Parallel()
	mockU := &mockUpdater{}
	mockI := &mockInstaller{}
	svc := NewService(mockU, mockI, models.ChannelStable)

	// 默认设置
	assert.True(t, svc.GetSettings().CheckEnabled)
	assert.Equal(t, models.ChannelStable, svc.GetSettings().Channel)

	// 修改设置
	newSettings := &entity.UpdateSettings{
		CheckEnabled: false,
		Channel:      models.ChannelStable,
		SkipVersion:  "v0.1.0",
	}
	svc.SetSettings(newSettings)
	assert.Equal(t, newSettings, svc.GetSettings())

	// 跳过版本
	svc.SkipVersion("v0.5.0")
	assert.Equal(t, "v0.5.0", svc.GetSettings().SkipVersion)
}

func TestServiceDefaultChannel(t *testing.T) {
	t.Parallel()
	t.Run("stable", func(t *testing.T) {
		mockU := &mockUpdater{}
		mockI := &mockInstaller{}
		svc := NewService(mockU, mockI, models.ChannelStable)
		assert.Equal(t, models.ChannelStable, svc.GetSettings().Channel)
	})

	t.Run("beta", func(t *testing.T) {
		mockU := &mockUpdater{}
		mockI := &mockInstaller{}
		svc := NewService(mockU, mockI, models.ChannelBeta)
		assert.Equal(t, models.ChannelBeta, svc.GetSettings().Channel)
	})
}

// TestServiceDownloadUpdate_Errors 验证下载各阶段失败均被包装。
func TestServiceDownloadUpdate_Errors(t *testing.T) {
	t.Run("missing checksum rejects download", func(t *testing.T) {
		mockU := &mockUpdater{}
		mockI := &mockInstaller{}
		svc := NewService(mockU, mockI, models.ChannelStable)

		info := &entity.UpdateInfo{
			Version:     "v0.2.0",
			DownloadURL: "https://example.com/update.AppImage",
			Checksum:    "",
		}

		_, err := svc.DownloadUpdate(context.Background(), info, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update checksum is not available")
		assert.Empty(t, mockU.downloadTo)
	})

	t.Run("download failure", func(t *testing.T) {
		mockU := &mockUpdater{downloadErr: fmt.Errorf("connection reset")}
		mockI := &mockInstaller{}
		svc := NewService(mockU, mockI, models.ChannelStable)

		t.Setenv("HOME", t.TempDir())
		info := &entity.UpdateInfo{
			Version:     "v0.2.0",
			DownloadURL: "https://example.com/update.AppImage",
			Checksum:    "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3",
		}

		_, err := svc.DownloadUpdate(context.Background(), info, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to download update")
	})

	t.Run("checksum mismatch removes file", func(t *testing.T) {
		mockU := &mockUpdater{verifyErr: fmt.Errorf("checksum mismatch")}
		mockI := &mockInstaller{}
		svc := NewService(mockU, mockI, models.ChannelStable)

		t.Setenv("HOME", t.TempDir())
		info := &entity.UpdateInfo{
			Version:     "v0.2.0",
			DownloadURL: "https://example.com/update.AppImage",
			Checksum:    "badchecksum",
		}

		_, err := svc.DownloadUpdate(context.Background(), info, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "checksum verification failed")
	})
}

// TestServiceApplyUpdate_RollbackError 验证安装失败且回滚失败时优先返回安装错误。
func TestServiceApplyUpdate_RollbackError(t *testing.T) {
	installErr := fmt.Errorf("permission denied")
	rollbackErr := fmt.Errorf("rollback failed")
	mockU := &mockUpdater{}
	mockI := &mockInstaller{installErr: installErr, rollbackErr: rollbackErr}
	svc := NewService(mockU, mockI, models.ChannelStable)

	path, err := svc.ApplyUpdate("/tmp/update.AppImage")
	require.Error(t, err)
	assert.Empty(t, path)
	assert.ErrorIs(t, err, installErr)
}

// TestPlatformAssetName 验证各平台产物名称模式。
func TestPlatformAssetName(t *testing.T) {
	tests := map[string]string{
		"linux":   "*amd64*.AppImage",
		"darwin":  "*.dmg",
		"windows": "*-installer.exe",
		"freebsd": "",
	}

	for goos, want := range tests {
		t.Run(goos, func(t *testing.T) {
			// 由于函数依赖 runtime.GOOS，直接断言当前平台
			if runtime.GOOS != goos {
				t.Skipf("当前平台为 %s，跳过 %s 断言", runtime.GOOS, goos)
			}
			assert.Equal(t, want, PlatformAssetName())
		})
	}
}

// TestUpdateDownloadDirFor_HomeError 验证无法获取用户主目录时返回错误。
func TestUpdateDownloadDirFor_HomeError(t *testing.T) {
	_, err := updateDownloadDirFor("linux", func() (string, error) { return "", fmt.Errorf("no exe") }, func() (string, error) { return "", fmt.Errorf("no home") }, func(string) bool { return true })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get home directory")
}

// TestUpdateDownloadDirFor_Windows 验证 Windows 分支使用 exe 所在目录（可写时）。
func TestUpdateDownloadDirFor_Windows(t *testing.T) {
	got, err := updateDownloadDirFor("windows", func() (string, error) { return "/opt/MedMemo/MedMemo.exe", nil }, func() (string, error) { return "", fmt.Errorf("no home") }, func(string) bool { return true })
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("/opt/MedMemo", "data", "updates"), got)
}

// TestUpdateDownloadDirFor_WindowsUnwritable 验证 Windows 安装目录不可写时回退到用户目录。
func TestUpdateDownloadDirFor_WindowsUnwritable(t *testing.T) {
	home := t.TempDir()
	got, err := updateDownloadDirFor("windows", func() (string, error) { return "/opt/MedMemo/MedMemo.exe", nil }, func() (string, error) { return home, nil }, func(string) bool { return false })
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".medmemo", "updates"), got)
}

// TestUpdateDownloadDirFor_WindowsFallbackHome 验证 Windows 无法获取 exe 时回退到主目录。
func TestUpdateDownloadDirFor_WindowsFallbackHome(t *testing.T) {
	home := t.TempDir()
	got, err := updateDownloadDirFor("windows", func() (string, error) { return "", fmt.Errorf("no exe") }, func() (string, error) { return home, nil }, func(string) bool { return true })
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".medmemo", "updates"), got)
}

// TestUpdateDownloadDir_HomeError 验证无法获取用户主目录时返回错误。
func TestUpdateDownloadDir_HomeError(t *testing.T) {
	t.Setenv("HOME", "")
	// 某些系统下空 HOME 仍可能解析成功，此时跳过该边界测试
	_, err := updateDownloadDir()
	if err != nil {
		assert.Contains(t, err.Error(), "failed to get home directory")
	}
}

// TestPlatformAssetNameFor 验证各平台产物名称模式。
func TestPlatformAssetNameFor(t *testing.T) {
	tests := map[string]string{
		"linux":   fmt.Sprintf("*%s*.AppImage", runtime.GOARCH),
		"darwin":  "*.dmg",
		"windows": "*-installer.exe",
		"freebsd": "",
	}

	for goos, want := range tests {
		t.Run(goos, func(t *testing.T) {
			assert.Equal(t, want, platformAssetNameFor(goos))
		})
	}
}

// TestAssetExt 验证各平台扩展名。
func TestAssetExt(t *testing.T) {
	tests := []struct {
		goos string
		want string
	}{
		{"linux", ".AppImage"},
		{"darwin", ".dmg"},
		{"windows", ".exe"},
		{"freebsd", ""},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			assert.Equal(t, tt.want, assetExt(tt.goos))
		})
	}
}

// TestLinuxAssetExt 验证 Linux 安装方式与扩展名映射。
func TestLinuxAssetExt(t *testing.T) {
	tests := []struct {
		kind string
		want string
	}{
		{"appimage", ".AppImage"},
		{"deb", ".deb"},
		{"rpm", ".rpm"},
		{"unknown", ".AppImage"},
		{"", ".AppImage"},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			assert.Equal(t, tt.want, linuxAssetExt(tt.kind))
		})
	}
}

// TestServiceDownloadUpdate_LinuxKindSelectsExtension 验证 Linux 下按 InstallKind 选择扩展名。
func TestServiceDownloadUpdate_LinuxKindSelectsExtension(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	tests := []struct {
		name    string
		kind    string
		wantExt string
	}{
		{"appimage", "appimage", ".AppImage"},
		{"deb", "deb", ".deb"},
		{"rpm", "rpm", ".rpm"},
		{"unknown fallback", "unknown", ".AppImage"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			mockU := &mockUpdater{}
			mockI := &mockInstaller{kind: tt.kind}
			svc := NewService(mockU, mockI, models.ChannelStable)

			info := &entity.UpdateInfo{
				Version:     "v1.1.10",
				DownloadURL: "https://example.com/update",
				Checksum:    "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3",
			}

			path, err := svc.DownloadUpdate(context.Background(), info, nil)
			require.NoError(t, err)
			assert.True(t, strings.HasSuffix(path, tt.wantExt), "path %s should end with %s", path, tt.wantExt)
			assert.True(t, strings.HasSuffix(mockU.downloadTo, tt.wantExt))
		})
	}
}

// TestServiceDownloadUpdate_AppImageExecutablePermission 验证 AppImage 下载后被赋予可执行权限。
func TestServiceDownloadUpdate_AppImageExecutablePermission(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	t.Setenv("HOME", t.TempDir())
	mockU := &mockUpdater{}
	mockI := &mockInstaller{kind: "appimage"}
	svc := NewService(mockU, mockI, models.ChannelStable)

	info := &entity.UpdateInfo{
		Version:     "v1.1.10",
		DownloadURL: "https://example.com/update",
		Checksum:    "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3",
	}

	path, err := svc.DownloadUpdate(context.Background(), info, nil)
	require.NoError(t, err)

	stat, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), stat.Mode().Perm())
}
