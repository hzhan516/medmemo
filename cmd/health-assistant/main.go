// Package main 是 MedMemo 应用入口。
// 仅负责依赖组装与生命周期管理，不包含业务逻辑。
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听优雅关闭信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n received shutdown signal, gracefully stopping...")
		cancel()
	}()

	// 初始化应用（通过 Wire 生成的 InitializeApp）
	app, cleanup, err := InitializeApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize app: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	fmt.Println("MedMemo started successfully")

	// 运行应用主循环
	if err := app.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "app run error: %v\n", err)
		os.Exit(1)
	}
}

// App 应用顶层封装，供 Wire 注入后返回。
type App struct {
	// TODO(作者): 注入 Wails 运行时、各用例、托盘管理等 [Issue#025]
}

// Run 启动应用主循环。
func (a *App) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

// NewApp 构造函数，供 Wire 调用。
func NewApp() *App {
	return &App{}
}
