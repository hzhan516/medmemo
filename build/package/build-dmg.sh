#!/bin/bash
# macOS .dmg 打包脚本（需在 macOS runner 上执行）
# 用法: ./build/package/build-dmg.sh [--arch x86_64|arm64|universal]
# 产物: build/bin/MedMemo_<arch>.dmg（未指定 arch 时回退到 MedMemo.dmg）
set -euo pipefail

echo "[TASK-027] Packaging dmg for macOS..."

APP_NAME="MedMemo"
APP_DIR="build/bin/${APP_NAME}.app"
ARCH=""
DMG_NAME="build/bin/${APP_NAME}.dmg"

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case "$1" in
        --arch)
            if [[ -z "${2:-}" ]]; then
                echo "ERROR: --arch requires a value (x86_64|arm64|universal)"
                exit 1
            fi
            ARCH="$2"
            shift 2
            ;;
        *)
            echo "ERROR: unknown argument: $1"
            exit 1
            ;;
    esac
done

if [ -n "$ARCH" ]; then
    DMG_NAME="build/bin/${APP_NAME}_${ARCH}.dmg"
fi

TMP_DMG="build/bin/${APP_NAME}-tmp.dmg"
MOUNT_DIR="/Volumes/${APP_NAME}"

if [ ! -d "$APP_DIR" ]; then
    echo "ERROR: $APP_DIR not found. Run wails build first."
    exit 1
fi

# 配置 Info.plist（如尚未由 Wails 配置完整）
PLIST="${APP_DIR}/Contents/Info.plist"
if [ -f "$PLIST" ]; then
    # 确保 LSUIElement 未设置，使应用在 Dock 中显示
    plutil -remove LSUIElement "$PLIST" 2>/dev/null || true
fi

# ad-hoc 签名（内测阶段不依赖 Apple Developer ID）
if command -v codesign &> /dev/null; then
    echo "[TASK-027] Ad-hoc signing ${APP_NAME}.app..."
    codesign --deep --force --verify --verbose --sign - "$APP_DIR" || true
fi

# 创建临时 dmg
if [ -f "$DMG_NAME" ]; then
    rm -f "$DMG_NAME"
fi

# 使用 hdiutil 创建 dmg（create-dmg 未安装时的降级方案）
if command -v create-dmg &> /dev/null; then
    create-dmg \
        --volname "${APP_NAME}" \
        --window-pos 200 120 \
        --window-size 800 400 \
        --icon-size 100 \
        --app-drop-link 600 185 \
        "${DMG_NAME}" \
        "${APP_DIR}"
else
    # 使用 hdiutil 创建基础 dmg
    mkdir -p build/package/dmg-staging
    cp -r "$APP_DIR" build/package/dmg-staging/
    hdiutil create -srcfolder build/package/dmg-staging -volname "${APP_NAME}" -fs HFS+ \
        -format UDZO -o "${DMG_NAME}" 2>/dev/null || true
    rm -rf build/package/dmg-staging
fi

if [ -f "$DMG_NAME" ]; then
    echo "[TASK-027] dmg created: ${DMG_NAME}"
else
    echo "[TASK-027] WARNING: dmg creation may have failed or requires macOS environment."
fi
