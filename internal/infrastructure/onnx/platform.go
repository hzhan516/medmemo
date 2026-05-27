// Package onnx 封装 Hugot ONNX 推理运行时。
// ONNX Session 非线程安全，本包通过 Worker 模型确保串行化调用。
package onnx

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// PlatformLibPath 返回当前平台的 ONNX Runtime 动态库绝对路径。
// 库文件按平台存放于 resources/lib/<GOOS>/ 目录下：
//   - linux:   libonnxruntime.so
//   - darwin:  libonnxruntime.dylib
//   - windows: onnxruntime.dll
// fallback: 若 .so 不存在则尝试 .so.1（Linux 版本化库常见情况）。
func PlatformLibPath(resourceDir string) (string, error) {
	var libName, fallbackName string
	switch runtime.GOOS {
	case "linux":
		libName = "libonnxruntime.so"
		fallbackName = "libonnxruntime.so.1"
	case "darwin":
		libName = "libonnxruntime.dylib"
	case "windows":
		libName = "onnxruntime.dll"
	default:
		return "", fmt.Errorf("unsupported platform %s for ONNX Runtime", runtime.GOOS)
	}
	primary := filepath.Join(resourceDir, "lib", runtime.GOOS, libName)
	if fallbackName != "" {
		fallback := filepath.Join(resourceDir, "lib", runtime.GOOS, fallbackName)
		if _, err := os.Stat(primary); os.IsNotExist(err) {
			if _, err2 := os.Stat(fallback); err2 == nil {
				return fallback, nil
			}
		}
	}
	return primary, nil
}

// DefaultModelPath 返回默认 NER 模型目录路径。
func DefaultModelPath(resourceDir string) string {
	return filepath.Join(resourceDir, "models", "distilbert-ner")
}
