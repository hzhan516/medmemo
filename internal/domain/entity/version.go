// Package entity 定义更新相关的领域模型。
// 遵循 AGENTS.md 零外部依赖铁律，pkg/models 是唯一允许的外部依赖。
package entity

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hzhan516/medmemo/pkg/models"
)

// UpdateInfo 描述一次可更新的版本信息。
// 由 GitHub Releases API 响应映射而来，经领域校验后传递给前端。
type UpdateInfo struct {
	Version         string               `json:"version"`          // GitHub tag，如 "v1.1.3-Pre-release-build.57"
	DisplayVersion  string               `json:"display_version"`  // UI 展示版本，如 "v1.1.3-Pre-release-build.57"
	Name            string               `json:"name"`             // 发布标题
	Body            string               `json:"body"`             // 发布说明（Markdown 格式）
	PublishedAt     time.Time            `json:"published_at"`     // 发布时间
	DownloadURL     string               `json:"download_url"`     // 对应平台产物的下载地址
	Checksum        string               `json:"checksum"`         // SHA256 校验值
	Mandatory       bool                 `json:"mandatory"`        // 是否为强制更新（安全补丁）
	Channel         models.UpdateChannel `json:"channel"`          // 用户选择的检查通道
	Prerelease      bool                 `json:"prerelease"`       // 该 release 本身是否为 prerelease
	BuildNumber     string               `json:"build_number"`     // 构建号
	PreReleaseLabel string               `json:"prerelease_label"` // 预发布标签，如 "Pre-release"
}

// HasUpdate 语义化版本比较：remote 是否比 current 更新。
// 支持 "v" 前缀、四段版本号（如 1.1.2.54）、build 后缀（如 1.1.2-build.54）。
// 当核心版本号相同时，优先比较 build 号；无法比较时若完整字符串不同则视为有更新。
func HasUpdate(current, remote string) (bool, error) {
	cv, err := parseSemver(current)
	if err != nil {
		return false, fmt.Errorf("failed to parse current version %q: %w", current, err)
	}
	rv, err := parseSemver(remote)
	if err != nil {
		return false, fmt.Errorf("failed to parse remote version %q: %w", remote, err)
	}

	for i := 0; i < 3; i++ {
		if rv.core[i] > cv.core[i] {
			return true, nil
		}
		if rv.core[i] < cv.core[i] {
			return false, nil
		}
	}

	// 核心版本号相同，优先比较 build 号
	if rv.build != -1 && cv.build != -1 {
		if rv.build > cv.build {
			return true, nil
		}
		if rv.build < cv.build {
			return false, nil
		}
		// build 号相同视为同版本（忽略 format 差异，如 1.1.2.54 与 1.1.2-build.54）
		return false, nil
	}

	// 至少一边无 build 号，回退到完整字符串比较
	currentClean := strings.TrimPrefix(current, "v")
	remoteClean := strings.TrimPrefix(remote, "v")
	return currentClean != remoteClean, nil
}

// semver 内部表示，含核心三段式版本号与可选的 build 号。
type semver struct {
	core  [3]int // major, minor, patch
	build int    // build/revision 序号，-1 表示无 build 号
}

// parseSemver 解析语义化版本字符串，支持以下格式：
//   - "v" 前缀
//   - 1~3 段核心版本（缺失段补 0）
//   - 四段版本号：第 4 段作为 build 号（如 1.1.2.54）
//   - build 后缀：-build.N（如 1.1.2-build.54）
//   - 预发布标签：-alpha、-Pre-release-build.N 等（build 号仍可提取）
func parseSemver(v string) (semver, error) {
	v = strings.TrimPrefix(v, "v")

	var s semver
	s.build = -1

	// 分离主版本号与后缀
	raw := v
	if idx := strings.Index(raw, "+"); idx != -1 {
		// 构建元数据（+build）不参与版本比较，直接截断
		raw = raw[:idx]
	}

	if idx := strings.Index(raw, "-"); idx != -1 {
		preStr := raw[idx+1:]
		raw = raw[:idx]

		if strings.HasPrefix(preStr, "build.") {
			// 标准 build 后缀，如 1.1.2-build.54
			if n, err := strconv.Atoi(preStr[len("build."):]); err == nil {
				s.build = n
			}
		} else {
			// 其他预发布标签，尝试提取其中可能包含的 build 号
			// 如 Pre-release-build.53
			if buildIdx := strings.LastIndex(preStr, "build."); buildIdx != -1 {
				if n, err := strconv.Atoi(preStr[buildIdx+len("build."):]); err == nil {
					s.build = n
				}
			}
		}
	}

	parts := strings.Split(raw, ".")
	if len(parts) > 4 {
		return semver{}, fmt.Errorf("invalid semver format: too many segments, got %q", v)
	}

	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return semver{}, fmt.Errorf("invalid version component %q: %w", p, err)
		}
		if i < 3 {
			s.core[i] = n
		} else {
			// 第 4 段作为 build 号，覆盖后缀中可能提取的 build 号
			s.build = n
		}
	}

	return s, nil
}

// IsStableVersion 判断版本是否为稳定版。
//
// 以下格式视为稳定版：
//   - 三段或四段纯数字版本（如 v1.0.1、1.1.2.54）
//   - 带 -build.N 后缀的版本（如 1.1.2-build.54）
//
// 以下格式视为非稳定版：
//   - 含非 build 预发布标签（如 -alpha、-Pre-release-build.53）
//   - 1~2 段版本号
//   - 含构建元数据（如 +build123）
func IsStableVersion(v string) bool {
	v = strings.TrimPrefix(v, "v")

	raw := v
	if idx := strings.Index(raw, "+"); idx != -1 {
		return false
	}

	if idx := strings.Index(raw, "-"); idx != -1 {
		preStr := raw[idx+1:]
		if !strings.HasPrefix(preStr, "build.") {
			return false
		}
		raw = raw[:idx]
	}

	parts := strings.Split(raw, ".")
	if len(parts) < 3 || len(parts) > 4 {
		return false
	}

	for _, p := range parts {
		if _, err := strconv.Atoi(p); err != nil {
			return false
		}
	}
	return true
}

// UpdateSettings 存储用户的更新偏好设置。
type UpdateSettings struct {
	CheckEnabled bool                 `json:"check_enabled"` // 是否启用自动检测
	Channel      models.UpdateChannel `json:"channel"`       // 当前选择的更新通道
	SkipVersion  string               `json:"skip_version"`  // 用户选择跳过的版本号
	LastChecked  time.Time            `json:"last_checked"`  // 上次检测时间
}

// DefaultUpdateSettings 返回默认更新设置。
// 默认启用检测，通道由构建时注入决定（正式版 stable，测试版 beta）。
func DefaultUpdateSettings() *UpdateSettings {
	return &UpdateSettings{
		CheckEnabled: true,
		Channel:      models.ChannelStable,
		SkipVersion:  "",
		LastChecked:  time.Time{},
	}
}

// ShouldCheck 判断当前是否应该执行更新检测。
// 当检测被关闭、或距离上次检测不足间隔时返回 false。
func (s *UpdateSettings) ShouldCheck(interval time.Duration) bool {
	if !s.CheckEnabled {
		return false
	}
	if s.LastChecked.IsZero() {
		return true
	}
	return time.Since(s.LastChecked) >= interval
}
