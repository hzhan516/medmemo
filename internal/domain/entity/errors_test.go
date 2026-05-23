package entity

import (
	"errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrNotFound", ErrNotFound},
		{"ErrInvalidConfig", ErrInvalidConfig},
		{"ErrDuplicateEntry", ErrDuplicateEntry},
		{"ErrUnauthorized", ErrUnauthorized},
		{"ErrComplianceBlocked", ErrComplianceBlocked},
		{"ErrSensitiveDataLeak", ErrSensitiveDataLeak},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Error("expected non-nil sentinel error")
			}
			if !errors.Is(tt.err, tt.err) {
				t.Error("expected errors.Is to match itself")
			}
		})
	}
}
