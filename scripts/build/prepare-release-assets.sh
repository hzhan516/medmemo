#!/bin/bash
# 汇集四平台构建产物为统一的六文件 Release 资产目录。
# 单一事实源：scripts/build/release-assets-manifest.json
# 用法: ./scripts/build/prepare-release-assets.sh <version> <output-dir> [linux-dir] [windows-dir] [darwin-amd64-dir] [darwin-arm64-dir]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
MANIFEST="${MEDMEMO_RELEASE_MANIFEST:-$REPO_ROOT/scripts/build/release-assets-manifest.json}"

VERSION="${1:-}"
OUTPUT_DIR="${2:-}"
LINUX_DIR="${3:-$REPO_ROOT/artifacts/linux}"
WINDOWS_DIR="${4:-$REPO_ROOT/artifacts/windows}"
DARWIN_AMD64_DIR="${5:-$REPO_ROOT/artifacts/darwin-amd64}"
DARWIN_ARM64_DIR="${6:-$REPO_ROOT/artifacts/darwin-arm64}"

if [ -z "$VERSION" ] || [ -z "$OUTPUT_DIR" ]; then
    echo "ERROR: usage: $0 <version> <output-dir> [linux-dir] [windows-dir] [darwin-amd64-dir] [darwin-arm64-dir]"
    exit 1
fi

if [ ! -f "$MANIFEST" ]; then
    echo "ERROR: release asset manifest not found: $MANIFEST"
    exit 1
fi

if ! command -v jq &>/dev/null; then
    echo "ERROR: jq is required"
    exit 1
fi

mkdir -p "$OUTPUT_DIR"
# 先清空输出目录，防止历史残留或重名导致误判
rm -f "$OUTPUT_DIR"/*

# 按平台映射源目录
resolve_source_dir() {
    local platform="$1"
    case "$platform" in
        linux) echo "$LINUX_DIR" ;;
        windows) echo "$WINDOWS_DIR" ;;
        darwin)
            local arch="$2"
            if [ "$arch" = "x86_64" ]; then
                echo "$DARWIN_AMD64_DIR"
            else
                echo "$DARWIN_ARM64_DIR"
            fi
            ;;
        *)
            echo "ERROR: unknown platform: $platform" >&2
            exit 1
            ;;
    esac
}

EXPECTED_COUNT=$(jq '.assets | length' "$MANIFEST")
COPIED=0

while IFS= read -r asset; do
    name_template=$(echo "$asset" | jq -r '.name')
    platform=$(echo "$asset" | jq -r '.platform')
    arch=$(echo "$asset" | jq -r '.arch')
    kind=$(echo "$asset" | jq -r '.kind')

    # 替换 ${VERSION} 占位符
    name="${name_template/\$\{VERSION\}/$VERSION}"

    src_dir=$(resolve_source_dir "$platform" "$arch")
    src_path="$src_dir/$name"
    dst_path="$OUTPUT_DIR/$name"

    if [ ! -f "$src_path" ]; then
        echo "ERROR: missing release asset: $src_path (platform=$platform arch=$arch kind=$kind)"
        exit 1
    fi

    if [ -e "$dst_path" ]; then
        echo "ERROR: duplicate release asset name: $name"
        exit 1
    fi

    cp -a "$src_path" "$dst_path"
    COPIED=$((COPIED + 1))
    echo "OK: staged $name"
done < <(jq -c '.assets[]' "$MANIFEST")

# 检查输出目录没有额外文件
ACTUAL_COUNT=$(find "$OUTPUT_DIR" -maxdepth 1 -type f | wc -l)
if [ "$ACTUAL_COUNT" -ne "$EXPECTED_COUNT" ]; then
    echo "ERROR: asset count mismatch: expected $EXPECTED_COUNT, found $ACTUAL_COUNT"
    ls -la "$OUTPUT_DIR"
    exit 1
fi

if [ "$COPIED" -ne "$EXPECTED_COUNT" ]; then
    echo "ERROR: copied asset count mismatch: expected $EXPECTED_COUNT, copied $COPIED"
    exit 1
fi

echo "OK: staged $COPIED release assets to $OUTPUT_DIR"
