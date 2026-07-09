#!/bin/bash
set -euo pipefail

PLATFORM="${1:-linux}"

verify_binary() {
    local bin="$1"

    if [ ! -f "$bin" ]; then
        echo "ERROR: binary not found: $bin"
        exit 1
    fi

    if ! strings "$bin" | grep 'web/dist/index.html' >/dev/null 2>&1; then
        echo "ERROR: web/dist/index.html not embedded in $bin"
        exit 1
    fi

    if ! strings "$bin" | grep 'web/dist/assets/index-' >/dev/null 2>&1; then
        echo "ERROR: web/dist/assets/index-* not embedded in $bin"
        exit 1
    fi

    echo "OK: $bin contains embedded frontend assets"
}

case "$PLATFORM" in
  linux)
    verify_binary "build/bin/MedMemo"
    if [ -f "build/bin/MedMemo-x86_64.AppImage" ]; then
      chmod +x build/bin/MedMemo-x86_64.AppImage
      ./build/bin/MedMemo-x86_64.AppImage --appimage-extract >/dev/null 2>&1
      verify_binary "squashfs-root/usr/bin/MedMemo"
      rm -rf squashfs-root
    fi
    ;;
  windows)
    verify_binary "build/bin/MedMemo.exe"
    INSTALLER="build/bin/MedMemo-amd64-installer.exe"
    if [ ! -f "$INSTALLER" ]; then
      echo "ERROR: Windows installer not found: $INSTALLER"
      exit 1
    fi
    echo "OK: $INSTALLER exists"
    ;;
  darwin|darwin-amd64|darwin-arm64)
    ARCH="${2:-}"
    if [ -z "$ARCH" ]; then
      # 兼容 workflow 直接传入 darwin-amd64/darwin-arm64 的场景
      case "$PLATFORM" in
        darwin-amd64) ARCH="x86_64" ;;
        darwin-arm64) ARCH="arm64" ;;
        *) ARCH="arm64" ;;
      esac
    fi
    DMG_FILE="build/bin/MedMemo_${ARCH}.dmg"
    if [ ! -f "$DMG_FILE" ]; then
      echo "ERROR: $DMG_FILE not found"
      exit 1
    fi
    verify_binary "build/bin/MedMemo.app/Contents/MacOS/MedMemo"
    ;;
  *)
    echo "Unsupported platform: $PLATFORM"
    exit 1
    ;;
esac
