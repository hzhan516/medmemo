//go:build windows

package config

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// defaultDataDirPath 返回 Windows 平台的默认数据目录。
//
// 选择优先级（显式配置在 Loader 层处理，此处仅计算默认值）：
//  1. 若旧库 %USERPROFILE%\.medmemo\data\medmemo.db 存在，返回旧目录，保证已升级用户的历史数据可见。
//  2. 若注册表中存在安装目录，且其 data 子目录可写，返回 <installDir>\data。
//  3. 否则回退到 %USERPROFILE%\.medmemo\data，覆盖便携模式与 Program Files 等不可写安装场景。
func defaultDataDirPath() string {
	legacyDir := fallbackDataDir()
	installDir := installDirFromRegistry()
	return selectWindowsDataDir(installDir, legacyDir, osFileExists, osDirWritable)
}

// selectWindowsDataDir 是 Windows 默认数据目录的可测试选择函数。
// 显式配置（config.yaml / MEDMEMO_DATA_DIR）由调用方在加载阶段覆盖，不进入本函数。
func selectWindowsDataDir(installDir, legacyDir string, fileExists func(string) bool, dirWritable func(string) bool) string {
	legacyDB := filepath.Join(legacyDir, "medmemo.db")
	if fileExists(legacyDB) {
		return legacyDir
	}
	if installDir == "" {
		return legacyDir
	}
	installData := filepath.Join(installDir, "data")
	if dirWritable(installData) {
		return installData
	}
	return legacyDir
}

// osFileExists 使用 os.Stat 判断文件是否存在。
func osFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// osDirWritable 探测目录是否可写：必要时创建目录，写入并删除临时文件。
func osDirWritable(dir string) bool {
	if dir == "" {
		return false
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false
	}
	f, err := os.CreateTemp(dir, ".medmemo-write-test-*")
	if err != nil {
		return false
	}
	_ = f.Close()
	_ = os.Remove(f.Name())
	return true
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
