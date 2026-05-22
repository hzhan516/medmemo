// Package main 是 MedMemo 应用入口。
// 仅负责依赖组装与生命周期管理，不包含业务逻辑。
package main

/*
#cgo LDFLAGS: -ldl
#define _GNU_SOURCE
#include <dlfcn.h>
#include <signal.h>

typedef int (*real_sigaction_t)(int, const struct sigaction *, struct sigaction *);

// 拦截所有 sigaction 调用，强制追加 SA_ONSTACK 标志。
// WebKit2GTK 安装的信号处理器缺少 SA_ONSTACK，导致 Go 1.21+ 运行时 fatal error。
// 此拦截器确保所有信号处理器（包括 WebKit 的）都使用 Go 运行时设置的信号栈。
int sigaction(int signum, const struct sigaction *act, struct sigaction *oldact) {
	static real_sigaction_t real_sigaction = NULL;
	if (!real_sigaction) {
		real_sigaction = (real_sigaction_t)dlsym(RTLD_NEXT, "sigaction");
	}
	if (act) {
		struct sigaction newact = *act;
		newact.sa_flags |= SA_ONSTACK;
		return real_sigaction(signum, &newact, oldact);
	}
	return real_sigaction(signum, act, oldact);
}
*/
import "C"

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/medmemo/medmemo/internal/application/pipeline"
	"github.com/medmemo/medmemo/internal/application/port"
	"github.com/medmemo/medmemo/internal/domain/entity"
	"github.com/medmemo/medmemo/internal/infrastructure/database"
	"github.com/medmemo/medmemo/internal/infrastructure/onnx"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:web/dist
var assets embed.FS

//go:embed wails.json
var wailsConfig embed.FS

// version 由构建时通过 -ldflags -X main.version={{.Version}} 注入。
var version = "dev"

func init() {
	if version != "dev" {
		return
	}
	data, err := wailsConfig.ReadFile("wails.json")
	if err != nil {
		return
	}
	var cfg struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return
	}
	v := strings.TrimSpace(cfg.Info.ProductVersion)
	if v != "" {
		version = "v" + v
	}
}

// buildTime 由构建时通过 -ldflags -X main.buildTime={{.Date}} 注入。
var buildTime = ""

func main() {
	// 将主 goroutine 绑定到当前 OS 线程，确保 GTK/WebKit 的初始化、
	// 事件循环和所有 C 回调都在同一个线程上执行，避免 Go 的 goroutine
	// 调度器在 CGO 调用期间切换线程导致的信号栈不匹配。
	runtime.LockOSThread()

	// 监听优雅关闭信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sigCh
		fmt.Println("\nreceived shutdown signal, gracefully stopping...")
		cancel()
	}()

	_ = ctx // 供后续 graceful shutdown 使用，当前由 cleanup 回调处理资源释放

	// 初始化应用（通过 Wire 生成的 InitializeApp）
	app, cleanup, err := InitializeApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize app: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	fmt.Printf("MedMemo %s starting...\n", version)

	// 启动 Wails 桌面应用
	err = wails.Run(&options.App{
		Title:     "MedMemo",
		Width:     1280,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup:        app.wailsApp.Startup,
		Bind: []interface{}{
			app.wailsApp,
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "app run error: %v\n", err)
		return
	}
}

// App 应用顶层封装，供 Wire 注入后返回。
type App struct {
	wailsApp *WailsApp
}

// NewEngineConfig 从 AppConfig 构造 ONNX Engine 配置，供 Wire 注入使用。
func NewEngineConfig(cfg *entity.AppConfig) onnx.EngineConfig {
	return onnx.EngineConfig{
		ResourceDir: "resources",
		ModelPath:   cfg.ModelDir,
	}
}

// NewApp 构造函数，供 Wire 调用。
// 启动时执行数据库迁移，返回 cleanup 回调用于关闭连接池。
// pipeline 参数当前仅作为 Wire 依赖 consumer，由 TASK-024 端云协同时正式启用。
func NewApp(wa *WailsApp, sqlite *database.SQLCipherConnector, _ *pipeline.DeidentifyPipeline, healthChecker port.HealthChecker) (*App, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := sqlite.Migrate(ctx); err != nil {
		return nil, nil, fmt.Errorf("database migration failed: %w", err)
	}

	cleanup := func() {
		wa.shutdownCallbackServers()
		if wa.deviceFlowSvc != nil {
			wa.deviceFlowSvc.Shutdown()
		}
		if wa.tokenRefreshSvc != nil {
			wa.tokenRefreshSvc.Shutdown()
		}
		if healthChecker != nil {
			healthChecker.Stop()
		}
		if err := sqlite.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to close database: %v\n", err)
		}
	}
	return &App{wailsApp: wa}, cleanup, nil
}
