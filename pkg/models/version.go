// Package models 定义跨层共享的纯数据类型。
package models

import (
	"fmt"
	"regexp"
	"strings"
)

// UpdateChannel 已在 update_channel.go 中定义，此处复用。

// AppVersion 是应用版本的统一解析结果，供 Wails 绑定、Updater、Config 共享。
// 内部比较使用带 v 前缀的 Version，UI 展示使用 DisplayVersion。
type AppVersion struct {
	Version         string        `json:"version"`          // 内部比较版本，统一带 v 前缀
	DisplayVersion  string        `json:"display_version"`  // UI 展示版本，不含 build 号
	Channel         UpdateChannel `json:"channel"`          // stable / beta
	Prerelease      bool          `json:"prerelease"`       // 是否预发布
	PrereleaseLabel string        `json:"prerelease_label"` // 预发布标签，如 Pre-release
	BuildNumber     string        `json:"build_number"`     // 构建号
}

// 四段版本号：最后一段视为 build 号。
var fourSegmentRE = regexp.MustCompile(`^v(\d+\.\d+\.\d+)\.(\d+)$`)

// 标准三段版本号及后续可选标签。
var versionCoreRE = regexp.MustCompile(`^v(\d+\.\d+\.\d+)(.*)$`)

// ParseAppVersion 解析版本字符串为结构化版本对象。
// 支持正式版、预发布版、build 后缀、四段版本号以及 dev 版本。
func ParseAppVersion(v string) AppVersion {
	v = strings.TrimSpace(v)
	if v == "" {
		// 空字符串兜底，避免下游比较时出现零值歧义
		return AppVersion{
			Version:        "v0.0.0",
			DisplayVersion: "v0.0.0",
			Channel:        ChannelStable,
		}
	}

	if strings.EqualFold(v, "dev") {
		// dev 版本不参与语义比较，独立展示
		return AppVersion{
			Version:        "dev",
			DisplayVersion: "dev",
			Channel:        ChannelStable,
		}
	}

	normalized := v
	if !strings.HasPrefix(strings.ToLower(normalized), "v") {
		normalized = "v" + normalized
	}

	// 先尝试四段版本号，避免与核心正则冲突
	if m := fourSegmentRE.FindStringSubmatch(normalized); m != nil {
		base := m[1]
		build := m[2]
		return AppVersion{
			Version:        fmt.Sprintf("v%s-build.%s", base, build),
			DisplayVersion: "v" + base,
			Channel:        ChannelBeta,
			Prerelease:     true,
			BuildNumber:    build,
		}
	}

	m := versionCoreRE.FindStringSubmatch(normalized)
	if m == nil {
		// 无法识别时透传原值，按稳定通道处理
		return AppVersion{
			Version:        normalized,
			DisplayVersion: normalized,
			Channel:        ChannelStable,
		}
	}

	base := m[1]
	rest := m[2]

	buildNumber := ""
	prereleaseLabel := ""

	if idx := strings.LastIndex(rest, "-build."); idx >= 0 {
		buildNumber = rest[idx+len("-build."):]
		rest = rest[:idx]
	}

	// 剩余部分若存在，即为预发布标签
	if strings.HasPrefix(rest, "-") {
		prereleaseLabel = rest[1:]
	} else if rest != "" {
		prereleaseLabel = rest
	}

	display := "v" + base
	if prereleaseLabel != "" {
		display = display + "-" + prereleaseLabel
	}

	prerelease := buildNumber != "" || prereleaseLabel != ""
	channel := ChannelStable
	if prerelease {
		channel = ChannelBeta
	}

	return AppVersion{
		Version:         normalized,
		DisplayVersion:  display,
		Channel:         channel,
		Prerelease:      prerelease,
		PrereleaseLabel: prereleaseLabel,
		BuildNumber:     buildNumber,
	}
}
