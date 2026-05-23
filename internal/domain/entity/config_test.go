package entity

import (
	"errors"
	"testing"
)

func TestAppConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  AppConfig
		wantErr error
	}{
		{
			name:    "empty data dir",
			config:  AppConfig{},
			wantErr: ErrInvalidConfig,
		},
		{
			name:    "valid config",
			config:  AppConfig{DataDir: "/tmp/medmemo"},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
