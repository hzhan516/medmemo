package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/application/updater"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockUpdater 是 port.Updater 的测试替身。
type mockUpdater struct {
	info *entity.UpdateInfo
	err  error

	fetchByTagInfo    *entity.UpdateInfo
	fetchByTagErr     error
	fetchByTagTag     string
	fetchByTagCalled  bool
	fetchLatestCalled bool

	downloadTo string
}

func (m *mockUpdater) FetchLatest(_ context.Context, _ models.UpdateChannel) (*entity.UpdateInfo, error) {
	m.fetchLatestCalled = true
	return m.info, m.err
}

func (m *mockUpdater) FetchByTag(_ context.Context, tag string) (*entity.UpdateInfo, error) {
	m.fetchByTagCalled = true
	m.fetchByTagTag = tag
	return m.fetchByTagInfo, m.fetchByTagErr
}

func (m *mockUpdater) Download(_ context.Context, _, destPath string, _ func(int64, int64)) error {
	m.downloadTo = destPath
	return nil
}

func (m *mockUpdater) VerifyChecksum(_, _ string) error {
	return nil
}

// mockInstaller 是 port.Installer 的测试替身。
type mockInstaller struct {
	installPath string
	installErr  error
}

func (m *mockInstaller) Install(_ string) (string, error) { return m.installPath, m.installErr }
func (m *mockInstaller) Rollback() error                  { return nil }
func (m *mockInstaller) CurrentBinaryPath() string        { return "" }

// TestCheckUpdate_NoUpdate 验证远程版本与当前一致时返回 nil。
func TestCheckUpdate_NoUpdate(t *testing.T) {
	old := version
	version = "v1.1.8"
	defer func() { version = old }()

	info := &entity.UpdateInfo{Version: "v1.1.8", Channel: models.ChannelStable}
	svc := updater.NewService(&mockUpdater{info: info}, &mockInstaller{}, models.ChannelStable)
	app := &WailsApp{ctx: t.Context(), updaterSvc: svc}

	resp, err := app.CheckUpdate()
	require.NoError(t, err)
	assert.Nil(t, resp)
}

// TestCheckUpdate_ServiceError 验证更新检测错误被正确包装。
func TestCheckUpdate_ServiceError(t *testing.T) {
	old := version
	version = "v1.1.0"
	defer func() { version = old }()

	svc := updater.NewService(&mockUpdater{err: fmt.Errorf("network unreachable")}, &mockInstaller{}, models.ChannelStable)
	app := &WailsApp{ctx: t.Context(), updaterSvc: svc}

	_, err := app.CheckUpdate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check update")
}

// TestCheckUpdate_MapsFields 验证 CheckUpdate 将 entity.UpdateInfo 的扩展字段完整映射到前端响应。
func TestCheckUpdate_MapsFields(t *testing.T) {
	old := version
	version = "v1.1.0"
	defer func() { version = old }()

	published := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	info := &entity.UpdateInfo{
		Version:         "v1.1.8-Pre-release-build.57",
		DisplayVersion:  "v1.1.8-Pre-release-build.57",
		Name:            "v1.1.8 Pre-release",
		Body:            "test release notes",
		PublishedAt:     published,
		Mandatory:       true,
		Channel:         models.ChannelBeta,
		Prerelease:      true,
		BuildNumber:     "57",
		PreReleaseLabel: "Pre-release",
	}

	svc := updater.NewService(&mockUpdater{info: info}, &mockInstaller{}, models.ChannelBeta)
	app := &WailsApp{ctx: t.Context(), updaterSvc: svc}

	resp, err := app.CheckUpdate()
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, info.Version, resp.Version)
	assert.Equal(t, info.DisplayVersion, resp.DisplayVersion)
	assert.Equal(t, info.Name, resp.Name)
	assert.Equal(t, info.Body, resp.Body)
	assert.Equal(t, info.PublishedAt.Format(time.RFC3339), resp.PublishedAt)
	assert.Equal(t, info.Mandatory, resp.Mandatory)
	assert.Equal(t, string(info.Channel), resp.Channel)
	assert.Equal(t, info.Prerelease, resp.Prerelease)
	assert.Equal(t, info.PreReleaseLabel, resp.PrereleaseLabel)
	assert.Equal(t, info.BuildNumber, resp.BuildNumber)
}

