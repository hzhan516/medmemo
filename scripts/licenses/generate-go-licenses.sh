#!/usr/bin/env bash
set -euo pipefail

# Generate Go dependency license table.
# Uses a Node.js wrapper around `go list -m -json all` to avoid go-licenses
# compatibility issues with Go 1.26+ standard-library modules.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$ROOT_DIR"
export GOTOOLCHAIN=go1.26.4
node "$SCRIPT_DIR/generate-go-licenses.js"
