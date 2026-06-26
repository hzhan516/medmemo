package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestParseAppVersion_EdgeCases 覆盖边界与异常输入。
func TestParseAppVersion_EdgeCases(t *testing.T) {
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
			name:           "uppercase_V_prefix_preserved",
			input:          "V1.1.8",
			wantVersion:    "V1.1.8",
			wantDisplay:    "V1.1.8",
			wantChannel:    ChannelStable,
			wantPrerelease: false,
		},
		{
			name:            "whitespace_around_version",
			input:           "  1.1.8-build.123  ",
			wantVersion:     "v1.1.8-build.123",
			wantDisplay:     "v1.1.8",
			wantChannel:     ChannelBeta,
			wantPrerelease:  true,
			wantBuildNumber: "123",
		},
		{
			name:            "four_segment_with_v_prefix",
			input:           "v1.1.8.123",
			wantVersion:     "v1.1.8-build.123",
			wantDisplay:     "v1.1.8",
			wantChannel:     ChannelBeta,
			wantPrerelease:  true,
			wantBuildNumber: "123",
		},
		{
			name:           "unrecognized_string_passthrough_with_v_prefix",
			input:          "not-a-version",
			wantVersion:    "vnot-a-version",
			wantDisplay:    "vnot-a-version",
			wantChannel:    ChannelStable,
			wantPrerelease: false,
		},
		{
			name:            "prerelease_label_without_v",
			input:           "1.1.8-RC1",
			wantVersion:     "v1.1.8-RC1",
			wantDisplay:     "v1.1.8-RC1",
			wantChannel:     ChannelBeta,
			wantPrerelease:  true,
			wantLabel:       "RC1",
			wantBuildNumber: "",
		},
		{
			name:            "prerelease_label_with_build_number",
			input:           "v1.1.8-alpha-build.7",
			wantVersion:     "v1.1.8-alpha-build.7",
			wantDisplay:     "v1.1.8-alpha",
			wantChannel:     ChannelBeta,
			wantPrerelease:  true,
			wantLabel:       "alpha",
			wantBuildNumber: "7",
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

// TestAppVersion_JSONSerialization 验证 AppVersion 可完整序列化与反序列化。
func TestAppVersion_JSONSerialization(t *testing.T) {
	t.Parallel()
	v := AppVersion{
		Version:         "v1.1.8-Pre-release-build.123",
		DisplayVersion:  "v1.1.8-Pre-release",
		Channel:         ChannelBeta,
		Prerelease:      true,
		PrereleaseLabel: "Pre-release",
		BuildNumber:     "123",
	}

	data, err := json.Marshal(v)
	require.NoError(t, err)

	var got AppVersion
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, v, got)
}
