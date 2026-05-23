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
		prev, err := parseSemver(AllVersionNotes[i-1].Version)
		if err != nil {
			t.Fatalf("parseSemver(%q) failed: %v", AllVersionNotes[i-1].Version, err)
		}
		curr, err := parseSemver(AllVersionNotes[i].Version)
		if err != nil {
			t.Fatalf("parseSemver(%q) failed: %v", AllVersionNotes[i].Version, err)
		}

		hasUpdate := false
		for j := 0; j < 3; j++ {
			if curr[j] > prev[j] {
				hasUpdate = true
				break
			}
			if curr[j] < prev[j] {
				break
			}
		}
		if !hasUpdate {
			t.Errorf("versions not ascending: %s -> %s", AllVersionNotes[i-1].Version, AllVersionNotes[i].Version)
		}
	}
}
