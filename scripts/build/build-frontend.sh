#!/bin/bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

cd "${PROJECT_ROOT}/web"

# CI 环境下强制重新安装，避免复用前面步骤残留的脏 node_modules
if [ "${CI:-false}" = "true" ] || [ ! -d node_modules ]; then
  if [ -f package-lock.json ]; then
    npm ci
  else
    npm install
  fi
fi

# 打印关键依赖版本，用于 CI 日志排查
npm ls vite @vitejs/plugin-react typescript || true

npm run build

"${PROJECT_ROOT}/scripts/build/verify-web-dist.sh"
