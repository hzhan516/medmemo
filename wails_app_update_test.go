package main

import (
	"context"
	"testing"
	"time"

	"github.com/hzhan516/medmemo/internal/application/updater"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockUpdater 是 port.Updater 的测试替身，仅返回预设的 UpdateInfo。
type mockUpdater struct {
	info *entity.UpdateInfo
	err  error
}

func (m *mockUpdater) FetchLatest(ctx context.Context, channel models.UpdateChannel) (*entity.UpdateInfo, error) {
	return m.info, m.err
}

func (m *mockUpdater) Download(ctx context.Context, url, dest string, progress func(int64, int64)) error {
	return nil
}

func (m *mockUpdater) VerifyChecksum(path, checksum string) error {
	return nil
}

// mockInstaller 是 port.Installer 的测试替身，安装与回滚均直接成功。
type mockInstaller struct{}

func (m *mockInstaller) Install(assetPath string) (string, error) { return "", nil }
func (m *mockInstaller) Rollback() error                          { return nil }
func (m *mockInstaller) CurrentBinaryPath() string                { return "" }

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
