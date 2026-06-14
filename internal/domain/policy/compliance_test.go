package policy

import "testing"

func TestRiskLevel_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		level RiskLevel
		want  string
	}{
		{L1Blocked, "BLOCKED"},
		{L2Warning, "WARNING"},
		{L3Notice, "NOTICE"},
		{L4Normal, "NORMAL"},
		{RiskLevel(99), "UNKNOWN"},
		{RiskLevel(0), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
