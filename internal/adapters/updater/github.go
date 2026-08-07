// Package updater 实现 GitHub Releases API 适配器。
// 将 GitHub REST API 响应转换为领域层 UpdateInfo 模型。
package updater

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/wire"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	infraUpdater "github.com/hzhan516/medmemo/internal/infrastructure/updater"
	"github.com/hzhan516/medmemo/pkg/models"
)

const (
	githubAPIBase      = "https://api.github.com"
	githubRepoOwner    = "hzhan516"
	githubRepoName     = "medmemo"
	defaultHTTPTimeout = 30 * time.Second
)

// GitHubUpdater 通过 GitHub Releases API 获取更新信息。
type GitHubUpdater struct {
	client     *http.Client
	apiBaseURL string // 可由同包测试覆盖
}

// NewGitHubUpdater 构造函数。
// 测试时可传入自定义 http.Client（如带 mock Transport）以模拟 API 响应。
func NewGitHubUpdater(client *http.Client) *GitHubUpdater {
	if client != nil {
		return &GitHubUpdater{client: client, apiBaseURL: githubAPIBase}
	}
	return &GitHubUpdater{
		client: &http.Client{
			Timeout: defaultHTTPTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			},
		},
		apiBaseURL: githubAPIBase,
	}
}

// NewDefaultHTTPClient 返回带默认超时的 HTTP 客户端，供 Wire 注入使用。
func NewDefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: defaultHTTPTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}
}

// getLinuxInstallKind 返回当前 Linux 安装方式，供资产匹配使用。
// 通过包级变量暴露，便于单测注入固定值。
var getLinuxInstallKind = func() string {
	exe, err := os.Executable()
	if err != nil {
		return infraUpdater.DetectInstallKind("")
	}
	return infraUpdater.DetectInstallKind(exe)
}

// Ensure GitHubUpdater 实现了 port.Updater 接口。
var _ port.Updater = (*GitHubUpdater)(nil)

// doGitHubGET 向 GitHub API 发送 GET 请求并返回响应体。
// 调用方负责关闭返回的 ReadCloser。
func (g *GitHubUpdater) doGitHubGET(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github api request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("github api returned %d: %s", resp.StatusCode, string(body))
	}
	return resp.Body, nil
}

