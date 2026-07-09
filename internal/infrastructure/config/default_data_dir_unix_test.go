//go:build !windows

package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDefaultDataDirPath_HomeError 验证无法获取用户主目录时回退到相对路径。
func TestDefaultDataDirPath_HomeError(t *testing.T) {
	t.Setenv("HOME", "")
	// 当 HOME 为空且系统无法解析时，应返回相对路径兜底
	got := defaultDataDirPath()
	if os.Getenv("HOME") == "" {
		assert.Equal(t, ".medmemo/data", got)
	}
}
