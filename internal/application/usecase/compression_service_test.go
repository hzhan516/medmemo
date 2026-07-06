package usecase

import (
	"testing"

	"github.com/hzhan516/medmemo/pkg/models"
)

func TestApplyDropEarliestN(t *testing.T) {
	history := []models.Message{
		{Role: models.RoleUser, Content: "1"},
		{Role: models.RoleAssistant, Content: "2"},
		{Role: models.RoleUser, Content: "3"},
		{Role: models.RoleAssistant, Content: "4"},
		{Role: models.RoleUser, Content: "5"},
	}

	t.Run("clamps N and preserves recent tail", func(t *testing.T) {
		got := applyDropEarliestN(history, 10, 2)
		if len(got) != 2 {
			t.Fatalf("len(got) = %d, want 2", len(got))
		}
		want := []string{"4", "5"}
		for i, w := range want {
			if got[i].Content != w {
				t.Errorf("got[%d].Content = %q, want %q", i, got[i].Content, w)
			}
		}
	})

	t.Run("drops exactly N when possible", func(t *testing.T) {
		got := applyDropEarliestN(history, 2, 2)
		if len(got) != 3 {
			t.Fatalf("len(got) = %d, want 3", len(got))
		}
		want := []string{"3", "4", "5"}
		for i, w := range want {
			if got[i].Content != w {
				t.Errorf("got[%d].Content = %q, want %q", i, got[i].Content, w)
			}
		}
	})

	t.Run("negative N with positive recentCount drops all but recent", func(t *testing.T) {
		got := applyDropEarliestN(history, -1, 3)
		if len(got) != 3 {
			t.Fatalf("len(got) = %d, want 3", len(got))
		}
		want := []string{"3", "4", "5"}
		for i, w := range want {
			if got[i].Content != w {
				t.Errorf("got[%d].Content = %q, want %q", i, got[i].Content, w)
			}
		}
	})
}

func TestApplySummarizeAndReplace(t *testing.T) {
	history := []models.Message{
		{Role: models.RoleUser, Content: "anchor1"},
		{Role: models.RoleAssistant, Content: "middle1"},
		{Role: models.RoleUser, Content: "middle2"},
		{Role: models.RoleAssistant, Content: "recent1"},
		{Role: models.RoleUser, Content: "recent2"},
	}

	summary := "summary"

	got := applySummarizeAndReplace(history, 1, 2, func(msgs []models.Message) string {
		if len(msgs) != 2 {
			t.Errorf("summarize called with %d messages, want 2", len(msgs))
		}
		return summary
	})

	if len(got) != 4 {
		t.Fatalf("len(got) = %d, want 4", len(got))
	}
	if got[0].Content != "anchor1" {
		t.Errorf("got[0].Content = %q, want anchor1", got[0].Content)
	}
	if got[1].Role != models.RoleSystem || got[1].Content != summary {
		t.Errorf("got[1] = {role=%s, content=%q}, want system summary", got[1].Role, got[1].Content)
	}
	wantRecent := []string{"recent1", "recent2"}
	for i, w := range wantRecent {
		if got[2+i].Content != w {
			t.Errorf("got[%d].Content = %q, want %q", 2+i, got[2+i].Content, w)
		}
	}
}

func TestBoundedRatioStillWorks(t *testing.T) {
	cases := []struct {
		used, max int
		want      float64
	}{
		{0, 100, 0},
		{50, 100, 0.5},
		{100, 100, 1},
		{200, 100, 1},
		{10, 0, 0},
	}

	for _, tt := range cases {
		got := boundedRatio(tt.used, tt.max)
		if got != tt.want {
			t.Errorf("boundedRatio(%d, %d) = %v, want %v", tt.used, tt.max, got, tt.want)
		}
	}
}
