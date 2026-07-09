package resourcepath

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestResolveSafe_Valid 验证 ResolveSafe 正常解析 resources 下的路径。
// Audit: RR-003 path traversal guard
func TestResolveSafe_Valid(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDir, dir)

	got, err := ResolveSafe(filepath.Join("resources", "rules", "compliance_rules_v1.json"))
	require.NoError(t, err)
	want := filepath.Join(dir, "rules", "compliance_rules_v1.json")
	if got != want {
		t.Fatalf("ResolveSafe() = %q, want %q", got, want)
	}
}

// TestResolveSafe_RejectTraversal 验证 ResolveSafe 拒绝目录穿越路径。
// Audit: RR-003 malicious path rejection
func TestResolveSafe_RejectTraversal(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDir, dir)

	cases := []struct {
		name    string
		subpath string
		wantErr string
	}{
		{
			name:    "dotdot_in_middle",
			subpath: "resources/rules/../../etc/passwd",
			wantErr: "directory traversal",
		},
		{
			name:    "dotdot_at_start",
			subpath: "../secret.txt",
			wantErr: "directory traversal",
		},
		{
			name:    "absolute_path",
			subpath: "/etc/passwd",
			wantErr: "must be relative",
		},
		{
			name:    "empty_path",
			subpath: "",
			wantErr: "empty",
		},
		{
			name:    "dotdot_only",
			subpath: "..",
			wantErr: "directory traversal",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ResolveSafe(tc.subpath)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

// TestResolveSafe_RejectEscape 验证即使经过 Clean 后路径仍逃出根目录的情况。
// Audit: RR-003 path traversal guard - post-clean escape check
func TestResolveSafe_RejectEscape(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDir, dir)

	// 构造一个路径：经过 Clean 后仍逃出根目录
	// 例如 "resources/../../secret.txt" — 先被 Resolve 处理为 dir + "../../secret.txt"
	// 然后 filepath.Join 和 Clean 后可能变成 dir 的父目录
	subpath := filepath.Join("resources", "..", "..", "secret.txt")
	_, err := ResolveSafe(subpath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "directory traversal")
}
