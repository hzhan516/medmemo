// Package main 是 MedMemo 应用入口。
// 仅负责依赖组装与生命周期管理，不包含业务逻辑。
package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/hzhan516/medmemo/internal/adapters/ai"
	"github.com/hzhan516/medmemo/internal/application/pipeline"
	"github.com/hzhan516/medmemo/internal/application/port"
	"github.com/hzhan516/medmemo/internal/domain/entity"
	"github.com/hzhan516/medmemo/internal/infrastructure/config"
	"github.com/hzhan516/medmemo/internal/infrastructure/database"
	"github.com/hzhan516/medmemo/internal/infrastructure/onnx"
	"github.com/hzhan516/medmemo/internal/infrastructure/secret"
	"github.com/hzhan516/medmemo/pkg/models"
	"github.com/hzhan516/medmemo/pkg/resourcepath"
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
// 在返回前确保 embedding 模型已复制到用户数据目录，保证 onnx.NewEngine 能正确加载。
func NewEngineConfig(cfg *entity.AppConfig) onnx.EngineConfig {
	userPath := filepath.Join(cfg.DataDir, "models", models.EmbeddingModelName)
	resourceDir := resourcepath.Dir()
	prepareEmbeddingModels(userPath, resourceDir)

	return onnx.EngineConfig{
		ResourceDir:        resourceDir,
		ModelPath:          resourcepath.Resolve(cfg.ModelDir),
		EmbeddingModelPath: userPath,
	}
}

// prepareEmbeddingModels 首次启动时将打包的模型文件复制到用户数据目录。
// 若用户目录已有模型，或找不到打包模型，则静默跳过。
func prepareEmbeddingModels(userDir string, resourceDir string) {
	if _, err := os.Stat(filepath.Join(userDir, "model.onnx")); err == nil {
		return
	}

	bundledDir := findBundledModelDir(resourceDir)
	if bundledDir == "" {
		return
	}

	_ = os.MkdirAll(userDir, 0755)
	for _, name := range []string{"model.onnx", "tokenizer.json"} {
		src := filepath.Join(bundledDir, name)
		dst := filepath.Join(userDir, name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		_ = os.WriteFile(dst, data, 0600)
	}
}

// findBundledModelDir 查找应用包内打包的模型目录。
func findBundledModelDir(resourceDir string) string {
	dir := filepath.Join(resourceDir, "models", models.EmbeddingModelName)
	if _, err := os.Stat(filepath.Join(dir, "model.onnx")); err == nil {
		return dir
	}
	return ""
}

// NewDefaultLoader 创建使用默认搜索路径的配置加载器。
func NewDefaultLoader() *config.Loader {
	return config.NewLoader("")
}

// NewSQLCipherConnectorFromConfig 从 AppConfig 获取数据目录创建数据库连接。
func NewSQLCipherConnectorFromConfig(cfg *entity.AppConfig, store secret.Store) (*database.SQLCipherConnector, error) {
	return database.NewSQLCipherConnector(cfg.DataDir, store)
}

// NewEmbeddingServiceAdapterWithVersion 创建带固定模型版本的嵌入服务适配器。
func NewEmbeddingServiceAdapterWithVersion(engine ai.EmbeddingEngine) *ai.EmbeddingServiceAdapter {
	return ai.NewEmbeddingServiceAdapter(engine, models.CurrentEmbeddingVersion)
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
