package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/application/updater"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Startup 是 Wails 启动回调，在前端加载完成后调用。
func (a *WailsApp) Startup(ctx context.Context) {
	a.ctx = ctx

	// 校验前端嵌入资源是否完整（编译时 embed，运行时读取）
	if err := validateEmbeddedAssets(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load frontend assets: %v\n", err)
		os.Exit(1)
	}

	// 初始化 token 刷新降级回调
	if a.tokenRefreshSvc != nil {
		a.tokenRefreshSvc.SetOnDegraded(func(providerID, reason string) {
			runtime.EventsEmit(a.ctx, "auth:degraded", map[string]string{
				"provider_id": providerID,
				"reason":      reason,
			})
		})
	}

	// 启动健康检测引擎
	if a.healthChecker != nil {
		a.healthChecker.SetOnChange(func(result port.HealthResult) {
			runtime.EventsEmit(a.ctx, "provider:health_changed", result)
		})
		a.healthChecker.Start(a.ctx)
	}

	// 启动时异步检测更新（不阻塞首屏）
	if a.config.UpdateCheckEnabled && a.updaterSvc != nil {
		// 将配置文件中的更新通道同步到 updater 服务
		a.updaterSvc.SetSettings(&entity.UpdateSettings{
			CheckEnabled: a.config.UpdateCheckEnabled,
			Channel:      a.config.UpdateChannel,
			SkipVersion:  "",
		})
		go a.checkUpdateAsync()
	}

	// 执行数据留存自动清理（后台 goroutine，不阻塞启动）
	go a.runRetentionCleanup()

	// 启动时扫描 cli_token / oauth_device provider，安排自动刷新
	if a.tokenRefreshSvc != nil {
		go a.scheduleAutoRefreshesAsync()
	}

	// ONNX 预热：启动后异步执行一次 dummy 推理，将 warmup 成本从首次对话转移到启动阶段
	go a.warmupONNX()

	// v1.1.4: 历史 embedding 迁移（版本升级后首次启动触发）
	go a.runEmbeddingMigration()

	// 初始化 Device Flow 事件回调
	if a.deviceFlowSvc != nil {
		a.deviceFlowSvc.SetRefreshService(a.tokenRefreshSvc)
		a.deviceFlowSvc.SetCallbacks(
			func(deviceCode, providerType string, cfg *models.ProviderConfig) {
				runtime.EventsEmit(a.ctx, "oauth:success", map[string]any{
					"device_code":   deviceCode,
					"provider_type": providerType,
					"provider_id":   cfg.ID,
					"provider_name": cfg.Name,
				})
			},
			func(deviceCode, providerType string, err error) {
				runtime.EventsEmit(a.ctx, "oauth:error", map[string]any{
					"device_code":   deviceCode,
					"provider_type": providerType,
					"error":         err.Error(),
				})
			},
			func(deviceCode, providerType string) {
				runtime.EventsEmit(a.ctx, "oauth:pending", map[string]any{
					"device_code":   deviceCode,
					"provider_type": providerType,
				})
			},
			func(deviceCode, providerType string, newInterval int) {
				runtime.EventsEmit(a.ctx, "oauth:slow_down", map[string]any{
					"device_code":   deviceCode,
					"provider_type": providerType,
					"new_interval":  newInterval,
				})
			},
		)
	}
}

// scheduleAutoRefreshesAsync 延迟 3 秒后扫描并安排自动刷新，避免与启动流程竞争。
func (a *WailsApp) scheduleAutoRefreshesAsync() {
	time.Sleep(3 * time.Second)
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	list, err := a.providerStore.List(ctx)
	if err != nil {
		return
	}

	for _, p := range list {
		if !p.Enabled {
			continue
		}
		if p.AuthMethod != models.AuthMethodCLIToken && p.AuthMethod != models.AuthMethodOAuthDevice {
			continue
		}
		creds, err := models.ReadCLICredentials(p.AuthParams.CLICredentialPath)
		if err != nil {
			continue
		}
		// 只有含 refresh_token 的 provider 才需要自动刷新
		if creds.RefreshToken != "" || p.AuthParams.OAuthRefreshToken != "" {
			_ = a.tokenRefreshSvc.ScheduleAutoRefresh(p.ID)
		}
	}
}

