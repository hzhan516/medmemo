#!/usr/bin/env bash
set -euo pipefail

# Generate Go dependency license table.
# Uses a Node.js wrapper around `go list -m -json all` to avoid go-licenses
# compatibility issues with Go 1.26+ standard-library modules.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$ROOT_DIR"
export GOTOOLCHAIN=go1.26.4

# go list -m 不触发模块下载；若模块缓存缺失则许可证检测会全部回退为 UNKNOWN，
# 因此生成前必须先填充模块缓存。
go mod download

node "$SCRIPT_DIR/generate-go-licenses.js"
