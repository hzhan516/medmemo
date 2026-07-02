#!/bin/bash
# Release asset whitelist verification.
# Usage: ./scripts/build/verify-release-assets.sh <directory>
#
# Allowed release asset patterns:
#   MedMemoSetup.exe
#   MedMemo_x86_64.dmg
#   MedMemo_arm64.dmg
#   MedMemo-x86_64.AppImage
#   MedMemo_*.deb
#   MedMemo-*.rpm
#   checksums.txt
#
# Any other file in the directory causes a non-zero exit.
set -euo pipefail

DIR="${1:-release-assets}"

if [ ! -d "$DIR" ]; then
    echo "ERROR: directory not found: $DIR"
    exit 1
fi

if [ -z "$(ls -A "$DIR")" ]; then
    echo "ERROR: release asset directory is empty: $DIR"
    exit 1
fi

EXIT_CODE=0

shopt -s nullglob
for path in "$DIR"/*; do
    if [ ! -f "$path" ]; then
        echo "ERROR: non-file entry is not allowed as a release asset: $path"
        EXIT_CODE=1
        continue
    fi

    name=$(basename "$path")
    case "$name" in
        MedMemoSetup.exe|\
        MedMemo_x86_64.dmg|\
        MedMemo_arm64.dmg|\
        MedMemo-x86_64.AppImage|\
        MedMemo_*.deb|\
        MedMemo-*.rpm|\
        checksums.txt)
            echo "OK: $name"
            ;;
        *)
            echo "ERROR: disallowed release asset: $name"
            EXIT_CODE=1
            ;;
    esac
done

if [ "$EXIT_CODE" -ne 0 ]; then
    echo "ERROR: release asset whitelist check failed for $DIR"
    exit 1
fi

echo "OK: all release assets in $DIR match the whitelist"
