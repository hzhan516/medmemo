// Package entity 定义更新相关的领域模型。
// 遵循 AGENTS.md 零外部依赖铁律，仅使用 Go 标准库。
package entity

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// UpdateChannel 定义更新通道类型。
type UpdateChannel string

const (
	ChannelStable UpdateChannel = "stable" // 稳定版通道，仅包含正式发布版本
	ChannelBeta   UpdateChannel = "beta"   // 测试版通道，包含预发布版本
)

// UpdateInfo 描述一次可更新的版本信息。
// 由 GitHub Releases API 响应映射而来，经领域校验后传递给前端。
type UpdateInfo struct {
	Version     string        `json:"version"`      // 语义化版本号，如 "v0.5.0"
	Name        string        `json:"name"`         // 发布标题
	Body        string        `json:"body"`         // 发布说明（Markdown 格式）
	PublishedAt time.Time     `json:"published_at"` // 发布时间
	DownloadURL string        `json:"download_url"` // 对应平台产物的下载地址
	Checksum    string        `json:"checksum"`     // SHA256 校验值
	Mandatory   bool          `json:"mandatory"`    // 是否为强制更新（安全补丁）
	Channel     UpdateChannel `json:"channel"`      // 所属通道
}

// HasUpdate 语义化版本比较：remote 是否比 current 更新。
// 支持 "v" 前缀，比较 Major.Minor.Patch 三段式版本号。
// 当核心版本号相同时，若完整版本字符串不同（如不同 build 号），也视为有更新。
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
		if rv[i] > cv[i] {
			return true, nil
		}
		if rv[i] < cv[i] {
			return false, nil
		}
	}

	// 核心版本号相同，比较完整字符串（去掉 v 前缀）
	// 不同 build 号或预发布标签视为有更新
	currentClean := strings.TrimPrefix(current, "v")
	remoteClean := strings.TrimPrefix(remote, "v")
	if currentClean == remoteClean {
		return false, nil // 完全相同，无需更新
	}
	return true, nil
}

// semver 内部表示三段式版本号 [Major, Minor, Patch]。
type semver [3]int

// parseSemver 解析语义化版本字符串，支持 "v" 前缀、两段式/一段式、预发布标签。
// 预发布标签（如 -alpha、-beta）仅用于稳定版判断，不参与版本号数值比较。
func parseSemver(v string) (semver, error) {
	v = strings.TrimPrefix(v, "v")
	// 去掉预发布标签和构建元数据，仅保留核心版本号
	if idx := strings.IndexAny(v, "-+"); idx != -1 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	if len(parts) > 3 {
		return semver{}, fmt.Errorf("invalid semver format: expected 1~3 segments, got %q", v)
	}
	var s semver
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return semver{}, fmt.Errorf("invalid version component %q: %w", p, err)
		}
		s[i] = n
	}
	// 缺失段补 0
	return s, nil
}

// IsStableVersion 判断版本是否为稳定版。
// 三段式纯数字版本（如 v1.0.1）为稳定版；两段式（如 v1.0）、一段式（如 v1）
// 或含预发布标签（-alpha、-beta、-rc、-SNAPSHOT）均为测试版。
func IsStableVersion(v string) bool {
	v = strings.TrimPrefix(v, "v")
	// 含预发布标签或构建元数据 → 非稳定
	if strings.ContainsAny(v, "-+") {
		return false
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
	// 确保每段都是纯数字
	for _, p := range parts {
		if _, err := strconv.Atoi(p); err != nil {
			return false
		}
	}
	return true
}

// UpdateSettings 存储用户的更新偏好设置。
type UpdateSettings struct {
	CheckEnabled bool          `json:"check_enabled"` // 是否启用自动检测
	Channel      UpdateChannel `json:"channel"`       // 当前选择的更新通道
	SkipVersion  string        `json:"skip_version"`  // 用户选择跳过的版本号
	LastChecked  time.Time     `json:"last_checked"`  // 上次检测时间
}

// DefaultUpdateSettings 返回默认更新设置。
// 内测阶段默认启用检测、使用 beta 通道。
func DefaultUpdateSettings() *UpdateSettings {
	return &UpdateSettings{
		CheckEnabled: true,
		Channel:      ChannelBeta,
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
