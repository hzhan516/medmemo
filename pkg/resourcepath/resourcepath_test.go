package resourcepath

import (
	"path/filepath"
	"testing"
)

func TestDirUsesEnvironmentOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDir, dir)

	if got := Dir(); got != filepath.Clean(dir) {
		t.Fatalf("Dir() = %q, want %q", got, filepath.Clean(dir))
	}
}

func TestResolveMapsResourcesPrefix(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDir, dir)

	got := Resolve(filepath.Join("resources", "rules", "compliance_rules_v1.json"))
	want := filepath.Join(dir, "rules", "compliance_rules_v1.json")
	if got != want {
		t.Fatalf("Resolve() = %q, want %q", got, want)
	}
}

func TestResolveLeavesAbsolutePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "model.onnx")
	if got := Resolve(path); got != path {
		t.Fatalf("Resolve() = %q, want %q", got, path)
	}
}
