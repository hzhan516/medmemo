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
    ;;
  darwin)
    verify_binary "build/bin/MedMemo.app/Contents/MacOS/MedMemo"
    if [ -f "build/bin/MedMemo" ]; then
      verify_binary "build/bin/MedMemo"
    fi
    ;;
  *)
    echo "Unsupported platform: $PLATFORM"
    exit 1
    ;;
esac
