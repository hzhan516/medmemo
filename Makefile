.PHONY: all dev build test lint wire clean install-tools

# 默认目标
all: build

# 开发模式（热重载）
dev:
	wails dev

# 生产构建（当前平台）
build:
	wails build -clean

# 运行测试
test:
	go test -race -coverprofile=coverage.out ./...

# 运行集成测试
test-integration:
	go test -race -tags=integration ./...

# 查看测试覆盖率
coverage: test
	go tool cover -html=coverage.out -o coverage.html

# 代码检查
lint:
	golangci-lint run ./...

# 生成 Wire 依赖注入代码
wire:
	wire ./cmd/health-assistant

# 安装开发依赖工具
install-tools:
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/vektra/mockery/v2@latest

# 下载模型资源（开发环境）
download-resources:
	python scripts/download_models.py --output resources/models
	python scripts/download_dicts.py --output resources/dict

# 格式化代码
fmt:
	gofmt -w .
	cd web && npm run lint -- --fix

# 清理构建产物
clean:
	rm -rf build/bin/
	rm -rf web/dist/
	rm -f coverage.out coverage.html

# 交叉编译（需对应平台环境）
build-darwin:
	wails build -platform darwin/universal -clean

build-windows:
	wails build -platform windows/amd64 -clean

build-linux:
	wails build -platform linux/amd64 -clean
