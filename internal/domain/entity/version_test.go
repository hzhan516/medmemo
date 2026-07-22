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
		{"stable to pre-release same core", "0.1.0", "0.1.0-Pre-release-build.5", false, false},
		{"pre-release to stable same core", "0.1.0-Pre-release-build.5", "0.1.0", true, false},
		// rc 标签尾号比较
		{"rc tail newer", "v1.1.10-rc.12", "v1.1.10-rc.13", true, false},
		{"rc tail older", "v1.1.10-rc.13", "v1.1.10-rc.12", false, false},
		{"rc tail same", "v1.1.10-rc.12", "v1.1.10-rc.12", false, false},
		{"rc to stable same core", "v1.1.10-rc.12", "v1.1.10", true, false},
		{"stable to rc same core", "v1.1.10", "v1.1.10-rc.12", false, false},
		{"core differs with pre-release", "v1.1.9-Pre-release-build.86", "v1.1.10-rc.12", true, false},
		{"cross-label fallback", "v1.1.10-Pre-release-build.86", "v1.1.10-rc.12", true, false},
		{"cross-label fallback reverse", "v1.1.10-rc.12", "v1.1.10-Pre-release-build.86", false, false},
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

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		a       string
		b       string
		want    int
		wantErr bool
	}{
		{"major newer", "v0.1.0", "v1.0.0", 1, false},
		{"major older", "v1.0.0", "v0.1.0", -1, false},
		{"same version", "v0.1.0", "v0.1.0", 0, false},
		{"patch newer", "v0.1.0", "v0.1.1", 1, false},
		{"patch older", "v0.1.1", "v0.1.0", -1, false},
		{"rc tail newer", "v1.1.10-rc.12", "v1.1.10-rc.13", 1, false},
		{"rc tail older", "v1.1.10-rc.13", "v1.1.10-rc.12", -1, false},
		{"rc tail same", "v1.1.10-rc.12", "v1.1.10-rc.12", 0, false},
		{"rc to stable", "v1.1.10-rc.12", "v1.1.10", 1, false},
		{"stable to rc", "v1.1.10", "v1.1.10-rc.12", -1, false},
		{"build newer", "0.1.0-build.10", "0.1.0-build.20", 1, false},
		{"build same", "0.1.0-build.10", "0.1.0-build.10", 0, false},
		{"invalid a", "abc", "v0.1.0", 0, true},
		{"invalid b", "v0.1.0", "abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompareVersions(tt.a, tt.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompareVersions(%q, %q) error = %v, wantErr %v", tt.a, tt.b, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
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
