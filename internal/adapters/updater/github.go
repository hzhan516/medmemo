// Package updater 实现 GitHub Releases API 适配器。
// 将 GitHub REST API 响应转换为领域层 UpdateInfo 模型。
package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/wire"
	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/internal/domain/entity"
)

const (
	githubAPIBase      = "https://api.github.com"
	githubRepoOwner    = "medmemo"
	githubRepoName     = "medmemo"
	defaultHTTPTimeout = 30 * time.Second
)

// GitHubUpdater 通过 GitHub Releases API 获取更新信息。
type GitHubUpdater struct {
	client *http.Client
}

// NewGitHubUpdater 构造函数。
func NewGitHubUpdater() *GitHubUpdater {
	return &GitHubUpdater{
		client: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
	}
}

// Ensure GitHubUpdater 实现了 port.Updater 接口。
var _ port.Updater = (*GitHubUpdater)(nil)

// FetchLatest 查询 GitHub Releases API 获取适合当前通道的最新版本。
// stable 通道过滤掉 prerelease，beta 通道包含全部。
func (g *GitHubUpdater) FetchLatest(ctx context.Context, channel entity.UpdateChannel) (*entity.UpdateInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=30", githubAPIBase, githubRepoOwner, githubRepoName)

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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github api returned %d: %s", resp.StatusCode, string(body))
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode github response: %w", err)
	}

	// 遍历 releases 列表，找到适合当前通道且包含当前平台产物的第一个 release
	for _, release := range releases {
		// 通道过滤：stable 通道跳过 prerelease
		if channel == entity.ChannelStable && release.Prerelease {
			continue
		}

		// 匹配当前平台的资产文件
		asset, checksum := g.matchPlatformAsset(release.Assets)
		if asset == nil {
			continue
		}

		info := &entity.UpdateInfo{
			Version:     release.TagName,
			Name:        release.Name,
			Body:        release.Body,
			PublishedAt: release.PublishedAt,
			DownloadURL: asset.BrowserDownloadURL,
			Checksum:    checksum,
			Mandatory:   g.isMandatory(release),
			Channel:     channel,
		}

		return info, nil
	}

	return nil, fmt.Errorf("no suitable release found for channel %s on platform %s/%s", channel, runtime.GOOS, runtime.GOARCH)
}

// Download 下载指定 URL 的资产到本地路径，支持进度回调。
func (g *GitHubUpdater) Download(ctx context.Context, url, destPath string, progress func(downloaded, total int64)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	// 创建目标文件
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	total := resp.ContentLength
	reader := &port.ProgressReader{
		Reader:   resp.Body,
		Total:    total,
		Callback: progress,
	}

	if _, err := io.Copy(out, reader); err != nil {
		_ = os.Remove(destPath)
		return fmt.Errorf("failed to write download: %w", err)
	}

	return nil
}

// VerifyChecksum 校验本地文件 SHA256。
func (g *GitHubUpdater) VerifyChecksum(path, expectedSHA256 string) error {
	if expectedSHA256 == "" {
		// 如果 Release 未提供 checksum，跳过校验（MVP 阶段允许）
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer f.Close()

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
func (g *GitHubUpdater) matchPlatformAsset(assets []githubAsset) (*githubAsset, string) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var targetAsset *githubAsset
	for i := range assets {
		a := &assets[i]
		name := strings.ToLower(a.Name)
		switch goos {
		case "linux":
			if strings.Contains(name, "appimage") && matchArch(name, goarch) {
				targetAsset = a
			}
		case "darwin":
			if strings.HasSuffix(name, ".dmg") {
				targetAsset = a
			}
		case "windows":
			if strings.Contains(name, "installer") && strings.HasSuffix(name, ".exe") {
				targetAsset = a
			}
		}
		if targetAsset != nil {
			break
		}
	}

	// 查找 checksums.txt 提取对应校验值
	checksum := ""
	if targetAsset != nil {
		for i := range assets {
			if strings.Contains(strings.ToLower(assets[i].Name), "checksums") {
				checksum = g.extractChecksum(assets[i].BrowserDownloadURL, targetAsset.Name)
				break
			}
		}
	}

	return targetAsset, checksum
}

// matchArch 检查文件名是否包含目标架构标识。
func matchArch(name, goarch string) bool {
	name = strings.ToLower(name)
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
// MVP 阶段简化实现：异步下载并解析。
func (g *GitHubUpdater) extractChecksum(checksumsURL, assetName string) string {
	resp, err := g.client.Get(checksumsURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
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
	wire.Bind(new(port.Updater), new(*GitHubUpdater)),
)
