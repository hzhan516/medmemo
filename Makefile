.PHONY: all dev build test lint wire clean install-tools

# 强制使用项目要求的 Go 工具链，避免本地旧版本导致解析 go.mod 失败
GOTOOLCHAIN ?= go1.26.4
export GOTOOLCHAIN

# 版本号从 wails.json 读取（单一来源，使用 JSON 解析器而非 grep/sed）
VERSION ?= $(shell go run scripts/read-version.go wails.json)
LDFLAGS := -ldflags "-s -w -X main.version=v$(VERSION)"

# golangci-lint 固定 v1 版本，避免 v2 配置不兼容导致 CI 不可复现
GOLANGCI_LINT_VERSION ?= v1.64.8
GOLANGCI_LINT := $(shell go env GOPATH)/bin/golangci-lint

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
	./scripts/build/build-frontend.sh
	CGO_LDFLAGS="$(CGO_LDFLAGS_LINUX)" wails build -s -clean -tags "$(BUILD_TAGS)" $(LDFLAGS)

# CGO 库路径（用于测试，go test 时 ${SRCDIR} 解析为临时目录，需显式指定）
CGO_LDFLAGS_LINUX := -L$(shell pwd)/resources/lib/linux
CGO_LDFLAGS_DARWIN := -L$(shell pwd)/resources/lib/darwin
CGO_LDFLAGS_WINDOWS := -L$(shell pwd)/resources/lib/windows -LC:/msys64/mingw64/lib -ldl

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
	$(GOLANGCI_LINT) run --timeout=10m ./...

# 生成 Wire 依赖注入代码
wire:
	wire .

# 安装开发依赖工具
install-tools:
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}
	$(GOLANGCI_LINT) version
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

DARWIN_PLATFORM ?= darwin/arm64
DARWIN_REQUIRE_UNIVERSAL = $(if $(filter darwin/universal,$(DARWIN_PLATFORM)),true,false)

# 交叉编译（需对应平台环境）
build-darwin:
	./scripts/build/build-frontend.sh
	wails build -s -platform $(DARWIN_PLATFORM) -clean -tags ORT $(LDFLAGS)
	./scripts/build/copy-runtime-resources.sh build/bin/MedMemo.app/Contents/Resources darwin $(DARWIN_REQUIRE_UNIVERSAL)

build-windows:
	./scripts/build/build-frontend.sh
	# Windows 上若缺少 libtokenizers.a，ORT tag 会导致编译失败。
	# 首次构建前请运行: .\scripts\build\download-tokenizers.ps1
	CGO_LDFLAGS="$(CGO_LDFLAGS_WINDOWS)" wails build -s -platform windows/amd64 -clean -tags ORT $(LDFLAGS)
	./scripts/build/copy-runtime-resources.sh build/bin windows

build-linux:
	./scripts/build/build-frontend.sh
	CGO_LDFLAGS="$(CGO_LDFLAGS_LINUX)" wails build -s -platform linux/amd64 -clean -tags webkit2_41,ORT $(LDFLAGS)

# 本地完整打包（当前平台，含版本注入与平台安装包）
release-local:
	CGO_LDFLAGS="$(CGO_LDFLAGS_LINUX)" ./scripts/build/wails-build.sh $(shell go env GOOS) v$(VERSION)

# GoReleaser 本地快照验证（不发布，仅验证配置与归档）
release-dry-run:
	cd web && npm ci && npm run build
	PLATFORM=local goreleaser release --snapshot --clean