// TestSetUpdateSettings_NilService 验证保存设置时 updaterSvc 未初始化返回错误。
func TestSetUpdateSettings_NilService(t *testing.T) {
	app := &WailsApp{ctx: t.Context()}
	err := app.SetUpdateSettings(UpdateSettingsResponse{Channel: "stable"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "updater service not initialized")
}

// TestSetUpdateSettings_MapsFields 验证设置可正确同步到 updater 服务。
func TestSetUpdateSettings_MapsFields(t *testing.T) {
	svc := updater.NewService(&mockUpdater{}, &mockInstaller{}, models.ChannelStable)
	app := &WailsApp{ctx: t.Context(), updaterSvc: svc}

	req := UpdateSettingsResponse{
		CheckEnabled: false,
		Channel:      "beta",
		SkipVersion:  "v1.1.0",
	}
	require.NoError(t, app.SetUpdateSettings(req))

	got, err := app.GetUpdateSettings()
	require.NoError(t, err)
	assert.Equal(t, req.CheckEnabled, got.CheckEnabled)
	assert.Equal(t, req.Channel, got.Channel)
	assert.Equal(t, req.SkipVersion, got.SkipVersion)
}

// TestSkipUpdateVersion_MapsToService 验证跳过版本可同步到 updater 服务。
func TestSkipUpdateVersion_MapsToService(t *testing.T) {
	svc := updater.NewService(&mockUpdater{}, &mockInstaller{}, models.ChannelStable)
	app := &WailsApp{ctx: t.Context(), updaterSvc: svc}

	require.NoError(t, app.SkipUpdateVersion("v1.1.0"))
	settings, err := app.GetUpdateSettings()
	require.NoError(t, err)
	assert.Equal(t, "v1.1.0", settings.SkipVersion)
}

// TestApplyUpdate_InstallFails 验证安装失败时不触发重启。
func TestApplyUpdate_InstallFails(t *testing.T) {
	svc := updater.NewService(&mockUpdater{}, &mockInstaller{installErr: fmt.Errorf("permission denied")}, models.ChannelStable)
	app := &WailsApp{ctx: t.Context(), updaterSvc: svc}

	err := app.ApplyUpdate("/tmp/update.AppImage")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to apply update")
}

// TestApplyUpdate_RestartFails 验证安装成功但重启失败时返回明确错误。
func TestApplyUpdate_RestartFails(t *testing.T) {
	svc := updater.NewService(&mockUpdater{}, &mockInstaller{installPath: "/nonexistent/AppImage"}, models.ChannelStable)
	app := &WailsApp{ctx: t.Context(), updaterSvc: svc}

	err := app.ApplyUpdate("/tmp/update.AppImage")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update installed but failed to restart")
}

// TestDownloadUpdate_ByVersion 验证指定版本时使用 FetchByTag 并正确进入下载流程。
func TestDownloadUpdate_ByVersion(t *testing.T) {
	old := version
	version = "v1.1.10"
	defer func() { version = old }()

	t.Setenv("HOME", t.TempDir())

	info := &entity.UpdateInfo{
		Version:     "v1.1.9",
		DownloadURL: "https://example.com/v1.1.9.AppImage",
		Checksum:    "",
	}
	mockU := &mockUpdater{fetchByTagInfo: info}
	svc := updater.NewService(mockU, &mockInstaller{}, models.ChannelStable)
	app := &WailsApp{ctx: t.Context(), updaterSvc: svc}

	path, err := app.DownloadUpdate(DownloadUpdateRequest{Version: "v1.1.9"})
	require.NoError(t, err)
	assert.Contains(t, path, "MedMemo-v1.1.9")
	assert.True(t, mockU.fetchByTagCalled)
	assert.Equal(t, "v1.1.9", mockU.fetchByTagTag)
	assert.False(t, mockU.fetchLatestCalled)
	assert.NotEmpty(t, mockU.downloadTo)
}

// TestDownloadUpdate_NoUpdateInfoNilError 验证返回 (nil, nil) 时提示无可用更新，且错误格式正确。
func TestDownloadUpdate_NoUpdateInfoNilError(t *testing.T) {
	old := version
	version = "v1.1.10"
	defer func() { version = old }()

	mockU := &mockUpdater{}
	svc := updater.NewService(mockU, &mockInstaller{}, models.ChannelStable)
	app := &WailsApp{ctx: t.Context(), updaterSvc: svc}

	_, err := app.DownloadUpdate(DownloadUpdateRequest{Version: "v1.1.9"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no update available for version v1.1.9")
	assert.NotContains(t, err.Error(), "%!w")
}
