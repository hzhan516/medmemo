package entity

import (
	"strings"
	"testing"
	"time"
)

func TestNewHealthMemory(t *testing.T) {
	mem := NewHealthMemory(TierShortTerm, "头痛", "conv_123")
	if mem.ID == "" {
		t.Error("expected non-empty ID")
	}
	if mem.Tier != TierShortTerm {
		t.Errorf("expected tier %v, got %v", TierShortTerm, mem.Tier)
	}
	if mem.Content != "头痛" {
		t.Errorf("expected content %q, got %q", "头痛", mem.Content)
	}
	if mem.SourceConv != "conv_123" {
		t.Errorf("expected source conv %q, got %q", "conv_123", mem.SourceConv)
	}
	if mem.Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", mem.Confidence)
	}
	if len(mem.Tags) != 0 {
		t.Errorf("expected empty tags, got %d", len(mem.Tags))
	}
	if mem.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if mem.AccessedAt.IsZero() {
		t.Error("expected AccessedAt to be set")
	}
	if !strings.HasPrefix(string(mem.ID), "mem_") {
		t.Errorf("expected ID to start with 'mem_', got %q", mem.ID)
	}
}

func TestHealthMemory_MarkAccessed(t *testing.T) {
	mem := NewHealthMemory(TierLongTerm, "感冒", "conv_456")
	before := mem.AccessedAt
	time.Sleep(10 * time.Millisecond)

	mem.MarkAccessed()

	if !mem.AccessedAt.After(before) {
		t.Error("expected AccessedAt to be updated")
	}
}
