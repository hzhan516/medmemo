package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDesensitizationLevel_IsValid(t *testing.T) {
	t.Parallel()
	assert.True(t, DesensitizationStandard.IsValid())
	assert.True(t, DesensitizationStrict.IsValid())
	assert.True(t, DesensitizationOff.IsValid())
	assert.False(t, DesensitizationLevel("unknown").IsValid())
	assert.False(t, DesensitizationLevel("").IsValid())
}

func TestDesensitizationLevel_Normalize(t *testing.T) {
	t.Parallel()
	assert.Equal(t, DesensitizationStandard, DesensitizationStandard.Normalize())
	assert.Equal(t, DesensitizationStrict, DesensitizationStrict.Normalize())
	assert.Equal(t, DesensitizationOff, DesensitizationOff.Normalize())
	assert.Equal(t, DesensitizationStandard, DesensitizationLevel("").Normalize())
	assert.Equal(t, DesensitizationStandard, DesensitizationLevel("UNKNOWN").Normalize())
	assert.Equal(t, DesensitizationStrict, DesensitizationLevel("Strict").Normalize())
	assert.Equal(t, DesensitizationOff, DesensitizationLevel("OFF").Normalize())
}

func TestDesensitizationLevel_Constants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, DesensitizationLevel("standard"), DesensitizationStandard)
	assert.Equal(t, DesensitizationLevel("strict"), DesensitizationStrict)
	assert.Equal(t, DesensitizationLevel("off"), DesensitizationOff)
}

func TestCanonicalizeDesensitizationLevel_Accept(t *testing.T) {
	t.Parallel()
	cases := map[string]DesensitizationLevel{
		"standard":  DesensitizationStandard,
		"strict":    DesensitizationStrict,
		"off":       DesensitizationOff,
		"OFF":       DesensitizationOff,
		"Strict":    DesensitizationStrict,
		" standard": DesensitizationStandard,
		"OFF ":      DesensitizationOff,
		"  Off  ":   DesensitizationOff,
	}
	for input, want := range cases {
		got, ok := CanonicalizeDesensitizationLevel(input)
		assert.True(t, ok, "input %q should be accepted", input)
		assert.Equal(t, want, got, "input %q", input)
	}
}

func TestCanonicalizeDesensitizationLevel_Reject(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"", "xyz", "standardx", "of", "none", "loose"} {
		got, ok := CanonicalizeDesensitizationLevel(input)
		assert.False(t, ok, "input %q should be rejected", input)
		assert.Equal(t, DesensitizationLevel(""), got, "rejected input must not fall back to a valid level")
	}
}
