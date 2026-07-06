package updater

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain 固定 Linux 安装方式为 appimage，避免测试二进制所在目录的 .install_kind 标记文件
// 影响 FetchLatest 相关测试，同时保证资产匹配结果可预期。
func TestMain(m *testing.M) {
	getLinuxInstallKind = func() string { return "appimage" }
	os.Exit(m.Run())
}

// TestNewGitHubUpdater_NilClient 验证传入 nil 时构造默认 HTTP 客户端。
func TestNewGitHubUpdater_NilClient(t *testing.T) {
	g := NewGitHubUpdater(nil)
	require.NotNil(t, g)
	require.NotNil(t, g.client)
	assert.Equal(t, defaultHTTPTimeout, g.client.Timeout)
}

// TestNewDefaultHTTPClient 验证默认 HTTP 客户端配置。
func TestNewDefaultHTTPClient(t *testing.T) {
	client := NewDefaultHTTPClient()
	require.NotNil(t, client)
	assert.Equal(t, defaultHTTPTimeout, client.Timeout)
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	assert.Equal(t, uint16(tls.VersionTLS12), transport.TLSClientConfig.MinVersion)
}

// mockTransport 用于模拟 GitHub API 响应。
type mockTransport struct {
	statusCode int
	body       string
}

func (m *mockTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	rec.WriteHeader(m.statusCode)
	_, _ = rec.WriteString(m.body)
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

	info, err := g.FetchLatest(context.Background(), models.ChannelStable)
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

	info, err := g.FetchLatest(context.Background(), models.ChannelBeta)
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

	_, err := g.FetchLatest(context.Background(), models.ChannelStable)
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

	_, err := g.FetchLatest(context.Background(), models.ChannelStable)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no suitable release found")
}

func TestFetchLatest_APIError(t *testing.T) {
	client := &http.Client{Transport: &mockTransport{statusCode: http.StatusInternalServerError, body: `{"message":"Internal Server Error"}`}}
	g := NewGitHubUpdater(client)

	_, err := g.FetchLatest(context.Background(), models.ChannelStable)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "github api returned 500")
}

func TestFetchLatest_EmptyList(t *testing.T) {
	client := &http.Client{Transport: &mockTransport{statusCode: http.StatusOK, body: `[]`}}
	g := NewGitHubUpdater(client)

	_, err := g.FetchLatest(context.Background(), models.ChannelStable)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no suitable release found")
}

func TestFindTargetAsset_LinuxArchSpecific(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo-x86_64.AppImage", BrowserDownloadURL: "https://example.com/arch", Size: 100},
	}
	got := findTargetAsset(assets, "linux", "amd64", "")
	require.NotNil(t, got)
	assert.Equal(t, "MedMemo-x86_64.AppImage", got.Name)
}

func TestFindTargetAsset_LinuxFallbackGeneric(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo.AppImage", BrowserDownloadURL: "https://example.com/generic", Size: 100},
	}
	got := findTargetAsset(assets, "linux", "amd64", "")
	require.NotNil(t, got)
	assert.Equal(t, "MedMemo.AppImage", got.Name)
}

func TestFindTargetAsset_LinuxPrefersArchOverGeneric(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo.AppImage", BrowserDownloadURL: "https://example.com/generic", Size: 100},
		{Name: "MedMemo-x86_64.AppImage", BrowserDownloadURL: "https://example.com/arch", Size: 100},
	}
	got := findTargetAsset(assets, "linux", "amd64", "")
	require.NotNil(t, got)
	assert.Equal(t, "MedMemo-x86_64.AppImage", got.Name)
}

func TestFindTargetAsset_LinuxNoMatch(t *testing.T) {
	assets := []githubAsset{
		{Name: "wrong-platform.zip", BrowserDownloadURL: "https://example.com/wrong", Size: 100},
	}
	got := findTargetAsset(assets, "linux", "amd64", "")
	assert.Nil(t, got)
}

// TestFindTargetAsset_Platforms 验证 Windows 与 Darwin 的资产匹配逻辑。
func TestFindTargetAsset_Platforms(t *testing.T) {
	t.Run("windows setup installer", func(t *testing.T) {
		assets := []githubAsset{
			{Name: "MedMemo-1.0.0-windows-amd64-setup.exe", BrowserDownloadURL: "https://example.com/setup", Size: 100},
		}
		got := findTargetAsset(assets, "windows", "amd64", "")
		require.NotNil(t, got)
		assert.Equal(t, "MedMemo-1.0.0-windows-amd64-setup.exe", got.Name)
	})

	t.Run("windows rejects bare exe", func(t *testing.T) {
		assets := []githubAsset{
			{Name: "MedMemo.exe", BrowserDownloadURL: "https://example.com/exe", Size: 100},
		}
		got := findTargetAsset(assets, "windows", "amd64", "")
		assert.Nil(t, got)
	})

	t.Run("darwin dmg", func(t *testing.T) {
		assets := []githubAsset{
			{Name: "MedMemo.dmg", BrowserDownloadURL: "https://example.com/dmg", Size: 100},
		}
		got := findTargetAsset(assets, "darwin", "arm64", "")
		require.NotNil(t, got)
		assert.Equal(t, "MedMemo.dmg", got.Name)
	})
}

