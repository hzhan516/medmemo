#!/usr/bin/env bash
#
# 下载 ONNX Runtime 跨平台动态库到 resources/lib/<platform>/
#
# 用法:
#   ./scripts/build/download-onnx.sh              # 下载全部平台
#   ./scripts/build/download-onnx.sh --platform=linux
#   ONNX_VERSION=1.26.0 ./scripts/build/download-onnx.sh
#
# 支持平台: linux, darwin, windows, all (默认)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

ONNX_VERSION="${ONNX_VERSION:-1.26.0}"
PLATFORM="${PLATFORM:-all}"

# 解析命令行参数
for arg in "$@"; do
    case "$arg" in
        --platform=*) PLATFORM="${arg#*=}" ;;
        --help|-h)
            echo "Usage: $0 [--platform=linux|darwin|windows|all]"
            echo "Env:   ONNX_VERSION (default: 1.26.0)"
            exit 0
            ;;
    esac
done

BASE_URL="${MEDMEMO_ONNX_BASE_URL:-https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}}"

download_linux() {
    local out_dir="${PROJECT_ROOT}/resources/lib/linux"
    local archive="onnxruntime-linux-x64-${ONNX_VERSION}.tgz"
    local url="${BASE_URL}/${archive}"

    if [[ -f "${out_dir}/libonnxruntime.so.1" ]]; then
        echo "[linux] ONNX Runtime already exists, skipping"
        return 0
    fi

    echo "[linux] Downloading ONNX Runtime ${ONNX_VERSION}..."
    mkdir -p "${out_dir}"
    curl -L -o "/tmp/${archive}" "${url}"
    tar -xzf "/tmp/${archive}" -C "/tmp"
    cp -r "/tmp/onnxruntime-linux-x64-${ONNX_VERSION}/lib/"* "${out_dir}/"
    rm -rf "/tmp/${archive}" "/tmp/onnxruntime-linux-x64-${ONNX_VERSION}"
    echo "[linux] Done → ${out_dir}"
}

download_darwin() {
    local out_dir="${PROJECT_ROOT}/resources/lib/darwin"
    local tmp_dir

    if [[ -f "${out_dir}/libonnxruntime.dylib" ]]; then
        echo "[darwin] ONNX Runtime already exists, skipping"
        return 0
    fi

    echo "[darwin] Downloading ONNX Runtime ${ONNX_VERSION}..."
    mkdir -p "${out_dir}"
    tmp_dir="$(mktemp -d)"

    local universal_archive="onnxruntime-osx-universal2-${ONNX_VERSION}.tgz"
    if curl -fL --retry 3 -o "${tmp_dir}/${universal_archive}" "${BASE_URL}/${universal_archive}"; then
        tar -xzf "${tmp_dir}/${universal_archive}" -C "${tmp_dir}"
        cp -r "${tmp_dir}/onnxruntime-osx-universal2-${ONNX_VERSION}/lib/"* "${out_dir}/"
        rm -rf "${tmp_dir}"
        echo "[darwin] Done → ${out_dir}"
        return 0
    fi

    echo "[darwin] universal2 archive not available; falling back to arm64"
    local arm_archive="onnxruntime-osx-arm64-${ONNX_VERSION}.tgz"
    if ! curl -fL --retry 3 -o "${tmp_dir}/${arm_archive}" "${BASE_URL}/${arm_archive}"; then
        echo "[darwin] Error: failed to download ${arm_archive}" >&2
        rm -rf "${tmp_dir}"
        exit 1
    fi
    tar -xzf "${tmp_dir}/${arm_archive}" -C "${tmp_dir}"
    cp -r "${tmp_dir}/onnxruntime-osx-arm64-${ONNX_VERSION}/lib/"* "${out_dir}/"

    local x64_archive="onnxruntime-osx-x86_64-${ONNX_VERSION}.tgz"
    if curl -fL --retry 3 -o "${tmp_dir}/${x64_archive}" "${BASE_URL}/${x64_archive}"; then
        tar -xzf "${tmp_dir}/${x64_archive}" -C "${tmp_dir}"
        if command -v lipo &>/dev/null; then
            lipo -create \
                "${tmp_dir}/onnxruntime-osx-arm64-${ONNX_VERSION}/lib/libonnxruntime.dylib" \
                "${tmp_dir}/onnxruntime-osx-x86_64-${ONNX_VERSION}/lib/libonnxruntime.dylib" \
                -output "${out_dir}/libonnxruntime.dylib"
            echo "[darwin] Created universal libonnxruntime.dylib from arm64 + x86_64"
        else
            echo "[darwin] Warning: x86_64 archive exists but lipo is unavailable; keeping arm64 dylib" >&2
        fi
    else
        echo "[darwin] x86_64 archive not available for ${ONNX_VERSION}; keeping arm64 dylib"
    fi

    rm -rf "${tmp_dir}"
    echo "[darwin] Done → ${out_dir}"
}

download_windows() {
    local out_dir="${PROJECT_ROOT}/resources/lib/windows"
    local archive="onnxruntime-win-x64-${ONNX_VERSION}.zip"
    local url="${BASE_URL}/${archive}"

    if [[ -f "${out_dir}/onnxruntime.dll" ]]; then
        echo "[windows] ONNX Runtime already exists, skipping"
        return 0
    fi

    echo "[windows] Downloading ONNX Runtime ${ONNX_VERSION}..."
    mkdir -p "${out_dir}"
    curl -L -o "/tmp/${archive}" "${url}"

    if command -v unzip &>/dev/null; then
        unzip -q "/tmp/${archive}" -d "/tmp"
    else
        echo "[windows] 'unzip' not found, attempting with Python..."
        python3 -c "import zipfile; zipfile.ZipFile('/tmp/${archive}').extractall('/tmp')"
    fi

    cp "/tmp/onnxruntime-win-x64-${ONNX_VERSION}/lib/onnxruntime.dll" "${out_dir}/"
    cp "/tmp/onnxruntime-win-x64-${ONNX_VERSION}/lib/onnxruntime_providers_shared.dll" "${out_dir}/" 2>/dev/null || true
    rm -rf "/tmp/${archive}" "/tmp/onnxruntime-win-x64-${ONNX_VERSION}"
    echo "[windows] Done → ${out_dir}"
}

main() {
    echo "=== ONNX Runtime Downloader ==="
    echo "Version:  ${ONNX_VERSION}"
    echo "Platform: ${PLATFORM}"
    echo ""

    if ! command -v curl &>/dev/null; then
        echo "Error: 'curl' is required but not found." >&2
        exit 1
    fi

    case "$PLATFORM" in
        linux)  download_linux ;;
        darwin) download_darwin ;;
        windows) download_windows ;;
        all)
            download_linux
            echo ""
            download_darwin
            echo ""
            download_windows
            ;;
        *)
            echo "Error: Unknown platform '${PLATFORM}'. Use: linux, darwin, windows, all" >&2
            exit 1
            ;;
    esac

    echo ""
    echo "=== All done ==="
}

main
