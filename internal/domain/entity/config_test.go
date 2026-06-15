package entity

import (
	"errors"
	"testing"

	"github.com/hzhan516/medmemo/pkg/models"
)

func TestAppConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  models.AppConfig
		wantErr error
	}{
		{
			name:    "empty data dir",
			config:  models.AppConfig{},
			wantErr: ErrInvalidConfig,
		},
		{
			name:    "valid config",
			config:  models.AppConfig{DataDir: "/tmp/medmemo"},
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
