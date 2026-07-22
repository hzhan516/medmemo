package resourcepath

import (
	"os"
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

// TestResolve_ExactResources 验证 "resources" 精确匹配返回资源根目录。
func TestResolve_ExactResources(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDir, dir)
	assert.Equal(t, dir, Resolve("resources"))
}

// TestResolve_EmptyPath 验证空路径返回空字符串。
func TestResolve_EmptyPath(t *testing.T) {
	assert.Empty(t, Resolve(""))
}

// TestDir_FallbackToRelativeResources 验证无环境变量且候选目录均不存在时回退到相对 resources。
func TestDir_FallbackToRelativeResources(t *testing.T) {
	t.Setenv(EnvDir, "")
	// 桩掉候选目录探测，避免宿主机（如已安装 DEB/RPM）环境差异影响回退断言
	original := candidateDirs
	candidateDirs = func() []string {
		return []string{filepath.Join(t.TempDir(), "nonexistent")}
	}
	t.Cleanup(func() { candidateDirs = original })

	got := Dir()
	assert.Equal(t, "resources", got)
}

// TestDir_CandidateMatch 验证当 exe 同级目录存在 resources 时优先返回该目录。
func TestDir_CandidateMatch(t *testing.T) {
	t.Setenv(EnvDir, "")
	exe, err := os.Executable()
	require.NoError(t, err)
	exeDir := filepath.Dir(exe)
	resDir := filepath.Join(exeDir, "resources")
	require.NoError(t, os.MkdirAll(resDir, 0755))
	t.Cleanup(func() { _ = os.RemoveAll(resDir) })

	got := Dir()
	assert.Equal(t, resDir, got)
}

// TestPath 验证 Path 拼接资源根目录与给定路径。
func TestPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDir, dir)
	got := Path("models", "model.onnx")
	want := filepath.Join(dir, "models", "model.onnx")
	assert.Equal(t, want, got)
}

// TestIsDir 验证目录存在性判断。
func TestIsDir(t *testing.T) {
	tmpDir := t.TempDir()
	assert.True(t, isDir(tmpDir))
	assert.False(t, isDir(filepath.Join(tmpDir, "nonexistent")))

	file := filepath.Join(tmpDir, "file.txt")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0644))
	assert.False(t, isDir(file))
}

// TestCleanAbs 验证 cleanAbs 对相对路径的清理结果。
func TestCleanAbs(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	got := cleanAbs("resources")
	want := filepath.Join(wd, "resources")
	assert.Equal(t, want, got)
}
