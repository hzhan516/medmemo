// Package updater 封装跨平台更新安装器实现。
// 使用 go:build 标签隔离各平台逻辑，编译期仅包含当前平台代码。
package updater

import (
	"os"
	"path/filepath"
)

// getCurrentBinary 获取当前运行二进制文件的绝对路径。
func getCurrentBinary() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		return exe
	}
	return abs
}
