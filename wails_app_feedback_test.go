package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetVersionInfo_Stable 验证正式版版本号被正确解析为结构化信息。
func TestGetVersionInfo_Stable(t *testing.T) {
	old := version
	version = "1.1.6"
	defer func() { version = old }()

	app := &WailsApp{}
	info, err := app.GetVersionInfo()
	require.NoError(t, err)

	assert.Equal(t, "v1.1.6", info.Version)
	assert.Equal(t, "v1.1.6", info.DisplayVersion)
	assert.Equal(t, "", info.BuildNumber)
	assert.Equal(t, "stable", info.Channel)
	assert.Equal(t, "", info.PrereleaseLabel)
	assert.False(t, info.Prerelease)
}

// TestGetVersionInfo_Prerelease 验证带预发布标签与 build 号的版本号解析正确。
func TestGetVersionInfo_Prerelease(t *testing.T) {
	old := version
	version = "v1.1.8-Pre-release-build.57"
	defer func() { version = old }()

	app := &WailsApp{}
	info, err := app.GetVersionInfo()
	require.NoError(t, err)

	assert.Equal(t, "v1.1.8-Pre-release-build.57", info.Version)
	assert.Equal(t, "v1.1.8-Pre-release", info.DisplayVersion)
	assert.Equal(t, "57", info.BuildNumber)
	assert.Equal(t, "beta", info.Channel)
	assert.Equal(t, "Pre-release", info.PrereleaseLabel)
	assert.True(t, info.Prerelease)
}

// TestGetVersionInfo_Dev 验证 dev 版本独立透传，不参与语义化解析。
func TestGetVersionInfo_Dev(t *testing.T) {
	old := version
	version = "dev"
	defer func() { version = old }()

	app := &WailsApp{}
	info, err := app.GetVersionInfo()
	require.NoError(t, err)

	assert.Equal(t, "dev", info.Version)
	assert.Equal(t, "dev", info.DisplayVersion)
	assert.Equal(t, "", info.BuildNumber)
	assert.Equal(t, "stable", info.Channel)
	assert.Equal(t, "", info.PrereleaseLabel)
	assert.False(t, info.Prerelease)
}

// TestProvideDefaultUpdateChannel 验证默认更新通道按构建版本正确推导。
func TestProvideDefaultUpdateChannel(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{"stable", "v1.1.8", "stable"},
		{"prerelease", "v1.1.8-Pre-release-build.123", "beta"},
		{"dev", "dev", "beta"},
		{"DEV_uppercase", "DEV", "beta"},
		{"four_segment", "v1.1.8.123", "beta"},
		{"empty_fallback", "", "stable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := version
			version = tt.version
			defer func() { version = old }()

			assert.Equal(t, tt.want, string(ProvideDefaultUpdateChannel()))
		})
	}
}

// TestGetVersionInfo_EdgeCases 验证 GetVersionInfo 对特殊版本号的处理。
func TestGetVersionInfo_EdgeCases(t *testing.T) {
	tests := []struct {
		name            string
		version         string
		wantVersion     string
		wantDisplay     string
		wantChannel     string
		wantPrerelease  bool
		wantBuildNumber string
		wantLabel       string
	}{
		{
			name:            "four_segment",
			version:         "1.1.8.57",
			wantVersion:     "v1.1.8-build.57",
			wantDisplay:     "v1.1.8",
			wantChannel:     "beta",
			wantPrerelease:  true,
			wantBuildNumber: "57",
		},
		{
			name:           "empty_fallback",
			version:        "",
			wantVersion:    "v0.0.0",
			wantDisplay:    "v0.0.0",
			wantChannel:    "stable",
			wantPrerelease: false,
		},
		{
			name:           "DEV_uppercase",
			version:        "DEV",
			wantVersion:    "dev",
			wantDisplay:    "dev",
			wantChannel:    "stable",
			wantPrerelease: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := version
			version = tt.version
			defer func() { version = old }()

			app := &WailsApp{}
			info, err := app.GetVersionInfo()
			require.NoError(t, err)
			assert.Equal(t, tt.wantVersion, info.Version)
			assert.Equal(t, tt.wantDisplay, info.DisplayVersion)
			assert.Equal(t, tt.wantChannel, info.Channel)
			assert.Equal(t, tt.wantPrerelease, info.Prerelease)
			assert.Equal(t, tt.wantBuildNumber, info.BuildNumber)
			assert.Equal(t, tt.wantLabel, info.PrereleaseLabel)
		})
	}
}
