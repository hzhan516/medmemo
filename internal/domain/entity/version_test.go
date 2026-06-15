package entity

import (
	"testing"
	"time"

	"github.com/hzhan516/medmemo/pkg/models"
)

func TestHasUpdate(t *testing.T) {
	tests := []struct {
		name    string
		current string
		remote  string
		want    bool
		wantErr bool
	}{
		{"major update", "v0.1.0", "v1.0.0", true, false},
		{"minor update", "v0.1.0", "v0.2.0", true, false},
		{"patch update", "v0.1.0", "v0.1.1", true, false},
		{"no update same", "v0.1.0", "v0.1.0", false, false},
		{"older version", "v0.2.0", "v0.1.0", false, false},
		{"without v prefix current", "0.1.0", "v0.2.0", true, false},
		{"without v prefix remote", "v0.1.0", "0.2.0", true, false},
		{"invalid current", "abc", "v0.2.0", false, true},
		{"invalid remote", "v0.1.0", "abc", false, true},
		{"invalid format", "v1.a.0", "v1.b.0", false, true},
		{"two segment current", "v1.0", "v1.0.1", true, false},
		{"two segment remote", "v1.0.0", "v1.1", true, false},
		{"one segment current", "v1", "v1.0.1", true, false},
		{"one segment same major", "v1", "v1", false, false},
		{"with prerelease tag", "v1.0.0-alpha", "v1.0.0", true, false},
		{"with prerelease tag remote newer", "v1.0.0", "v1.0.1-beta", true, false},
		{"same version different build", "0.1.0-build.10", "0.1.0-build.20", true, false},
		{"same version same build", "0.1.0-build.10", "0.1.0-build.10", false, false},
		{"stable to pre-release same core", "0.1.0", "0.1.0-Pre-release-build.5", true, false},
		{"pre-release to stable same core", "0.1.0-Pre-release-build.5", "0.1.0", true, false},
		// 四段版本号与 build 后缀兼容场景
		{"4-segment prerelease to 4-segment newer", "1.1.2-Pre-release-build.53", "1.1.2.54", true, false},
		{"4-segment same", "1.1.2.54", "1.1.2.54", false, false},
		{"4-segment newer build", "1.1.2.54", "1.1.2.55", true, false},
		{"4-segment avoid downgrade", "1.1.2.55", "1.1.2.54", false, false},
		{"v prefix to 4-segment", "v1.1.2", "1.1.2.54", true, false},
		{"build suffix incremental", "1.1.2-build.53", "1.1.2-build.54", true, false},
		{"build suffix same as 4-segment", "1.1.2-build.54", "1.1.2.54", false, false},
		{"stable to build suffix", "1.1.2", "1.1.2-build.1", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HasUpdate(tt.current, tt.remote)
			if (err != nil) != tt.wantErr {
				t.Errorf("HasUpdate(%q, %q) error = %v, wantErr %v", tt.current, tt.remote, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("HasUpdate(%q, %q) = %v, want %v", tt.current, tt.remote, got, tt.want)
			}
		})
	}
}

func TestUpdateSettingsShouldCheck(t *testing.T) {
	tests := []struct {
		name     string
		settings *UpdateSettings
		interval time.Duration
		want     bool
	}{
		{
			name:     "enabled first check",
			settings: &UpdateSettings{CheckEnabled: true, LastChecked: time.Time{}},
			interval: time.Hour,
			want:     true,
		},
		{
			name:     "disabled",
			settings: &UpdateSettings{CheckEnabled: false, LastChecked: time.Time{}},
			interval: time.Hour,
			want:     false,
		},
		{
			name:     "checked recently",
			settings: &UpdateSettings{CheckEnabled: true, LastChecked: time.Now().Add(-30 * time.Minute)},
			interval: time.Hour,
			want:     false,
		},
		{
			name:     "checked long ago",
			settings: &UpdateSettings{CheckEnabled: true, LastChecked: time.Now().Add(-2 * time.Hour)},
			interval: time.Hour,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.settings.ShouldCheck(tt.interval)
			if got != tt.want {
				t.Errorf("ShouldCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsStableVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{"stable patch", "v1.0.1", true},
		{"stable patch v1.0.2", "v1.0.2", true},
		{"stable patch v1.1.1", "v1.1.1", true},
		{"stable without v", "1.2.3", true},
		{"test two segment", "v1.0", false},
		{"test two segment v1.1", "v1.1", false},
		{"test one segment", "v1", false},
		{"alpha tag", "v1.0.0-alpha", false},
		{"beta tag", "v1.0.0-beta", false},
		{"rc tag", "v1.0.0-rc1", false},
		{"snapshot tag", "v1.0.0-SNAPSHOT", false},
		{"with build metadata", "v1.0.0+build123", false},
		{"dev version", "dev", false},
		{"empty string", "", false},
		{"mixed alphanumeric", "1.2.3a", false},
		// 四段版本号与 build 后缀场景
		{"4-segment stable", "1.1.2.54", true},
		{"4-segment with v", "v1.1.2.54", true},
		{"build suffix stable", "1.1.2-build.54", true},
		{"build suffix with v", "v1.1.2-build.54", true},
		{"pre-release with build", "1.1.2-Pre-release-build.53", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsStableVersion(tt.version)
			if got != tt.want {
				t.Errorf("IsStableVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestDefaultUpdateSettings(t *testing.T) {
	s := DefaultUpdateSettings()
	if !s.CheckEnabled {
		t.Error("expected CheckEnabled to be true")
	}
	if s.Channel != models.ChannelStable {
		t.Errorf("expected default channel to be stable, got %s", s.Channel)
	}
	if s.SkipVersion != "" {
		t.Error("expected SkipVersion to be empty")
	}
}
