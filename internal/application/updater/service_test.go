package updater

import (
	"context"
	"fmt"
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockUpdater 是 port.Updater 的测试替身。
type mockUpdater struct {
	latestInfo *entity.UpdateInfo
	fetchErr   error
	downloadTo string
	verifyErr  error
}

func (m *mockUpdater) FetchLatest(ctx context.Context, channel models.UpdateChannel) (*entity.UpdateInfo, error) {
	return m.latestInfo, m.fetchErr
}

func (m *mockUpdater) Download(ctx context.Context, url, destPath string, progress func(downloaded, total int64)) error {
	m.downloadTo = destPath
	return nil
}

func (m *mockUpdater) VerifyChecksum(path, expectedSHA256 string) error {
	return m.verifyErr
}

// mockInstaller 是 port.Installer 的测试替身。
type mockInstaller struct {
	installPath string
	installErr  error
	rollbackErr error
	currentPath string
}

func (m *mockInstaller) Install(assetPath string) (string, error) {
	return m.installPath, m.installErr
}

func (m *mockInstaller) Rollback() error {
	return m.rollbackErr
}

func (m *mockInstaller) CurrentBinaryPath() string {
	return m.currentPath
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
		Checksum:    "",
	}

	path, err := svc.DownloadUpdate(context.Background(), info, nil)
	require.NoError(t, err)
	assert.Contains(t, path, "MedMemo-v0.2.0")
	assert.NotEmpty(t, mockU.downloadTo)
}

func TestServiceApplyUpdate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		installErr error
		wantErr    bool
	}{
		{"success", nil, false},
		{"install failed", fmt.Errorf("permission denied"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockU := &mockUpdater{}
			mockI := &mockInstaller{installErr: tt.installErr}
			svc := NewService(mockU, mockI, models.ChannelStable)

			err := svc.ApplyUpdate("/tmp/update.AppImage")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
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
