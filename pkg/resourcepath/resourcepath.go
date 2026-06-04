package resourcepath

import (
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

func candidateDirs() []string {
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
