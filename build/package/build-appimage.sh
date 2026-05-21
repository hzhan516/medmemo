#!/bin/bash
# Linux AppImage 打包脚本
# 产物: build/bin/MedMemo-x86_64.AppImage
# 注意：AppImage 依赖目标系统预装 webkit2gtk-4.1（Ubuntu 22.04+/Fedora 38+ 已内置）
set -euo pipefail

echo "[TASK-027] Packaging AppImage..."

# 确保产物目录存在
mkdir -p build/bin

# 下载 appimagetool（如果不存在）
APPIMAGETOOL="build/ci/appimagetool-x86_64.AppImage"
if [ ! -f "$APPIMAGETOOL" ]; then
    mkdir -p build/ci
    curl -sL -o "$APPIMAGETOOL" \
        "https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage"
    chmod +x "$APPIMAGETOOL"
fi

# 创建 AppDir 结构
APPDIR="build/package/AppDir"
rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin"
mkdir -p "$APPDIR/usr/share/applications"
mkdir -p "$APPDIR/usr/share/icons/hicolor/256x256/apps"

# 复制二进制与资源
cp build/bin/MedMemo "$APPDIR/usr/bin/"
cp -r resources "$APPDIR/usr/bin/" 2>/dev/null || true

# 复制图标（appimagetool 要求在 AppDir 根目录）
cp build/appicon.png "$APPDIR/medmemo.png" 2>/dev/null || true

# 创建 .desktop 文件（appimagetool 要求在 AppDir 根目录）
cat > "$APPDIR/medmemo.desktop" <<EOF
[Desktop Entry]
Name=MedMemo
Exec=MedMemo
Icon=medmemo
Type=Application
Categories=Utility;MedicalSoftware;
Comment=Your personal health memory assistant
EOF

# 创建 AppRun 入口脚本
cat > "$APPDIR/AppRun" <<'EOF'
#!/bin/bash
HERE="$(dirname "$(readlink -f "$0")")"
export PATH="${HERE}/usr/bin:${PATH}"

# AppImage 内部为只读文件系统，确保数据目录落在用户可写区域
if [ -z "$HOME" ]; then
    HOME="$(getent passwd "$(id -u)" | cut -d: -f6 2>/dev/null)" || HOME="/tmp"
    export HOME
fi

# 切换到用户主目录，避免在只读挂载点创建 .medmemo
cd "$HOME" || exit 1

exec "${HERE}/usr/bin/MedMemo" "$@"
EOF
chmod +x "$APPDIR/AppRun"

# 使用 appimagetool 生成 AppImage
OUTPUT="build/bin/MedMemo-x86_64.AppImage"
APPIMAGE_EXTRACT_AND_RUN=1 "$APPIMAGETOOL" "$APPDIR" "$OUTPUT"

echo "[TASK-027] AppImage created: $OUTPUT"
ls -lh "$OUTPUT"