func TestFindTargetAsset_WindowsInstaller(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo-amd64-installer.exe", BrowserDownloadURL: "https://example.com/installer", Size: 100},
		{Name: "MedMemo.exe", BrowserDownloadURL: "https://example.com/exe", Size: 100},
	}
	got := findTargetAsset(assets, "windows", "amd64", "")
	require.NotNil(t, got)
	assert.Equal(t, "MedMemo-amd64-installer.exe", got.Name)
}

func TestFindTargetAsset_WindowsRejectsBareExe(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo.exe", BrowserDownloadURL: "https://example.com/exe", Size: 100},
	}
	got := findTargetAsset(assets, "windows", "amd64", "")
	assert.Nil(t, got)
}

// TestFindTargetAsset_LinuxAppImagePrefersAppImage 验证 AppImage 安装方式优先匹配 AppImage 资产。
func TestFindTargetAsset_LinuxAppImagePrefersAppImage(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo_1.1.10_amd64.deb", BrowserDownloadURL: "https://example.com/deb", Size: 100},
		{Name: "MedMemo-x86_64.AppImage", BrowserDownloadURL: "https://example.com/appimage", Size: 100},
	}
	got := findTargetAsset(assets, "linux", "amd64", "appimage")
	require.NotNil(t, got)
	assert.Equal(t, "MedMemo-x86_64.AppImage", got.Name)
}

// TestFindTargetAsset_LinuxDebPrefersDeb 验证 DEB 安装方式优先匹配 .deb 资产。
func TestFindTargetAsset_LinuxDebPrefersDeb(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo-x86_64.AppImage", BrowserDownloadURL: "https://example.com/appimage", Size: 100},
		{Name: "MedMemo_1.1.10_amd64.deb", BrowserDownloadURL: "https://example.com/deb", Size: 100},
	}
	got := findTargetAsset(assets, "linux", "amd64", "deb")
	require.NotNil(t, got)
	assert.Equal(t, "MedMemo_1.1.10_amd64.deb", got.Name)
}

// TestFindTargetAsset_LinuxRPMPrefersRPM 验证 RPM 安装方式优先匹配 .rpm 资产。
func TestFindTargetAsset_LinuxRPMPrefersRPM(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo-x86_64.AppImage", BrowserDownloadURL: "https://example.com/appimage", Size: 100},
		{Name: "MedMemo-1.1.10-1.x86_64.rpm", BrowserDownloadURL: "https://example.com/rpm", Size: 100},
	}
	got := findTargetAsset(assets, "linux", "amd64", "rpm")
	require.NotNil(t, got)
	assert.Equal(t, "MedMemo-1.1.10-1.x86_64.rpm", got.Name)
}

// TestFindTargetAsset_LinuxUnknownFallbackAppImage 验证未知安装方式回退到 AppImage。
func TestFindTargetAsset_LinuxUnknownFallbackAppImage(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo-x86_64.AppImage", BrowserDownloadURL: "https://example.com/appimage", Size: 100},
	}
	got := findTargetAsset(assets, "linux", "amd64", "unknown")
	require.NotNil(t, got)
	assert.Equal(t, "MedMemo-x86_64.AppImage", got.Name)
}

// TestFindTargetAsset_LinuxDebNoFallbackToAppImage 验证 DEB 安装方式在未找到 .deb 时不回退到 AppImage。
func TestFindTargetAsset_LinuxDebNoFallbackToAppImage(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo-x86_64.AppImage", BrowserDownloadURL: "https://example.com/appimage", Size: 100},
	}
	got := findTargetAsset(assets, "linux", "amd64", "deb")
	assert.Nil(t, got)
}

// TestFindTargetAsset_DarwinAmd64PrefersX86_64DMG 验证 Intel Mac 优先匹配 x86_64 DMG。
func TestFindTargetAsset_DarwinAmd64PrefersX86_64DMG(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo_arm64.dmg", BrowserDownloadURL: "https://example.com/arm64", Size: 100},
		{Name: "MedMemo_x86_64.dmg", BrowserDownloadURL: "https://example.com/x86_64", Size: 100},
	}
	got := findTargetAsset(assets, "darwin", "amd64", "")
	require.NotNil(t, got)
	assert.Equal(t, "MedMemo_x86_64.dmg", got.Name)
}

