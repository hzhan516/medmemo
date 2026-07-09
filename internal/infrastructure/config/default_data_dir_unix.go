//go:build !windows

package config

import (
	"os"
	"path/filepath"
)

// defaultDataDirPath 返回非 Windows 平台的默认数据目录。
// 当前保持 ~/.medmemo/data 以兼容现有用户数据。
func defaultDataDirPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".medmemo/data"
	}
	return filepath.Join(home, ".medmemo", "data")
}
