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
  BIN_DIRS=("/c/msys64/mingw64/bin" "/c/msys64/ucrt64/bin" "/c/msys64/clang64/bin")
  missing=0
  for dll in libgcc_s_seh-1.dll libstdc++-6.dll libwinpthread-1.dll; do
    found=0
    for bin_dir in "${BIN_DIRS[@]}"; do
      if [ -f "$bin_dir/$dll" ]; then
        cp "$bin_dir/$dll" "$OUTPUT_DIR/"
        echo "Copied $dll from $bin_dir"
        found=1
        break
      fi
    done
    if [ "$found" -eq 0 ]; then
      echo "Warning: $dll not found in any MinGW bin directory"
      missing=1
    fi
  done
  if [ "$missing" -ne 0 ]; then
    echo "Error: Failed to collect required runtime DLLs" >&2
    exit 1
  fi
fi
