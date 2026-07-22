package resourcepath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const EnvDir = "MEDMEMO_RESOURCE_DIR"

func Dir() string {
	if v := strings.TrimSpace(os.Getenv(EnvDir)); v != "" {
		return cleanAbs(v)
	}

	for _, candidate := range candidateDirs() {
		if isDir(candidate) {
			return cleanAbs(candidate)
		}
	}

	return "resources"
}

func Path(elem ...string) string {
	parts := append([]string{Dir()}, elem...)
	return filepath.Join(parts...)
}

func Resolve(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}

	clean := filepath.Clean(path)
	if clean == "resources" {
		return Dir()
	}

	prefix := "resources" + string(os.PathSeparator)
	if strings.HasPrefix(clean, prefix) {
		return filepath.Join(Dir(), strings.TrimPrefix(clean, prefix))
	}

	return path
}

// ResolveSafe 解析相对于资源根目录的路径，返回错误如果解析后的路径超出资源根目录。
// 用于防止路径遍历攻击（path traversal）。
// Audit: RR-003 path traversal guard
func ResolveSafe(subpath string) (string, error) {
	if subpath == "" {
		return "", fmt.Errorf("subpath is empty")
	}

	root := Dir()
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("failed to resolve resource root: %w", err)
	}
	rootAbs = filepath.Clean(rootAbs)

	// 拒绝绝对路径（必须由调用方显式处理）
	if filepath.IsAbs(subpath) {
		return "", fmt.Errorf("subpath must be relative: %s", subpath)
	}

	// 拒绝包含 .. 的路径（纵深防御，即使 Clean 后也应检查）
	if strings.Contains(subpath, "..") {
		return "", fmt.Errorf("subpath contains directory traversal: %s", subpath)
	}

	// 先按原有 Resolve 逻辑处理 "resources/" 前缀
	resolved := Resolve(subpath)
	if resolved == subpath {
		// Resolve 未处理，说明不是 resources/ 前缀，直接拼接
		resolved = filepath.Join(rootAbs, subpath)
	}

	resolvedAbs, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("failed to resolve subpath: %w", err)
	}
	resolvedAbs = filepath.Clean(resolvedAbs)

	// 校验最终路径是否在资源根目录下
	sep := string(filepath.Separator)
	if !strings.HasPrefix(resolvedAbs, rootAbs+sep) && resolvedAbs != rootAbs {
		return "", fmt.Errorf("resolved path escapes resource root: %s", resolvedAbs)
	}

	return resolvedAbs, nil
}

// candidateDirs 为包级变量，便于测试替换为与环境无关的候选列表。
var candidateDirs = func() []string {
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "..", "Resources", "resources"),
			filepath.Join(exeDir, "resources"),
			filepath.Join(exeDir, "..", "share", "resources"),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "resources"))
	}
	// DEB/RPM 安装路径回退（当 wrapper 未设置 MEDMEMO_RESOURCE_DIR 时）
	candidates = append(candidates, "/opt/medmemo/resources")
	candidates = append(candidates, "resources")
	return candidates
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func cleanAbs(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
}
