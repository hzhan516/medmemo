#!/usr/bin/env bash
set -euo pipefail

# Generate Go dependency license table.
# Uses a Node.js wrapper around `go list -m -json all` to avoid go-licenses
# compatibility issues with Go 1.26+ standard-library modules.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$ROOT_DIR"
export GOTOOLCHAIN=go1.26.4

# go list -m 不触发模块下载；普通 `go mod download` 只会拉取当前构建所需的模块，
# 而 go.mod 中还有不少未被主构建图引用的间接依赖。若这些模块缓存缺失，
# `go list -m -json` 不会输出 Dir，许可证检测会全部回退为 UNKNOWN。
# 使用 `go mod download all` 强制下载所有声明依赖的源码，确保许可证识别准确。
go mod download all

node "$SCRIPT_DIR/generate-go-licenses.js"
