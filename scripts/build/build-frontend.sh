#!/bin/bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

cd "${PROJECT_ROOT}/web"
if [ ! -d node_modules ]; then
  if [ -f package-lock.json ]; then
    npm ci
  else
    npm install
  fi
fi
npm run build

"${PROJECT_ROOT}/scripts/build/verify-web-dist.sh"