// checkUpdateAsync 延迟 5 秒后异步检测更新，避免与启动流程竞争资源。
func (a *WailsApp) checkUpdateAsync() {
	time.Sleep(5 * time.Second)
	if !a.updaterSvc.GetSettings().ShouldCheck(updater.CheckInterval) {
		return
	}

	info, err := a.updaterSvc.CheckUpdate(a.ctx, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to check update: %v\n", err)
		return
	}
	if info == nil {
		return
	}

	// 通过 Wails Events 推送更新通知到前端
	payload := map[string]any{
		"version":      info.Version,
		"name":         info.Name,
		"body":         info.Body,
		"published_at": info.PublishedAt.Format(time.RFC3339),
		"mandatory":    info.Mandatory,
		"channel":      string(info.Channel),
	}
	runtime.EventsEmit(a.ctx, "update:available", payload)
}

// warmupONNX 执行一次 dummy 推理，触发 ONNX Runtime 的首次模型 warmup。
// 使用应用生命周期 context，不设置短 timeout，让 warmup 自然完成。
// 失败仅记录日志，不影响应用启动。
func (a *WailsApp) warmupONNX() {
	defer a.onnxOnce.Do(func() { close(a.onnxReady) })

	// 延迟 2 秒，确保 ONNX engine 已完全初始化
	time.Sleep(2 * time.Second)

	start := time.Now()
	if a.embeddingSvc != nil {
		_, err := a.embeddingSvc.EmbedSingle(a.ctx, "warmup")
		if err != nil {
			fmt.Printf("[ONNX Warmup] embedding 预热失败: %v\n", err)
		} else {
			fmt.Printf("[ONNX Warmup] embedding 预热完成，耗时 %v\n", time.Since(start))
		}
	}
}
func (a *WailsApp) waitForONNXReady() {
	<-a.onnxReady
}

// safeEventsEmit 安全地发射 Wails 事件，在测试环境（标准 context.Background）下静默跳过。
func (a *WailsApp) safeEventsEmit(eventName string, data ...any) {
	if a.ctx == nil || a.ctx == context.Background() || a.ctx == context.TODO() {
		return
	}
	runtime.EventsEmit(a.ctx, eventName, data...)
}

// runEmbeddingMigration 在 ONNX warmup 后执行 embedding 版本迁移。
func (a *WailsApp) runEmbeddingMigration() {
	a.waitForONNXReady()

	if a.migrator == nil || !a.embeddingSvc.IsAvailable() {
		if a.migrationState != nil {
			a.migrationState.SetComplete(true)
		}
		return
	}

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Minute)
	defer cancel()

	needs, total, err := a.migrator.NeedsMigration(ctx)
	if err != nil || !needs {
		a.migrationState.SetComplete(true)
		return
	}

	a.safeEventsEmit("embedding:migration:start", map[string]any{
		"total": total,
	})

	processed, failed, err := a.migrator.RunMigration(ctx, func(p, t int) {
		a.safeEventsEmit("embedding:migration:progress", map[string]any{
			"processed": p,
			"total":     t,
		})
	})

	a.safeEventsEmit("embedding:migration:done", map[string]any{
		"processed": processed,
		"failed":    failed,
	})

	if err != nil {
		fmt.Printf("[EmbeddingMigration] 迁移异常: %v\n", err)
	}
	if failed > 0 {
		fmt.Printf("[EmbeddingMigration] %d 条 fact 迁移失败（已记录日志）\n", failed)
	}
	fmt.Printf("[EmbeddingMigration] 迁移完成：处理 %d 条，失败 %d 条\n", processed, failed)
}

// validateEmbeddedAssets 校验嵌入的前端资源是否完整可用。
// 放在 Startup 中执行而非 main() 开头，避免 Wails binding 生成阶段触发误报。
func validateEmbeddedAssets() error {
	data, err := assets.ReadFile("web/dist/index.html")
	if err != nil {
		return fmt.Errorf("embedded web/dist/index.html missing: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return fmt.Errorf("embedded web/dist/index.html is empty")
	}
	return nil
}
