#!/bin/bash
# 跨平台 Wails 构建脚本，供 GoReleaser 或 CI 调用。
# 用法: ./scripts/build/wails-build.sh <os> <version>
set -euo pipefail

OS="${1:-linux}"
VERSION="${2:-dev}"

case "$OS" in
  linux)
    echo "[TASK-027] Building for Linux..."
    export CGO_LDFLAGS="-L$(pwd)/resources/lib/linux"
    wails build -ldflags "-s -w -X main.version=${VERSION}" -tags "webkit2_41,ORT"
    echo "[TASK-027] Building AppImage..."
    ./build/package/build-appimage.sh
    ;;
  windows)
    echo "[TASK-027] Building for Windows..."
    export CGO_CFLAGS="-IC:/msys64/mingw64/include"
    export CGO_LDFLAGS="-LC:/msys64/mingw64/lib -L$(pwd)/resources/lib/windows -ldl -lntdll"
    # 下载 ONNX Runtime 与 Tokenizers Windows 库
    if command -v pwsh &>/dev/null; then
      pwsh -ExecutionPolicy Bypass -File scripts/build/download-onnx.ps1 -Platform windows
      pwsh -ExecutionPolicy Bypass -File scripts/build/download-tokenizers.ps1
    elif command -v powershell &>/dev/null; then
      powershell -ExecutionPolicy Bypass -File scripts/build/download-onnx.ps1 -Platform windows
      powershell -ExecutionPolicy Bypass -File scripts/build/download-tokenizers.ps1
    else
      echo "[TASK-027] Warning: PowerShell not found. Skipping library download."
      echo "[TASK-027] Please manually download libraries to resources/lib/windows/"
    fi
    # 确保 Rust std 需要的 ntdll 导入库在项目目录中可用
    ntdll_found=false
    for path in "C:/msys64/mingw64/lib/libntdll.a" "C:/mingw64/lib/libntdll.a"; do
      if [ -f "$path" ]; then
        cp "$path" resources/lib/windows/libntdll.a
        echo "[TASK-027] Copied libntdll.a from $path"
        ntdll_found=true
        break
      fi
    done
    if [ "$ntdll_found" != "true" ]; then
      echo "[TASK-027] libntdll.a not found, installing mingw-w64-x86_64-crt..."
      C:/msys64/usr/bin/pacman.exe -S --noconfirm mingw-w64-x86_64-crt
      if [ -f "C:/msys64/mingw64/lib/libntdll.a" ]; then
        cp "C:/msys64/mingw64/lib/libntdll.a" resources/lib/windows/libntdll.a
        echo "[TASK-027] Copied libntdll.a after install"
      else
        echo "[TASK-027] ERROR: Still cannot find libntdll.a"
        exit 1
      fi
    fi
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