// FetchLatest 查询 GitHub Releases API 获取适合当前通道的最新版本。
// stable 通道过滤掉 prerelease，beta 通道包含全部。
func (g *GitHubUpdater) FetchLatest(ctx context.Context, channel models.UpdateChannel) (*entity.UpdateInfo, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=30", g.apiBaseURL, githubRepoOwner, githubRepoName)

	body, err := g.doGitHubGET(ctx, apiURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()

	var releases []githubRelease
	if err := json.NewDecoder(body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode github response: %w", err)
	}

	// 收集所有通过通道过滤且包含当前平台产物的 release，再从中选出最高版本
	var best *entity.UpdateInfo
	for _, release := range releases {
		// 通道过滤：stable 通道跳过 prerelease
		if channel == models.ChannelStable && release.Prerelease {
			continue
		}

		// 匹配当前平台的资产文件
		asset, checksum := g.matchPlatformAsset(ctx, release.Assets)
		if asset == nil {
			continue
		}

		candidate := g.releaseToUpdateInfo(release, channel, asset, checksum)

		if best == nil {
			best = candidate
			continue
		}

		cmp, err := entity.CompareVersions(best.Version, candidate.Version)
		if err != nil {
			// 版本解析失败时跳过异常候选，避免中断整个流程
			continue
		}
		if cmp > 0 {
			best = candidate
		}
	}

	if best == nil {
		return nil, fmt.Errorf("no suitable release found for channel %s on platform %s/%s", channel, runtime.GOOS, runtime.GOARCH)
	}
	return best, nil
}

// FetchByTag 根据指定 tag 查询 GitHub Release，用于下载用户点击的特定版本。
func (g *GitHubUpdater) FetchByTag(ctx context.Context, tag string) (*entity.UpdateInfo, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s", g.apiBaseURL, githubRepoOwner, githubRepoName, url.PathEscape(tag))

	body, err := g.doGitHubGET(ctx, apiURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()

	var release githubRelease
	if err := json.NewDecoder(body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode github response: %w", err)
	}

	asset, checksum := g.matchPlatformAsset(ctx, release.Assets)
	if asset == nil {
		return nil, fmt.Errorf("no suitable asset found for tag %s on platform %s/%s", tag, runtime.GOOS, runtime.GOARCH)
	}

	channel := models.ChannelStable
	if release.Prerelease {
		channel = models.ChannelBeta
	}

	return g.releaseToUpdateInfo(release, channel, asset, checksum), nil
}

// Download 下载指定 URL 的资产到本地路径，支持进度回调。
// Audit: RR-002 filepath.Clean + base directory validation prevents path traversal
func (g *GitHubUpdater) Download(ctx context.Context, url, destPath string, progress func(downloaded, total int64)) error {
	// 路径安全校验：防止恶意文件名导致目录穿越
	cleanPath := filepath.Clean(destPath)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve download path: %w", err)
	}

	// 获取目标目录的绝对路径并校验写入路径是否在其下
	destDir := filepath.Dir(absPath)
	destDirAbs, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("failed to resolve destination directory: %w", err)
	}

	// 确保目标路径不是目录（防止恶意路径覆盖目录）
	info, err := os.Stat(absPath)
	if err == nil && info.IsDir() {
		return fmt.Errorf("download path is a directory: %s", absPath)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	// 创建目标文件（使用已校验的绝对路径）
	if err := os.MkdirAll(destDirAbs, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	out, err := os.Create(absPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = out.Close() }()

	total := resp.ContentLength
	reader := &port.ProgressReader{
		Reader:   resp.Body,
		Total:    total,
		Callback: progress,
	}

	if _, err := io.Copy(out, reader); err != nil {
		_ = os.Remove(absPath)
		return fmt.Errorf("failed to write download: %w", err)
	}

	return nil
}

// VerifyChecksum 校验本地文件 SHA256。
func (g *GitHubUpdater) VerifyChecksum(path, expectedSHA256 string) error {
	if expectedSHA256 == "" {
		return fmt.Errorf("checksum information is missing")
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expectedSHA256) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, actual)
	}
	return nil
}

// matchPlatformAsset 根据当前平台匹配对应的 Release 资产文件。
// 同时查找配套的 checksums.txt 提取 SHA256 值。
func (g *GitHubUpdater) matchPlatformAsset(ctx context.Context, assets []githubAsset) (*githubAsset, string) {
	installKind := ""
	if runtime.GOOS == "linux" {
		installKind = getLinuxInstallKind()
	}
	targetAsset := findTargetAsset(assets, runtime.GOOS, runtime.GOARCH, installKind)
	checksum := ""
	if targetAsset != nil {
		checksum = g.findChecksum(ctx, assets, targetAsset.Name)
	}
	return targetAsset, checksum
}

// findTargetAsset 按平台与安装方式匹配最佳 Release 资产。
func findTargetAsset(assets []githubAsset, goos, goarch, installKind string) *githubAsset {
	for i := range assets {
		a := &assets[i]
		if matchesPlatform(a.Name, goos, goarch, installKind) {
			return a
		}
	}
	// Linux 回退：未匹配到带架构的 AppImage 时取任意 .AppImage（仅 AppImage 或未知安装方式）
	if goos == "linux" && (installKind == "appimage" || installKind == "unknown" || installKind == "") {
		if fallback := findLinuxFallback(assets); fallback != nil {
			return fallback
		}
	}
	// Darwin 回退：未匹配到架构特定 DMG 时取任意 .dmg（向后兼容）
	if goos == "darwin" {
		for i := range assets {
			if strings.HasSuffix(strings.ToLower(assets[i].Name), ".dmg") {
				return &assets[i]
			}
		}
	}
	return nil
}

// matchesPlatform 检查资产文件名是否匹配当前平台、架构与 Linux 安装方式。
func matchesPlatform(name, goos, goarch, installKind string) bool {
	name = strings.ToLower(name)
	switch goos {
	case "linux":
		switch installKind {
		case "deb":
			return strings.HasSuffix(name, ".deb")
		case "rpm":
			return strings.HasSuffix(name, ".rpm")
		default:
			return strings.Contains(name, "appimage") && matchArch(name, goarch)
		}
	case "darwin":
		return strings.HasSuffix(name, ".dmg") && matchArch(name, goarch)
	case "windows":
		return strings.HasSuffix(name, ".exe") &&
			(strings.Contains(name, "setup") || strings.Contains(name, "installer"))
	}
	return false
}

