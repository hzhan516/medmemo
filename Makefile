.PHONY: all dev build test lint wire clean install-tools

# 版本号从 wails.json 读取（单一来源）
VERSION ?= $(shell cat wails.json | grep '"productVersion"' | sed 's/.*"productVersion": *"\(.*\)".*/\1/')
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

# 平台检测（用于本地构建）
UNAME_S := $(shell uname -s)
BUILD_TAGS := ORT
ifeq ($(UNAME_S),Linux)
	BUILD_TAGS := webkit2_41,ORT
endif

# 默认目标
all: build

# 开发模式（热重载）
# Fedora 43+ 使用 webkit2gtk-4.1，需传递 -tags webkit2_41
# ORT tag 启用 ONNX Runtime 后端（Embedding + NER）
dev:
	cd web && npm install
	wails dev -tags "$(BUILD_TAGS)"

# 生产构建（当前平台）
# 先手动构建前端，再调用 wails build（Wails v2.12 在 frontend.dir != frontend 时可能跳过前端构建）
build:
	cd web && npm install && npm run build
	wails build -clean -tags "$(BUILD_TAGS)" $(LDFLAGS)

# CGO 库路径（用于测试，go test 时 ${SRCDIR} 解析为临时目录，需显式指定）
CGO_LDFLAGS_LINUX := -L$(shell pwd)/resources/lib/linux
CGO_LDFLAGS_DARWIN := -L$(shell pwd)/resources/lib/darwin

# 运行测试（含 ORT 后端）
test:
	CGO_LDFLAGS="$(CGO_LDFLAGS_LINUX)" go test -tags "$(BUILD_TAGS)" -race -coverprofile=coverage.out ./...

# 运行集成测试
test-integration:
	CGO_LDFLAGS="$(CGO_LDFLAGS_LINUX)" go test -race -tags=integration,ORT ./...

# 运行 E2E 测试
test-e2e:
	go test -tags=e2e ./e2e/go/...

# 查看测试覆盖率
coverage: test
	go tool cover -html=coverage.out -o coverage.html

# 代码检查
lint:
	golangci-lint run ./...

# 生成 Wire 依赖注入代码
wire:
	wire .

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
	cd web && npm install && npm run build
	wails build -platform darwin/universal -clean -tags ORT $(LDFLAGS)

build-windows:
	cd web && npm install && npm run build
	# Windows 上若缺少 libtokenizers.a，ORT tag 会导致编译失败。
	# 首次构建前请运行: .\scripts\build\download-tokenizers.ps1
	wails build -platform windows/amd64 -clean -tags ORT $(LDFLAGS)

build-linux:
	cd web && npm install && npm run build
	wails build -platform linux/amd64 -clean -tags webkit2_41,ORT $(LDFLAGS)

# 本地完整打包（当前平台，含版本注入与平台安装包）
release-local:
	./scripts/build/wails-build.sh $(shell go env GOOS) $(VERSION)

# GoReleaser 本地快照验证（不发布，仅验证配置与归档）
release-dry-run:
	cd web && npm ci && npm run build
	PLATFORM=local goreleaser release --snapshot --clean
