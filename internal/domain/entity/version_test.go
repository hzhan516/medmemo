package entity

import (
	"testing"
	"time"
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
		{"with prerelease tag", "v1.0.0-alpha", "v1.0.0", false, false},
		{"with prerelease tag remote newer", "v1.0.0", "v1.0.1-beta", true, false},
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
	if s.Channel != ChannelBeta {
		t.Errorf("expected default channel to be beta, got %s", s.Channel)
	}
	if s.SkipVersion != "" {
		t.Error("expected SkipVersion to be empty")
	}
}