// findLinuxFallback 在未匹配到带架构的 AppImage 时回退到任意 .AppImage。
func findLinuxFallback(assets []githubAsset) *githubAsset {
	for i := range assets {
		if strings.HasSuffix(strings.ToLower(assets[i].Name), ".appimage") {
			return &assets[i]
		}
	}
	return nil
}

// findChecksum 从 assets 中查找 checksums.txt 并提取目标文件的 SHA256。
func (g *GitHubUpdater) findChecksum(ctx context.Context, assets []githubAsset, targetName string) string {
	for i := range assets {
		if strings.Contains(strings.ToLower(assets[i].Name), "checksums") {
			return g.extractChecksum(ctx, assets[i].BrowserDownloadURL, targetName)
		}
	}
	return ""
}

// matchArch 检查文件名是否包含目标架构标识。
// 注意：调用方（matchesPlatform）已将 name 转为小写，此处无需重复。
func matchArch(name, goarch string) bool {
	switch goarch {
	case "amd64":
		return strings.Contains(name, "amd64") || strings.Contains(name, "x86_64")
	case "arm64":
		return strings.Contains(name, "arm64") || strings.Contains(name, "aarch64")
	default:
		return strings.Contains(name, goarch)
	}
}

// extractChecksum 从 checksums.txt 中提取指定文件名的 SHA256 值。
func (g *GitHubUpdater) extractChecksum(ctx context.Context, checksumsURL, assetName string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumsURL, nil)
	if err != nil {
		return ""
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			// 格式通常为 "<hash>  <filename>" 或 "<hash> *<filename>"
			fname := strings.TrimPrefix(parts[1], "*")
			if fname == assetName {
				return parts[0]
			}
		}
	}
	return ""
}

// isMandatory 判断是否为强制更新。
// 规则：版本号包含 "security" 或 "critical" 标签时视为强制。
func (g *GitHubUpdater) isMandatory(release githubRelease) bool {
	lowerName := strings.ToLower(release.Name)
	lowerBody := strings.ToLower(release.Body)
	return strings.Contains(lowerName, "security") ||
		strings.Contains(lowerBody, "security") ||
		strings.Contains(lowerName, "critical") ||
		strings.Contains(lowerBody, "critical")
}

// releaseToUpdateInfo 将 GitHub Release 转换为领域层 UpdateInfo。
func (g *GitHubUpdater) releaseToUpdateInfo(release githubRelease, channel models.UpdateChannel, asset *githubAsset, checksum string) *entity.UpdateInfo {
	version := models.ParseAppVersion(release.TagName)
	displayVersion := version.DisplayVersion
	if displayVersion == "" {
		// 解析失败时回退到发布标题，避免 UI 展示空值
		displayVersion = release.Name
	}

	return &entity.UpdateInfo{
		Version:         release.TagName,
		DisplayVersion:  displayVersion,
		Name:            release.Name,
		Body:            release.Body,
		PublishedAt:     release.PublishedAt,
		DownloadURL:     asset.BrowserDownloadURL,
		Checksum:        checksum,
		Mandatory:       g.isMandatory(release),
		Channel:         channel,
		Prerelease:      release.Prerelease,
		BuildNumber:     version.BuildNumber,
		PreReleaseLabel: version.PrereleaseLabel,
	}
}

// githubRelease 表示 GitHub API 返回的 Release 结构。
type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

// githubAsset 表示 GitHub Release 中的单个资产文件。
type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// ProviderSet 供 Wire 使用的 Provider 集合。
var ProviderSet = wire.NewSet(
	NewGitHubUpdater,
	NewDefaultHTTPClient,
	wire.Bind(new(port.Updater), new(*GitHubUpdater)),
	NewInstallerAdapter,
	wire.Bind(new(port.Installer), new(*InstallerAdapter)),
)
