#!/bin/bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIST_DIR="${PROJECT_ROOT}/web/dist"
INDEX_FILE="${DIST_DIR}/index.html"

if [ ! -s "$INDEX_FILE" ]; then
  echo "[build] missing frontend asset: web/dist/index.html"
  echo "[build] run ./scripts/build/build-frontend.sh before wails build"
  exit 1
fi

file_count="$(find "$DIST_DIR" -type f | wc -l | tr -d ' ')"
if [ "$file_count" -lt 1 ]; then
  echo "[build] frontend dist is empty: web/dist"
  exit 1
fi

echo "[build] frontend assets verified: ${file_count} files"
