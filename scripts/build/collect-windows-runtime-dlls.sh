#!/bin/bash
set -euo pipefail

# Wails Windows 构建后钩子：二进制生成后、NSIS 打包前，将 MinGW 运行时 DLL 复制到 build/bin
# wails build -clean 会在编译前清空 build/bin，因此必须在构建完成后复制

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

BIN_PATH="${PROJECT_ROOT}/build/bin/MedMemo.exe"
OUTPUT_DIR="${PROJECT_ROOT}/build/bin"

PS_SCRIPT="${PROJECT_ROOT}/scripts/build/collect-windows-runtime-dlls.ps1"

if command -v pwsh &>/dev/null; then
  pwsh -ExecutionPolicy Bypass -File "$PS_SCRIPT" -BinaryPath "$BIN_PATH" -OutputDir "$OUTPUT_DIR"
elif command -v powershell &>/dev/null; then
  powershell -ExecutionPolicy Bypass -File "$PS_SCRIPT" -BinaryPath "$BIN_PATH" -OutputDir "$OUTPUT_DIR"
else
  echo "Warning: PowerShell not found, manually copying known MinGW runtime DLLs"
  for dll in libgcc_s_seh-1.dll libstdc++-6.dll libwinpthread-1.dll; do
    if [ -f "/c/msys64/mingw64/bin/$dll" ]; then
      cp "/c/msys64/mingw64/bin/$dll" "$OUTPUT_DIR/"
    else
      echo "Warning: $dll not found at /c/msys64/mingw64/bin/"
    fi
  done
fi
