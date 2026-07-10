#!/bin/bash
# 跨平台 Wails 构建脚本，供 GoReleaser 或 CI 调用。
# 用法: ./scripts/build/wails-build.sh <os> <version>
set -euo pipefail

OS="${1:-linux}"
export VERSION="${2:-dev}"

./scripts/build/build-frontend.sh

case "$OS" in
  linux|windows|darwin|darwin-amd64|darwin-arm64)
    # 外层仅做路由；带架构的 darwin-* 由各自分支处理
    ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

case "$OS" in
  linux)
    echo "[TASK-027] Building for Linux..."
    export MEDMEMO_ONNX_BASE_URL="https://github.com/hzhan516/medmemo/releases/download/onnx-runtime-v1.26.0"
    export MEDMEMO_TOKENIZERS_BASE_URL="https://github.com/hzhan516/medmemo/releases/download/tokenizers-v1.27.0"
    ./scripts/build/download-onnx.sh --platform=linux
    ./scripts/build/download-tokenizers.sh --platform=linux
    export CGO_LDFLAGS="-L$(pwd)/resources/lib/linux"
    wails build -s -clean -ldflags "-s -w -X main.version=${VERSION}" -tags "webkit2_41,ORT"
    echo "[TASK-027] Building AppImage..."
    ./build/package/build-appimage.sh
    echo "[TASK-027] Building DEB/RPM packages..."
    if ! command -v nfpm &>/dev/null; then
      echo "[TASK-027] Error: nfpm is required to build DEB/RPM packages. Install it from https://nfpm.goreleaser.com/"
      exit 1
    fi
    nfpm package -f build/package/nfpm.yaml -p deb -t "build/bin/MedMemo_${VERSION}_amd64.deb"
    nfpm package -f build/package/nfpm.yaml -p rpm -t "build/bin/MedMemo-${VERSION}-1.x86_64.rpm"
    ;;
  windows)
    echo "[TASK-027] Building for Windows..."
    export CGO_CFLAGS="-IC:/msys64/mingw64/include"
    export CGO_LDFLAGS="-LC:/msys64/mingw64/lib${CGO_LDFLAGS:+ ${CGO_LDFLAGS}}"
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
    
    # 预先创建 build/bin 并收集 MinGW 运行时 DLL，供后续 NSIS 打包。
    # NSIS 模板（project.nsi 第 54 行）需要 build/bin/*.dll 存在，而 wails build -nsis
    # 在构建二进制后立即调用 makensis，因此必须在 wails build 前准备好 DLL。
    mkdir -p build/bin
    if command -v pwsh &>/dev/null; then
      pwsh -ExecutionPolicy Bypass -File scripts/build/collect-windows-runtime-dlls.ps1
    elif command -v powershell &>/dev/null; then
      powershell -ExecutionPolicy Bypass -File scripts/build/collect-windows-runtime-dlls.ps1
    else
      echo "[TASK-027] Warning: PowerShell not found. Skipping runtime DLL collection."
      echo "[TASK-027] Manually copying known MinGW runtime DLLs..."
      for dll in libgcc_s_seh-1.dll libstdc++-6.dll libwinpthread-1.dll; do
        if [ -f "/c/msys64/mingw64/bin/$dll" ]; then
          cp "/c/msys64/mingw64/bin/$dll" build/bin/
        else
          echo "[TASK-027] Warning: $dll not found at /c/msys64/mingw64/bin/"
        fi
      done
    fi
    
    # choco 安装的 NSIS 不会自动写入后续 CI 步骤的 PATH（同一 job 内步骤间 PATH 不刷新），
    # 这里显式补全 makensis 所在目录，确保 `wails build -nsis` 能生成安装程序而非静默跳过。
    if ! command -v makensis &>/dev/null; then
      for nsis_dir in "/c/Program Files (x86)/NSIS" "/c/Program Files/NSIS"; do
        if [ -x "${nsis_dir}/makensis.exe" ]; then
          export PATH="${nsis_dir}:${PATH}"
          echo "[TASK-027] Added NSIS to PATH: ${nsis_dir}"
          break
        fi
      done
    fi
    if ! command -v makensis &>/dev/null; then
      echo "[TASK-027] Error: makensis not found; cannot build the NSIS installer." >&2
      exit 1
    fi
    wails build -s -clean -ldflags "-s -w -X main.version=${VERSION}" -tags "ORT" -nsis
    ./scripts/build/copy-runtime-resources.sh "build/bin" "windows"
    ;;
  darwin|darwin-amd64|darwin-arm64)
    echo "[TASK-027] Building for macOS..."
    export MEDMEMO_ONNX_BASE_URL="https://github.com/hzhan516/medmemo/releases/download/onnx-runtime-v1.26.0"
    export MEDMEMO_TOKENIZERS_BASE_URL="https://github.com/hzhan516/medmemo/releases/download/tokenizers-v1.27.0"
    ./scripts/build/download-onnx.sh --platform=darwin
    ./scripts/build/download-tokenizers.sh --platform=darwin
    case "$OS" in
      darwin-amd64)
        DARWIN_PLATFORM="darwin/amd64"
        DMG_ARCH="x86_64"
        ;;
      darwin-arm64)
        DARWIN_PLATFORM="darwin/arm64"
        DMG_ARCH="arm64"
        ;;
      *)
        DARWIN_PLATFORM="${MEDMEMO_DARWIN_PLATFORM:-darwin/arm64}"
        DMG_ARCH="arm64"
        ;;
    esac
    REQUIRE_UNIVERSAL="false"
    if [ "$DARWIN_PLATFORM" = "darwin/universal" ]; then
      REQUIRE_UNIVERSAL="true"
      DMG_ARCH="universal"
    fi
    wails build -s -clean -ldflags "-s -w -X main.version=${VERSION}" -tags "ORT" -platform "$DARWIN_PLATFORM"
    ./scripts/build/copy-runtime-resources.sh "build/bin/MedMemo.app/Contents/Resources" "darwin" "$REQUIRE_UNIVERSAL"
    echo "[TASK-027] Building dmg..."
    ./build/package/build-dmg.sh --arch "$DMG_ARCH"
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
