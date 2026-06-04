#!/usr/bin/env bash
#
# 下载 daulet/tokenizers 跨平台静态库到 resources/lib/<platform>/
#
# 用法:
#   ./scripts/build/download-tokenizers.sh              # 下载全部平台
#   ./scripts/build/download-tokenizers.sh --platform=linux
#
# 支持平台: linux, darwin, all (默认，Windows 无官方预编译库)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

TOKENIZERS_VERSION="${TOKENIZERS_VERSION:-v1.27.0}"
PLATFORM="${PLATFORM:-all}"

# 解析命令行参数
for arg in "$@"; do
    case "$arg" in
        --platform=*) PLATFORM="${arg#*=}" ;;
        --help|-h)
            echo "Usage: $0 [--platform=linux|darwin|all]"
            echo "Env:   TOKENIZERS_VERSION (default: v1.27.0)"
            exit 0
            ;;
    esac
done

BASE_URL="${MEDMEMO_TOKENIZERS_BASE_URL:-https://github.com/daulet/tokenizers/releases/download/${TOKENIZERS_VERSION}}"

download_linux() {
    local out_dir="${PROJECT_ROOT}/resources/lib/linux"
    local archive="libtokenizers.linux-amd64.tar.gz"
    local url="${BASE_URL}/${archive}"

    if [[ -f "${out_dir}/libtokenizers.a" ]]; then
        echo "[linux] libtokenizers.a already exists, skipping"
        return 0
    fi

    echo "[linux] Downloading tokenizers ${TOKENIZERS_VERSION}..."
    mkdir -p "${out_dir}"
    curl -L -o "/tmp/${archive}" "${url}"
    tar -xzf "/tmp/${archive}" -C "${out_dir}"
    rm -f "/tmp/${archive}"
    echo "[linux] Done → ${out_dir}/libtokenizers.a"
}

download_darwin() {
    local out_dir="${PROJECT_ROOT}/resources/lib/darwin"
    local archive="libtokenizers.darwin-x86_64.tar.gz"
    local url="${BASE_URL}/${archive}"

    if [[ -f "${out_dir}/libtokenizers.a" ]]; then
        echo "[darwin] libtokenizers.a already exists, skipping"
        return 0
    fi

    echo "[darwin] Downloading tokenizers ${TOKENIZERS_VERSION}..."
    mkdir -p "${out_dir}"
    curl -L -o "/tmp/${archive}" "${url}"
    tar -xzf "/tmp/${archive}" -C "${out_dir}"
    rm -f "/tmp/${archive}"

    # 同时下载 aarch64 用于 universal 构建
    local archive_arm="libtokenizers.darwin-aarch64.tar.gz"
    local url_arm="${BASE_URL}/${archive_arm}"
    curl -L -o "/tmp/${archive_arm}" "${url_arm}"
    tar -xzf "/tmp/${archive_arm}" -C "/tmp"
    # 使用 lipo 合并（如果可用）
    if command -v lipo &>/dev/null; then
        lipo -create "${out_dir}/libtokenizers.a" "/tmp/libtokenizers.a" -output "${out_dir}/libtokenizers.a" 2>/dev/null || true
        rm -f "/tmp/libtokenizers.a"
    fi
    rm -f "/tmp/${archive_arm}"
    echo "[darwin] Done → ${out_dir}/libtokenizers.a"
}

main() {
    echo "=== Tokenizers Library Downloader ==="
    echo "Version:  ${TOKENIZERS_VERSION}"
    echo "Platform: ${PLATFORM}"
    echo ""

    if ! command -v curl &>/dev/null; then
        echo "Error: 'curl' is required but not found." >&2
        exit 1
    fi

    case "$PLATFORM" in
        linux)   download_linux ;;
        darwin)  download_darwin ;;
        all)
            download_linux
            echo ""
            download_darwin
            ;;
        *)
            echo "Error: Unknown platform '${PLATFORM}'. Use: linux, darwin, all" >&2
            exit 1
            ;;
    esac

    echo ""
    echo "=== All done ==="
}

main
