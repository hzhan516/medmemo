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