// TestFindTargetAsset_DarwinArm64PrefersArm64DMG 验证 Apple Silicon Mac 优先匹配 arm64 DMG。
func TestFindTargetAsset_DarwinArm64PrefersArm64DMG(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo_x86_64.dmg", BrowserDownloadURL: "https://example.com/x86_64", Size: 100},
		{Name: "MedMemo_arm64.dmg", BrowserDownloadURL: "https://example.com/arm64", Size: 100},
	}
	got := findTargetAsset(assets, "darwin", "arm64", "")
	require.NotNil(t, got)
	assert.Equal(t, "MedMemo_arm64.dmg", got.Name)
}

// TestFindTargetAsset_DarwinFallbackToGenericDMG 验证无架构特定 DMG 时回退到通用 .dmg。
func TestFindTargetAsset_DarwinFallbackToGenericDMG(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo.dmg", BrowserDownloadURL: "https://example.com/generic", Size: 100},
	}
	got := findTargetAsset(assets, "darwin", "arm64", "")
	require.NotNil(t, got)
	assert.Equal(t, "MedMemo.dmg", got.Name)
}

// TestFindTargetAsset_DarwinNoMatch 验证无 DMG 资产时返回 nil。
func TestFindTargetAsset_DarwinNoMatch(t *testing.T) {
	assets := []githubAsset{
		{Name: "MedMemo-x86_64.AppImage", BrowserDownloadURL: "https://example.com/linux", Size: 100},
	}
	got := findTargetAsset(assets, "darwin", "arm64", "")
	assert.Nil(t, got)
}

// TestMatchesPlatform_DarwinArchX86_64 验证 x86_64 DMG 匹配 amd64 架构。
func TestMatchesPlatform_DarwinArchX86_64(t *testing.T) {
	assert.True(t, matchesPlatform("MedMemo_x86_64.dmg", "darwin", "amd64", ""))
}

// TestMatchesPlatform_DarwinArchArm64 验证 arm64 DMG 匹配 arm64 架构。
func TestMatchesPlatform_DarwinArchArm64(t *testing.T) {
	assert.True(t, matchesPlatform("MedMemo_arm64.dmg", "darwin", "arm64", ""))
}

// TestMatchesPlatform_DarwinArchCrossMismatch 验证跨架构 DMG 不匹配。
func TestMatchesPlatform_DarwinArchCrossMismatch(t *testing.T) {
	assert.False(t, matchesPlatform("MedMemo_x86_64.dmg", "darwin", "arm64", ""))
}

// TestMatchArch 验证架构识别支持 amd64/x86_64 与 arm64/aarch64。
func TestMatchArch(t *testing.T) {
	tests := []struct {
		name   string
		goarch string
		asset  string
		want   bool
	}{
		{"amd64 exact", "amd64", "MedMemo-amd64.AppImage", true},
		{"amd64 x86_64 alias", "amd64", "MedMemo-x86_64.AppImage", true},
		{"arm64 exact", "arm64", "MedMemo-arm64.dmg", true},
		{"arm64 aarch64 alias", "arm64", "MedMemo-aarch64.dmg", true},
		{"mismatch", "amd64", "MedMemo-arm64.dmg", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, matchArch(strings.ToLower(tt.asset), tt.goarch))
		})
	}
}

// TestFindChecksum_ChecksumsAssetButExtractionFails 验证存在 checksums.txt 但内容提取失败时返回空。
func TestFindChecksum_ChecksumsAssetButExtractionFails(t *testing.T) {
	g := NewGitHubUpdater(nil)
	assets := []githubAsset{
		{Name: "MedMemo-x86_64.AppImage", BrowserDownloadURL: "https://example.com/appimage", Size: 100},
		{Name: "checksums.txt", BrowserDownloadURL: "http://127.0.0.1:1/invalid", Size: 100},
	}
	got := g.findChecksum(context.Background(), assets, "MedMemo-x86_64.AppImage")
	assert.Empty(t, got)
}

// TestMatchesPlatform_UnsupportedOS 验证非支持平台返回 false。
func TestMatchesPlatform_UnsupportedOS(t *testing.T) {
	assert.False(t, matchesPlatform("MedMemo.AppImage", "freebsd", "amd64", ""))
}

// TestFindChecksum_NoChecksumsAsset 验证无 checksums.txt 资产时返回空字符串。
func TestFindChecksum_NoChecksumsAsset(t *testing.T) {
	g := NewGitHubUpdater(nil)
	assets := []githubAsset{
		{Name: "MedMemo-x86_64.AppImage", BrowserDownloadURL: "https://example.com/appimage", Size: 100},
	}
	got := g.findChecksum(context.Background(), assets, "MedMemo-x86_64.AppImage")
	assert.Empty(t, got)
}

