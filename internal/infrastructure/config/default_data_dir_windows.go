//go:build windows

package config

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// defaultDataDirPath 返回 Windows 平台的默认数据目录。
// 优先使用注册表中记录的安装目录下的 data 子目录；若未安装或未写入注册表，
// 则回退到 ~/.medmemo/data，保证便携模式与开发调试仍可运行。
func defaultDataDirPath() string {
	if dir := installDirFromRegistry(); dir != "" {
		return filepath.Join(dir, "data")
	}
	return fallbackDataDir()
}

// installDirFromRegistry 从 HKCU\Software\MedMemo 读取 InstallPath 并归一化为目录。
func installDirFromRegistry() string {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\MedMemo`, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()

	installPath, _, err := k.GetStringValue("InstallPath")
	if err != nil {
		return ""
	}

	installPath = strings.TrimSpace(installPath)
	if strings.EqualFold(filepath.Base(installPath), "MedMemo.exe") {
		installPath = filepath.Dir(installPath)
	}
	if installPath == "" {
		return ""
	}
	return installPath
}

// fallbackDataDir 在无法获取安装目录时返回用户主目录下的默认数据路径。
func fallbackDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".medmemo/data"
	}
	return filepath.Join(home, ".medmemo", "data")
}
