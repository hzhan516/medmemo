package entity

import "testing"

func TestAllVersionNotesNonEmpty(t *testing.T) {
	if len(AllVersionNotes) == 0 {
		t.Fatal("AllVersionNotes should not be empty")
	}

	for i, note := range AllVersionNotes {
		if note.Version == "" {
			t.Errorf("note[%d].Version is empty", i)
		}
		if note.Title == "" {
			t.Errorf("note[%d].Title is empty", i)
		}
		if len(note.Features) == 0 && len(note.Fixes) == 0 {
			t.Errorf("note[%d] (%s) has no features or fixes", i, note.Version)
		}
	}
}

func TestAllVersionNotesParseable(t *testing.T) {
	for _, note := range AllVersionNotes {
		_, err := parseSemver(note.Version)
		if err != nil {
			t.Errorf("parseSemver(%q) failed: %v", note.Version, err)
		}
	}
}

func TestLoadVersionNotes(t *testing.T) {
	t.Run("valid json", func(t *testing.T) {
		var original = AllVersionNotes
		AllVersionNotes = nil
		defer func() { AllVersionNotes = original }()

		data := []byte(`[{"version":"v1.0","title":"Test","features":["f1"],"fixes":[]}]`)
		if err := loadVersionNotes(data); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(AllVersionNotes) != 1 {
			t.Errorf("expected 1 note, got %d", len(AllVersionNotes))
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		var original = AllVersionNotes
		AllVersionNotes = nil
		defer func() { AllVersionNotes = original }()

		data := []byte(`invalid json`)
		if err := loadVersionNotes(data); err == nil {
			t.Error("expected error for invalid json")
		}
	})
}

func TestAllVersionNotesAscending(t *testing.T) {
	for i := 1; i < len(AllVersionNotes); i++ {
		prevStr := AllVersionNotes[i-1].Version
		currStr := AllVersionNotes[i].Version

		cmp, err := CompareVersions(prevStr, currStr)
		if err != nil {
			t.Fatalf("CompareVersions(%q, %q) failed: %v", prevStr, currStr, err)
		}
		if cmp > 0 {
			continue
		}

		// 允许热修复 rc 紧跟同核心稳定版之后（如 v1.1.10 -> v1.1.10-rc.13）。
		// 按语义化版本 rc < stable，但 changelog 按发布时间排序时会出现此情况。
		prev, err := parseSemver(prevStr)
		if err != nil {
			t.Fatalf("parseSemver(%q) failed: %v", prevStr, err)
		}
		curr, err := parseSemver(currStr)
		if err != nil {
			t.Fatalf("parseSemver(%q) failed: %v", currStr, err)
		}
		if cmp < 0 && prev.core == curr.core && prev.label == "" && curr.label != "" {
			continue
		}

		t.Errorf("versions not ascending: %s -> %s", prevStr, currStr)
	}
}
