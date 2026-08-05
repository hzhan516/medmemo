#!/bin/bash
# Release asset whitelist verification.
# Usage: ./scripts/build/verify-release-assets.sh <directory> [version]
#
# Allowed release asset patterns are defined in scripts/build/release-assets-manifest.json
# and substituted with the provided version (defaults to the version inferred from DEB/RPM names).
#
# Any other file in the directory causes a non-zero exit.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST="$SCRIPT_DIR/release-assets-manifest.json"

DIR="${1:-release-assets}"
VERSION="${2:-}"

if [ ! -d "$DIR" ]; then
    echo "ERROR: directory not found: $DIR"
    exit 1
fi

if [ -z "$(ls -A "$DIR")" ]; then
    echo "ERROR: release asset directory is empty: $DIR"
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

# 若未传入版本，尝试从 DEB/RPM 文件名推断
if [ -z "$VERSION" ]; then
    for f in "$DIR"/*; do
        name=$(basename "$f")
        case "$name" in
            MedMemo_*_amd64.deb)
                VERSION="${name#MedMemo_}"
                VERSION="${VERSION%_amd64.deb}"
                break
                ;;
            MedMemo-*-1.x86_64.rpm)
                VERSION="${name#MedMemo-}"
                VERSION="${VERSION%-1.x86_64.rpm}"
                break
                ;;
        esac
    done
fi

if [ -z "$VERSION" ]; then
    echo "ERROR: cannot infer version from assets; provide version as second argument"
    exit 1
fi

# 构建白名单（checksums.txt + manifest 中替换 ${VERSION} 后的名称）
WHITELIST=("checksums.txt")
while IFS= read -r name_template; do
    name="${name_template/\$\{VERSION\}/$VERSION}"
    WHITELIST+=("$name")
done < <(jq -r '.assets[].name' "$MANIFEST")

EXIT_CODE=0

shopt -s nullglob
for path in "$DIR"/*; do
    if [ ! -f "$path" ]; then
        echo "ERROR: non-file entry is not allowed as a release asset: $path"
        EXIT_CODE=1
        continue
    fi

    name=$(basename "$path")
    found=false
    for allowed in "${WHITELIST[@]}"; do
        if [ "$name" = "$allowed" ]; then
            found=true
            break
        fi
    done

    if [ "$found" = "true" ]; then
        echo "OK: $name"
    else
        echo "ERROR: disallowed release asset: $name"
        EXIT_CODE=1
    fi
done

if [ "$EXIT_CODE" -ne 0 ]; then
    echo "ERROR: release asset whitelist check failed for $DIR"
    exit 1
fi

EXPECTED_COUNT=$((${#WHITELIST[@]}))
ACTUAL_COUNT=$(find "$DIR" -maxdepth 1 -type f | wc -l)
if [ "$ACTUAL_COUNT" -ne "$EXPECTED_COUNT" ]; then
    echo "ERROR: release asset count mismatch: expected $EXPECTED_COUNT, found $ACTUAL_COUNT"
    exit 1
fi

echo "OK: all release assets in $DIR match the whitelist (version=$VERSION)"
