package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hzhan516/medmemo/internal/adapters/ai"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// OllamaDetectResult Ollama 环境检测结果，供前端展示。
type OllamaDetectResult struct {
	Installed      bool   `json:"installed"`                 // ollama 命令是否存在于 PATH
	Running        bool   `json:"running"`                   // 11434 端口是否响应
	HasSmolLM2     bool   `json:"has_smollm2"`               // smollm2:135m 模型是否已下载
	InstallGuide   string `json:"install_guide,omitempty"`   // 未安装时返回的安装引导
	ServerStarting bool   `json:"server_starting,omitempty"` // 正在后台启动服务
	PullProgress   string `json:"pull_progress,omitempty"`   // 模型下载进度文本
}

// DetectOllama 检测本地 Ollama 环境状态，返回即时结果（不触发后台操作）。
func (a *WailsApp) DetectOllama() (*OllamaDetectResult, error) {
	detector := ai.NewOllamaDetector()
	d := detector.Detect()

	return &OllamaDetectResult{
		Installed:    d.Installed,
		Running:      d.Running,
		HasSmolLM2:   d.HasSmolLM2,
		InstallGuide: d.InstallGuide,
	}, nil
}

// StartOllamaServer 在后台启动 ollama serve，通过 Wails Events 推送状态变更。
// 事件名称：ollama:server_starting, ollama:server_ready, ollama:server_error
func (a *WailsApp) StartOllamaServer() error {
	a.ollamaMu.Lock()
	defer a.ollamaMu.Unlock()

	detector := ai.NewOllamaDetector()

	if detector.IsRunning() {
		runtime.EventsEmit(a.ctx, "ollama:server_ready", map[string]any{
			"already_running": true,
		})
		return nil
	}

	if !detector.IsInstalled() {
		return fmt.Errorf("ollama is not installed")
	}

	go func() {
		detector := ai.NewOllamaDetector()

		_, err := detector.StartServer()
		if err != nil {
			runtime.EventsEmit(a.ctx, "ollama:server_error", map[string]string{
				"error": fmt.Errorf("failed to start ollama server: %w", err).Error(),
			})
			return
		}

		runtime.EventsEmit(a.ctx, "ollama:server_starting", map[string]string{})

		if err := detector.WaitForServer(30 * time.Second); err != nil {
			runtime.EventsEmit(a.ctx, "ollama:server_error", map[string]string{
				"error": fmt.Errorf("ollama server failed to become ready: %w", err).Error(),
			})
			return
		}

		runtime.EventsEmit(a.ctx, "ollama:server_ready", map[string]any{
			"already_running": false,
		})
	}()

	return nil
}

// PullOllamaModel 在后台执行 ollama pull 下载指定模型，通过 Wails Events 推送进度。
// 事件名称：ollama:pull_progress（每行进度）, ollama:pull_done, ollama:pull_error
func (a *WailsApp) PullOllamaModel(modelName string) error {
	if modelName == "" {
		modelName = ai.DefaultModelName
	}

	a.ollamaMu.Lock()
	defer a.ollamaMu.Unlock()

	detector := ai.NewOllamaDetector()
	if !detector.IsInstalled() {
		return fmt.Errorf("ollama is not installed")
	}

	go func(name string) {
		detector := ai.NewOllamaDetector()
		ctx, cancel := context.WithTimeout(a.ctx, 30*time.Minute)
		defer cancel()

		err := detector.PullModel(ctx, name, func(progress string) {
			runtime.EventsEmit(a.ctx, "ollama:pull_progress", map[string]string{
				"model":    name,
				"progress": progress,
			})
		})

		if err != nil {
			runtime.EventsEmit(a.ctx, "ollama:pull_error", map[string]string{
				"model": name,
				"error": fmt.Errorf("failed to pull model %s: %w", name, err).Error(),
			})
			return
		}

		runtime.EventsEmit(a.ctx, "ollama:pull_done", map[string]string{
			"model": name,
		})
	}(modelName)

	return nil
}

// EnsureOllamaAndSmolLM2 一键检测并确保 Ollama + SmolLM2 就绪。
// 返回当前检测状态；若需要后台操作（启动服务/下载模型），通过 Events 推送进度。
func (a *WailsApp) EnsureOllamaAndSmolLM2() (*OllamaDetectResult, error) {
	detector := ai.NewOllamaDetector()
	d := detector.Detect()

	result := &OllamaDetectResult{
		Installed:    d.Installed,
		Running:      d.Running,
		HasSmolLM2:   d.HasSmolLM2,
		InstallGuide: d.InstallGuide,
	}

	// 未安装：仅返回引导，不触发后台
	if !result.Installed {
		return result, nil
	}

	// 已安装未运行：后台启动
	if !result.Running {
		result.ServerStarting = true
		_ = a.StartOllamaServer()
		return result, nil
	}

	// 已运行但无模型：后台下载
	if !result.HasSmolLM2 {
		_ = a.PullOllamaModel(ai.DefaultModelName)
	}

	return result, nil
}

// CreateOllamaProvider 创建并保存 Ollama Provider 配置到数据库。
func (a *WailsApp) CreateOllamaProvider() (*models.ProviderConfig, error) {
	detector := ai.NewOllamaDetector()
	cfg := detector.BuildProviderConfig()

	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	if err := a.providerStore.Create(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to create ollama provider: %w", err)
	}

	return cfg, nil
}
