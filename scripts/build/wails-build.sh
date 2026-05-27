#!/bin/bash
# 跨平台 Wails 构建脚本，供 GoReleaser 或 CI 调用。
# 用法: ./scripts/build/wails-build.sh <os> <version>
set -euo pipefail

OS="${1:-linux}"
VERSION="${2:-dev}"

case "$OS" in
  linux)
    echo "[TASK-027] Building for Linux..."
    wails build -ldflags "-s -w -X main.version=${VERSION}" -tags "webkit2_41,ORT"
    echo "[TASK-027] Building AppImage..."
    ./build/package/build-appimage.sh
    ;;
  windows)
    echo "[TASK-027] Building for Windows..."
    wails build -ldflags "-s -w -X main.version=${VERSION}" -tags "ORT" -nsis
    ;;
  darwin)
    echo "[TASK-027] Building for macOS..."
    wails build -ldflags "-s -w -X main.version=${VERSION}" -tags "ORT" -platform darwin/universal
    echo "[TASK-027] Building dmg..."
    ./build/package/build-dmg.sh
    # GoReleaser prebuilt 期望 build/bin/MedMemo，而 Wails macOS 产物为 .app bundle
    # 复制二进制供 GoReleaser 归档，用户实际应使用 .dmg
    if [ -d build/bin/MedMemo.app ]; then
      cp build/bin/MedMemo.app/Contents/MacOS/MedMemo build/bin/MedMemo
    fi
    ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

echo "[TASK-027] Build complete for $OS."
