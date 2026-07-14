#!/usr/bin/env bash
set -euo pipefail

# Markdown link checker for docs and root README files.
# Requires lychee: https://github.com/lycheeverse/lychee
if ! command -v lychee &> /dev/null; then
    echo "lychee not found. Install: cargo install lychee or download from https://github.com/lycheeverse/lychee/releases"
    exit 1
fi

lychee \
  --no-progress \
  --exclude-path "node_modules" \
  --exclude-path "web/node_modules" \
  --exclude "https://github.com/.*/releases/download/.*" \
  --exclude "https://opensource.org/licenses/.*" \
  'docs/**/*.md' '*.md'
