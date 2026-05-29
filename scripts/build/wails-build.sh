#!/bin/bash
# 跨平台 Wails 构建脚本，供 GoReleaser 或 CI 调用。
# 用法: ./scripts/build/wails-build.sh <os> <version>
set -euo pipefail

OS="${1:-linux}"
VERSION="${2:-dev}"

case "$OS" in
  linux)
    echo "[TASK-027] Building for Linux..."
    export MEDMEMO_ONNX_BASE_URL="https://github.com/hzhan516/medmemo/releases/download/onnx-runtime-v1.26.0"
    export MEDMEMO_TOKENIZERS_BASE_URL="https://github.com/hzhan516/medmemo/releases/download/tokenizers-v1.27.0"
    ./scripts/build/download-onnx.sh --platform=linux
    ./scripts/build/download-tokenizers.sh --platform=linux
    export CGO_LDFLAGS="-L$(pwd)/resources/lib/linux"
    wails build -ldflags "-s -w -X main.version=${VERSION}" -tags "webkit2_41,ORT"
    echo "[TASK-027] Building AppImage..."
    ./build/package/build-appimage.sh
    ;;
  windows)
    echo "[TASK-027] Building for Windows..."
    export CGO_CFLAGS="-IC:/msys64/mingw64/include"
    # 不再 export CGO_LDFLAGS —— 避免含空格路径被 Go CGO 安全检查拒绝
    # cgo_ort_libs_windows.go 已提供 -L${SRCDIR}/resources/lib/windows -lntdll
    # ortgenai / onnxruntime_go 已自动注入 -ldl
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
    # MinGW-w64 官方 libntdll.a 不包含 Native API 符号，必须从 ntdll.dll 现场生成
    echo "[TASK-027] Generating libntdll.a from system ntdll.dll..."
    objdump -p /c/Windows/System32/ntdll.dll | \
      awk '/Ordinal.*Hint.*Name/{start=1; next} start && /^[[:space:]]*$/ {exit} start {last=$NF; if(last ~ /[a-zA-Z]/) print last}' \
      > /tmp/ntdll_exports.txt

    echo "LIBRARY ntdll.dll" > /tmp/ntdll.def
    echo "EXPORTS" >> /tmp/ntdll.def
    cat /tmp/ntdll_exports.txt >> /tmp/ntdll.def

    dlltool -D ntdll.dll -d /tmp/ntdll.def -l resources/lib/windows/libntdll.a
    echo "[TASK-027] Generated libntdll.a with $(wc -l < /tmp/ntdll_exports.txt) exports"
    wails build -ldflags "-s -w -X main.version=${VERSION}" -tags "ORT" -nsis
    ;;
  darwin)
    echo "[TASK-027] Building for macOS..."
    export MEDMEMO_ONNX_BASE_URL="https://github.com/hzhan516/medmemo/releases/download/onnx-runtime-v1.26.0"
    export MEDMEMO_TOKENIZERS_BASE_URL="https://github.com/hzhan516/medmemo/releases/download/tokenizers-v1.27.0"
    ./scripts/build/download-onnx.sh --platform=darwin
    ./scripts/build/download-tokenizers.sh --platform=darwin
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
