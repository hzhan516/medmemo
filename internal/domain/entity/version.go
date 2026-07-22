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

// CompareVersions 语义化版本比较。
// 返回值大于 0 表示 b 比 a 新，小于 0 表示 a 比 b 新，等于 0 表示相同。
// 支持 "v" 前缀、四段版本号、build 后缀以及带尾号的预发布标签（如 rc.12）。
// 比较顺序：核心版本号 > build 号 > 预发布标签等级 > 同标签尾号 > 字符串回退。
func CompareVersions(a, b string) (int, error) {
	av, err := parseSemver(a)
	if err != nil {
		return 0, fmt.Errorf("failed to parse version %q: %w", a, err)
	}
	bv, err := parseSemver(b)
	if err != nil {
		return 0, fmt.Errorf("failed to parse version %q: %w", b, err)
	}

	for i := 0; i < 3; i++ {
		if av.core[i] < bv.core[i] {
			return 1, nil
		}
		if av.core[i] > bv.core[i] {
			return -1, nil
		}
	}

	// 核心版本相同，优先比较 build 号（保留 4 段版本号与 build 后缀兼容性）
	if av.build != -1 && bv.build != -1 {
		if av.build < bv.build {
			return 1, nil
		}
		if av.build > bv.build {
			return -1, nil
		}
		return 0, nil
	}

	// build 号路径无法比较时，对非 build 预发布标签进行等级比较
	if av.label != "build" && bv.label != "build" {
		// 稳定版（空标签）高于任何预发布版
		if av.label == "" && bv.label != "" {
			return -1, nil
		}
		if av.label != "" && bv.label == "" {
			return 1, nil
		}
		// 同标签预发布版比较尾号
		if av.label != "" && av.label == bv.label {
			if av.preN < bv.preN {
				return 1, nil
			}
			if av.preN > bv.preN {
				return -1, nil
			}
			return 0, nil
		}
	}

	// 至少一边无 build 号或存在 build 标签，回退到完整字符串比较。
	aClean := strings.TrimPrefix(a, "v")
	bClean := strings.TrimPrefix(b, "v")
	if aClean < bClean {
		return 1, nil
	}
	if aClean > bClean {
		return -1, nil
	}
	return 0, nil
}

// HasUpdate 语义化版本比较：remote 是否比 current 更新。
func HasUpdate(current, remote string) (bool, error) {
	cmp, err := CompareVersions(current, remote)
	if err != nil {
		return false, err
	}
	return cmp > 0, nil
}

// semver 内部表示，含核心三段式版本号、可选 build 号以及预发布标签信息。
type semver struct {
	core  [3]int // major, minor, patch
	build int    // build/revision 序号，-1 表示无 build 号
	label string // 预发布标签，空字符串表示稳定版
	preN  int    // 预发布标签尾号，-1 表示无尾号
}

// parseSemver 解析语义化版本字符串，支持以下格式：
//   - "v" 前缀
//   - 1~3 段核心版本（缺失段补 0）
//   - 四段版本号：第 4 段作为 build 号（如 1.1.2.54）
//   - build 后缀：-build.N（如 1.1.2-build.54）
//   - 带尾号的预发布标签：-rc.12、-Pre-release-build.86 等
//   - 无尾号的预发布标签：-alpha、-rc 等
func parseSemver(v string) (semver, error) {
	v = strings.TrimPrefix(v, "v")

	var s semver
	s.build = -1
	s.preN = -1

	// 分离主版本号与后缀
	raw := v
	if idx := strings.Index(raw, "+"); idx != -1 {
		// 构建元数据（+build）不参与版本比较，直接截断
		raw = raw[:idx]
	}

	if idx := strings.Index(raw, "-"); idx != -1 {
		preStr := raw[idx+1:]
		raw = raw[:idx]

		// 解析预发布标签与尾号：最后一个 "." 后若为数字则作为尾号
		if dotIdx := strings.LastIndex(preStr, "."); dotIdx != -1 {
			s.label = preStr[:dotIdx]
			if n, err := strconv.Atoi(preStr[dotIdx+1:]); err == nil {
				s.preN = n
			}
		} else {
			s.label = preStr
		}

		if strings.HasPrefix(preStr, "build.") {
			// 标准 build 后缀，如 1.1.2-build.54
			if s.preN != -1 {
				s.build = s.preN
			}
		} else {
			// 其他预发布标签，仍尝试提取其中可能包含的 build 号
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
