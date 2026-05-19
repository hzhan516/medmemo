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
	return false, nil // 版本相同
}

// semver 内部表示三段式版本号 [Major, Minor, Patch]。
type semver [3]int

// parseSemver 解析语义化版本字符串，支持 "v" 前缀。
func parseSemver(v string) (semver, error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return semver{}, fmt.Errorf("invalid semver format: expected Major.Minor.Patch, got %q", v)
	}
	var s semver
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return semver{}, fmt.Errorf("invalid version component %q: %w", p, err)
		}
		s[i] = n
	}
	return s, nil
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
