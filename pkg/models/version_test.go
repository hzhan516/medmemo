package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseAppVersion 覆盖常见版本字符串的解析行为。
func TestParseAppVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		input           string
		wantVersion     string
		wantDisplay     string
		wantChannel     UpdateChannel
		wantPrerelease  bool
		wantLabel       string
		wantBuildNumber string
	}{
		{
			name:           "stable_with_v_prefix",
			input:          "v1.1.8",
			wantVersion:    "v1.1.8",
			wantDisplay:    "v1.1.8",
			wantChannel:    ChannelStable,
			wantPrerelease: false,
		},
		{
			name:           "stable_without_v_prefix",
			input:          "1.1.8",
			wantVersion:    "v1.1.8",
			wantDisplay:    "v1.1.8",
			wantChannel:    ChannelStable,
			wantPrerelease: false,
		},
		{
			name:            "prerelease_with_label_and_build",
			input:           "v1.1.8-Pre-release-build.123",
			wantVersion:     "v1.1.8-Pre-release-build.123",
			wantDisplay:     "v1.1.8-Pre-release",
			wantChannel:     ChannelBeta,
			wantPrerelease:  true,
			wantLabel:       "Pre-release",
			wantBuildNumber: "123",
		},
		{
			name:            "build_suffix_only",
			input:           "v1.1.8-build.123",
			wantVersion:     "v1.1.8-build.123",
			wantDisplay:     "v1.1.8",
			wantChannel:     ChannelBeta,
			wantPrerelease:  true,
			wantBuildNumber: "123",
		},
		{
			name:            "four_segment_version",
			input:           "1.1.8.123",
			wantVersion:     "v1.1.8-build.123",
			wantDisplay:     "v1.1.8",
			wantChannel:     ChannelBeta,
			wantPrerelease:  true,
			wantBuildNumber: "123",
		},
		{
			name:           "dev_version",
			input:          "dev",
			wantVersion:    "dev",
			wantDisplay:    "dev",
			wantChannel:    ChannelStable,
			wantPrerelease: false,
		},
		{
			name:           "empty_string_fallback",
			input:          "",
			wantVersion:    "v0.0.0",
			wantDisplay:    "v0.0.0",
			wantChannel:    ChannelStable,
			wantPrerelease: false,
		},
		{
			name:            "prerelease_label_without_build",
			input:           "v1.1.8-Pre-release",
			wantVersion:     "v1.1.8-Pre-release",
			wantDisplay:     "v1.1.8-Pre-release",
			wantChannel:     ChannelBeta,
			wantPrerelease:  true,
			wantLabel:       "Pre-release",
			wantBuildNumber: "",
		},
		{
			name:            "build_suffix_without_v_prefix",
			input:           "1.1.8-build.456",
			wantVersion:     "v1.1.8-build.456",
			wantDisplay:     "v1.1.8",
			wantChannel:     ChannelBeta,
			wantPrerelease:  true,
			wantBuildNumber: "456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAppVersion(tt.input)
			assert.Equal(t, tt.wantVersion, got.Version, "Version")
			assert.Equal(t, tt.wantDisplay, got.DisplayVersion, "DisplayVersion")
			assert.Equal(t, tt.wantChannel, got.Channel, "Channel")
			assert.Equal(t, tt.wantPrerelease, got.Prerelease, "Prerelease")
			assert.Equal(t, tt.wantLabel, got.PrereleaseLabel, "PrereleaseLabel")
			assert.Equal(t, tt.wantBuildNumber, got.BuildNumber, "BuildNumber")
		})
	}
}

// TestParseAppVersion_CaseInsensitiveDev 验证 dev 识别不区分大小写。
func TestParseAppVersion_CaseInsensitiveDev(t *testing.T) {
	t.Parallel()
	got := ParseAppVersion("DEV")
	assert.Equal(t, "dev", got.Version)
	assert.Equal(t, ChannelStable, got.Channel)
	assert.False(t, got.Prerelease)
}

// TestParseAppVersion_JSONRoundTrip 验证结构体字段 tag 可正常序列化。
func TestParseAppVersion_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	v := ParseAppVersion("v1.1.8-Pre-release-build.123")
	assert.Equal(t, "v1.1.8-Pre-release-build.123", v.Version)
	assert.Equal(t, "beta", string(v.Channel))
	assert.True(t, v.Prerelease)
}