// TestMatchArch_UnknownArch 验证自定义架构按原字符串匹配。
func TestMatchArch_UnknownArch(t *testing.T) {
	assert.True(t, matchArch("medmemo-ppc64le.appimage", "ppc64le"))
	assert.False(t, matchArch("medmemo-amd64.appimage", "ppc64le"))
}

// TestExtractChecksum_RequestFails 验证 checksums.txt 请求失败时返回空。
func TestExtractChecksum_RequestFails(t *testing.T) {
	g := NewGitHubUpdater(nil)
	got := g.extractChecksum(context.Background(), "http://127.0.0.1:1/invalid", "MedMemo.AppImage")
	assert.Empty(t, got)
}

// TestExtractChecksum_NonOKStatus 验证非 200 响应返回空。
func TestExtractChecksum_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	g := NewGitHubUpdater(server.Client())
	got := g.extractChecksum(context.Background(), server.URL, "MedMemo.AppImage")
	assert.Empty(t, got)
}

// TestExtractChecksum_ReadBodyFails 验证响应体读取失败时返回空。
func TestExtractChecksum_ReadBodyFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		// 写入后立即关闭连接，模拟读取异常
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
		}
	}))
	defer server.Close()

	g := NewGitHubUpdater(server.Client())
	got := g.extractChecksum(context.Background(), server.URL, "MedMemo.AppImage")
	assert.Empty(t, got)
}

// TestIsMandatory 验证强制更新判断规则。
func TestIsMandatory(t *testing.T) {
	g := &GitHubUpdater{}

	tests := []struct {
		name string
		rel  githubRelease
		want bool
	}{
		{"security in name", githubRelease{Name: "v1.1.8 Security Patch", Body: "fixes"}, true},
		{"security in body", githubRelease{Name: "v1.1.8", Body: "critical security fix"}, true},
		{"critical in name", githubRelease{Name: "v1.1.8 Critical", Body: "fixes"}, true},
		{"critical in body", githubRelease{Name: "v1.1.8", Body: "critical bugfix"}, true},
		{"normal release", githubRelease{Name: "v1.1.8", Body: "regular release"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, g.isMandatory(tt.rel))
		})
	}
}

// TestExtractChecksum 验证从 checksums.txt 中提取目标文件哈希。
func TestExtractChecksum(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		assetName string
		want      string
	}{
		{
			name:      "standard format",
			body:      "abc123  MedMemo.AppImage\ndef456  Other.exe\n",
			assetName: "MedMemo.AppImage",
			want:      "abc123",
		},
		{
			name:      "binary prefix",
			body:      "abc123 *MedMemo.AppImage\n",
			assetName: "MedMemo.AppImage",
			want:      "abc123",
		},
		{
			name:      "not found",
			body:      "abc123  Other.AppImage\n",
			assetName: "MedMemo.AppImage",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			g := NewGitHubUpdater(server.Client())
			got := g.extractChecksum(context.Background(), server.URL, tt.assetName)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestVerifyChecksum 验证 SHA256 校验逻辑。
func TestVerifyChecksum(t *testing.T) {
	g := &GitHubUpdater{}

	t.Run("empty checksum skips", func(t *testing.T) {
		err := g.VerifyChecksum("/nonexistent", "")
		require.NoError(t, err)
	})

	t.Run("valid checksum", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.txt")
		data := []byte("hello")
		require.NoError(t, os.WriteFile(path, data, 0644))

		h := sha256.New()
		_, _ = h.Write(data)
		expected := hex.EncodeToString(h.Sum(nil))

		require.NoError(t, g.VerifyChecksum(path, expected))
	})

	t.Run("mismatch checksum", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "test.txt")
		require.NoError(t, os.WriteFile(path, []byte("hello"), 0644))

		err := g.VerifyChecksum(path, "0000000000000000000000000000000000000000000000000000000000000000")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "checksum mismatch")
	})

	t.Run("missing file", func(t *testing.T) {
		err := g.VerifyChecksum("/nonexistent/path", "abc")
		require.Error(t, err)
	})
}

// TestDownload 验证更新包下载与路径安全校验。
func TestDownload(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("update payload"))
		}))
		defer server.Close()

		g := NewGitHubUpdater(server.Client())
		dest := filepath.Join(t.TempDir(), "MedMemo-v0.2.0.AppImage")
		err := g.Download(context.Background(), server.URL, dest, nil)
		require.NoError(t, err)

		data, err := os.ReadFile(dest)
		require.NoError(t, err)
		assert.Equal(t, "update payload", string(data))
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		g := NewGitHubUpdater(server.Client())
		dest := filepath.Join(t.TempDir(), "MedMemo.AppImage")
		err := g.Download(context.Background(), server.URL, dest, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "download returned 500")
	})

	t.Run("destination is directory", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		g := NewGitHubUpdater(server.Client())
		dest := t.TempDir()
		err := g.Download(context.Background(), server.URL, dest, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "download path is a directory")
	})
}
