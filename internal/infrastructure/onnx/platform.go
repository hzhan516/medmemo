// Package onnx 封装 Hugot ONNX 推理运行时。
// ONNX Session 非线程安全，本包通过 Worker 模型确保串行化调用。
package onnx

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// PlatformLibPath 返回当前平台的 ONNX Runtime 动态库绝对路径。
// 库文件按平台存放于 resources/lib/<GOOS>/ 目录下：
//   - linux:   libonnxruntime.so
//   - darwin:  libonnxruntime.dylib
//   - windows: onnxruntime.dll
func PlatformLibPath(resourceDir string) (string, error) {
	var libName string
	switch runtime.GOOS {
	case "linux":
		libName = "libonnxruntime.so"
	case "darwin":
		libName = "libonnxruntime.dylib"
	case "windows":
		libName = "onnxruntime.dll"
	default:
		return "", fmt.Errorf("unsupported platform %s for ONNX Runtime", runtime.GOOS)
	}
	return filepath.Join(resourceDir, "lib", runtime.GOOS, libName), nil
}

// DefaultModelPath 返回默认 NER 模型目录路径。
func DefaultModelPath(resourceDir string) string {
	return filepath.Join(resourceDir, "models", "distilbert-ner")
}
