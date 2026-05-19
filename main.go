// Package main 是 MedMemo 应用入口。
// 仅负责依赖组装与生命周期管理，不包含业务逻辑。
package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/medmemo/medmemo/internal/infrastructure/database"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:web/dist
var assets embed.FS

func main() {
	// 监听优雅关闭信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nreceived shutdown signal, gracefully stopping...")
	}()

	// 初始化应用（通过 Wire 生成的 InitializeApp）
	app, cleanup, err := InitializeApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize app: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	fmt.Println("MedMemo starting...")

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
		os.Exit(1)
	}
}

// App 应用顶层封装，供 Wire 注入后返回。
type App struct {
	wailsApp *WailsApp
}

// NewApp 构造函数，供 Wire 调用。
// 启动时执行数据库迁移，返回 cleanup 回调用于关闭连接池。
func NewApp(wa *WailsApp, sqlite *database.SQLiteConnector) (*App, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := sqlite.Migrate(ctx); err != nil {
		return nil, nil, fmt.Errorf("database migration failed: %w", err)
	}

	cleanup := func() {
		if err := sqlite.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to close database: %v\n", err)
		}
	}
	return &App{wailsApp: wa}, cleanup, nil
}
