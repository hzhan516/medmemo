package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"

	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/hzhan516/medmemo/pkg/resourcepath"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// EmbeddingStatusResponse Embedding 模型状态响应。
type EmbeddingStatusResponse struct {
	Available         bool   `json:"available"`           // 语义搜索是否真实可用
	ModelPresent      bool   `json:"model_present"`       // 模型文件是否存在
	EngineAvailable   bool   `json:"engine_available"`    // ONNX embedding 引擎是否可用
	RuntimeLibPresent bool   `json:"runtime_lib_present"` // ONNX Runtime 动态库是否存在
	RuntimeLibPath    string `json:"runtime_lib_path"`    // ONNX Runtime 动态库路径
	FailureReason     string `json:"failure_reason"`      // 初始化失败原因
	ModelPath         string `json:"model_path"`          // 模型存放路径
	ModelName         string `json:"model_name"`          // 模型名称
	DownloadURL       string `json:"download_url"`        // 模型下载页面 URL
}

type embeddingFailureReasonReporter interface {
	FailureReason() string
}

type embeddingRuntimeLibPathReporter interface {
	RuntimeLibPath() string
}

// embeddingModelDir 返回 Embedding 模型目录的绝对路径。
// 始终使用用户数据目录 ~/.medmemo/data/models/all-MiniLM-L6-v2，
// 确保在 AppImage（只读 FS）、macOS .app bundle 及 Windows 安装目录中均可正常读写。
func (a *WailsApp) embeddingModelDir() string {
	return filepath.Join(a.config.DataDir, "models", models.EmbeddingModelName)
}

// GetEmbeddingStatus 获取本地 Embedding 模型状态。
func (a *WailsApp) GetEmbeddingStatus() (*EmbeddingStatusResponse, error) {
	if a.config == nil {
		return nil, fmt.Errorf("app config not initialized")
	}
	modelPath := a.embeddingModelDir()
	modelFile := filepath.Join(modelPath, "model.onnx")
	tokenizerFile := filepath.Join(modelPath, "tokenizer.json")

	modelPresent := fileExists(modelFile)
	tokenizerPresent := fileExists(tokenizerFile)

	runtimeLibPath := ""
	runtimeLibPresent := false
	if reporter, ok := a.embeddingSvc.(embeddingRuntimeLibPathReporter); ok {
		runtimeLibPath = reporter.RuntimeLibPath()
	}
	if runtimeLibPath == "" {
		runtimeLibPath = defaultONNXRuntimeLibPath(resourcepath.Dir())
	}
	if runtimeLibPath != "" {
		runtimeLibPresent = fileExists(runtimeLibPath)
	}

	engineAvailable := false
	if a.embeddingSvc != nil {
		engineAvailable = a.embeddingSvc.IsAvailable()
	}

	failureReason := ""
	if reporter, ok := a.embeddingSvc.(embeddingFailureReasonReporter); ok {
		failureReason = reporter.FailureReason()
	}
	switch {
	case !modelPresent:
		failureReason = "embedding model file is missing"
	case !tokenizerPresent && !engineAvailable:
		failureReason = fmt.Sprintf("embedding tokenizer file is missing: %s", tokenizerFile)
	case !runtimeLibPresent:
		if runtimeLibPath == "" {
			failureReason = "ONNX Runtime library path could not be resolved"
		} else {
			failureReason = fmt.Sprintf("ONNX Runtime library not found: %s", runtimeLibPath)
		}
	case !engineAvailable && failureReason == "":
		failureReason = "embedding engine not available"
	}

	downloadURL := a.config.EmbeddingModelDownloadURL
	if downloadURL == "" {
		// 默认指向 GitHub Release 下载页面，用户可在 config.yaml 中自定义
		downloadURL = "https://github.com/hzhan516/medmemo/releases/tag/embedding-model-v1"
	}

	// 向前端返回绝对路径，便于展示
	absPath, _ := filepath.Abs(modelPath)
	if absPath != "" {
		modelPath = absPath
	}

	return &EmbeddingStatusResponse{
		Available:         modelPresent && engineAvailable,
		ModelPresent:      modelPresent,
		EngineAvailable:   engineAvailable,
		RuntimeLibPresent: runtimeLibPresent,
		RuntimeLibPath:    runtimeLibPath,
		FailureReason:     failureReason,
		ModelPath:         modelPath,
		ModelName:         models.EmbeddingModelName,
		DownloadURL:       downloadURL,
	}, nil
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func defaultONNXRuntimeLibPath(resourceDir string) string {
	switch goruntime.GOOS {
	case "linux":
		primary := filepath.Join(resourceDir, "lib", "linux", "libonnxruntime.so")
		if fileExists(primary) {
			return primary
		}
		return filepath.Join(resourceDir, "lib", "linux", "libonnxruntime.so.1")
	case "darwin":
		return filepath.Join(resourceDir, "lib", "darwin", "libonnxruntime.dylib")
	case "windows":
		return filepath.Join(resourceDir, "lib", "windows", "onnxruntime.dll")
	default:
		return ""
	}
}

// GetEmbeddingModelDirPath 返回 Embedding 模型目录的绝对路径。
func (a *WailsApp) GetEmbeddingModelDirPath() (string, error) {
	modelPath := a.embeddingModelDir()
	absPath, err := filepath.Abs(modelPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve model dir path: %w", err)
	}
	// 确保目录存在（使用用户可写路径，不会在只读 FS 上失败）
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create model dir: %w", err)
	}
	return absPath, nil
}

// OpenEmbeddingModelDir 打开 Embedding 模型所在目录。
// 使用平台特定命令打开文件管理器，比 BrowserOpenURL 更可靠。
func (a *WailsApp) OpenEmbeddingModelDir() error {
	absPath, err := a.GetEmbeddingModelDirPath()
	if err != nil {
		return fmt.Errorf("failed to get embedding model directory: %w", err)
	}

	var cmd string
	var args []string
	switch goruntime.GOOS {
	case "windows":
		cmd = "explorer.exe"
		args = []string{absPath}
	case "darwin":
		cmd = "open"
		args = []string{absPath}
	default: // linux and others
		cmd = "xdg-open"
		args = []string{absPath}
	}

	c := exec.Command(cmd, args...)
	if err := c.Start(); err != nil {
		// 命令启动失败时，弹窗提示用户手动前往
		_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.InfoDialog,
			Title:   "打开模型目录",
			Message: fmt.Sprintf("无法自动打开文件管理器，请手动前往以下目录：\n\n%s", absPath),
		})
		return fmt.Errorf("failed to open model dir with %s: %w", cmd, err)
	}
	return nil
}
