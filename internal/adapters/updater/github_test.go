package updater

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTransport 用于模拟 GitHub API 响应。
type mockTransport struct {
	statusCode int
	body       string
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	rec.WriteHeader(m.statusCode)
	rec.WriteString(m.body)
	return rec.Result(), nil
}

// platformAssetName 返回当前测试运行平台对应的 mock 资产文件名。
func platformAssetName() string {
	switch runtime.GOOS {
	case "linux":
		return "MedMemo-x86_64.AppImage"
	case "darwin":
		return "MedMemo.dmg"
	case "windows":
		return "MedMemoSetup.exe"
	default:
		return "MedMemo"
	}
}

func TestFetchLatest_StableSkipsPrerelease(t *testing.T) {
	assetName := platformAssetName()
	body := fmt.Sprintf(`[
		{
			"tag_name": "0.1.0-Pre-release-build.20",
			"name": "Pre-release",
			"body": "beta",
			"prerelease": true,
			"published_at": "2026-05-20T00:00:00Z",
			"assets": [
				{"name": "%s", "browser_download_url": "https://example.com/pre-release", "size": 100}
			]
		},
		{
			"tag_name": "0.1.0-build.10",
			"name": "Stable",
			"body": "stable",
			"prerelease": false,
			"published_at": "2026-05-19T00:00:00Z",
			"assets": [
				{"name": "%s", "browser_download_url": "https://example.com/stable", "size": 100}
			]
		}
	]`, assetName, assetName)

	client := &http.Client{Transport: &mockTransport{statusCode: http.StatusOK, body: body}}
	g := NewGitHubUpdater(client)

	info, err := g.FetchLatest(context.Background(), entity.ChannelStable)
	require.NoError(t, err)
	assert.Equal(t, "0.1.0-build.10", info.Version)
	assert.Equal(t, "Stable", info.Name)
	assert.Equal(t, "https://example.com/stable", info.DownloadURL)
	assert.False(t, info.Mandatory)
}

func TestFetchLatest_BetaIncludesPrerelease(t *testing.T) {
	assetName := platformAssetName()
	body := fmt.Sprintf(`[
		{
			"tag_name": "0.1.0-Pre-release-build.20",
			"name": "Pre-release",
			"body": "beta",
			"prerelease": true,
			"published_at": "2026-05-20T00:00:00Z",
			"assets": [
				{"name": "%s", "browser_download_url": "https://example.com/pre-release", "size": 100}
			]
		},
		{
			"tag_name": "0.1.0-build.10",
			"name": "Stable",
			"body": "stable",
			"prerelease": false,
			"published_at": "2026-05-19T00:00:00Z",
			"assets": [
				{"name": "%s", "browser_download_url": "https://example.com/stable", "size": 100}
			]
		}
	]`, assetName, assetName)

	client := &http.Client{Transport: &mockTransport{statusCode: http.StatusOK, body: body}}
	g := NewGitHubUpdater(client)

	info, err := g.FetchLatest(context.Background(), entity.ChannelBeta)
	require.NoError(t, err)
	assert.Equal(t, "0.1.0-Pre-release-build.20", info.Version)
	assert.Equal(t, "https://example.com/pre-release", info.DownloadURL)
}

func TestFetchLatest_StableNoReleaseAvailable(t *testing.T) {
	assetName := platformAssetName()
	body := fmt.Sprintf(`[
		{
			"tag_name": "0.1.0-Pre-release-build.20",
			"name": "Pre-release",
			"body": "beta",
			"prerelease": true,
			"published_at": "2026-05-20T00:00:00Z",
			"assets": [
				{"name": "%s", "browser_download_url": "https://example.com/pre-release", "size": 100}
			]
		}
	]`, assetName)

	client := &http.Client{Transport: &mockTransport{statusCode: http.StatusOK, body: body}}
	g := NewGitHubUpdater(client)

	_, err := g.FetchLatest(context.Background(), entity.ChannelStable)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no suitable release found")
}

func TestFetchLatest_NoMatchingAsset(t *testing.T) {
	body := `[
		{
			"tag_name": "0.1.0-build.10",
			"name": "Stable",
			"body": "stable",
			"prerelease": false,
			"published_at": "2026-05-19T00:00:00Z",
			"assets": [
				{"name": "wrong-platform.zip", "browser_download_url": "https://example.com/wrong", "size": 100}
			]
		}
	]`

	client := &http.Client{Transport: &mockTransport{statusCode: http.StatusOK, body: body}}
	g := NewGitHubUpdater(client)

	_, err := g.FetchLatest(context.Background(), entity.ChannelStable)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no suitable release found")
}

func TestFetchLatest_APIError(t *testing.T) {
	client := &http.Client{Transport: &mockTransport{statusCode: http.StatusInternalServerError, body: `{"message":"Internal Server Error"}`}}
	g := NewGitHubUpdater(client)

	_, err := g.FetchLatest(context.Background(), entity.ChannelStable)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "github api returned 500")
}

func TestFetchLatest_EmptyList(t *testing.T) {
	client := &http.Client{Transport: &mockTransport{statusCode: http.StatusOK, body: `[]`}}
	g := NewGitHubUpdater(client)

	_, err := g.FetchLatest(context.Background(), entity.ChannelStable)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no suitable release found")
}

func TestFindTargetAsset_LinuxArchSpecific(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo-x86_64.AppImage", BrowserDownloadURL: "https://example.com/arch", Size: 100},
	}
	got := findTargetAsset(assets, "linux", "amd64")
	require.NotNil(t, got)
	assert.Equal(t, "MedMemo-x86_64.AppImage", got.Name)
}

func TestFindTargetAsset_LinuxFallbackGeneric(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo.AppImage", BrowserDownloadURL: "https://example.com/generic", Size: 100},
	}
	got := findTargetAsset(assets, "linux", "amd64")
	require.NotNil(t, got)
	assert.Equal(t, "MedMemo.AppImage", got.Name)
}

func TestFindTargetAsset_LinuxPrefersArchOverGeneric(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo.AppImage", BrowserDownloadURL: "https://example.com/generic", Size: 100},
		{Name: "MedMemo-x86_64.AppImage", BrowserDownloadURL: "https://example.com/arch", Size: 100},
	}
	got := findTargetAsset(assets, "linux", "amd64")
	require.NotNil(t, got)
	assert.Equal(t, "MedMemo-x86_64.AppImage", got.Name)
}

func TestFindTargetAsset_LinuxNoMatch(t *testing.T) {
	assets := []githubAsset{
		{Name: "wrong-platform.zip", BrowserDownloadURL: "https://example.com/wrong", Size: 100},
	}
	got := findTargetAsset(assets, "linux", "amd64")
	assert.Nil(t, got)
}
