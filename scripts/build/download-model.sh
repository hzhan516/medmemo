#!/bin/bash
# download-model.sh — 下载并导出 DistilBERT NER ONNX 模型
# 用法: ./scripts/build/download-model.sh [--model-id=<id>] [--output-dir=<dir>]
# 环境变量: MODEL_ID, OUTPUT_DIR

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

MODEL_ID="${MODEL_ID:-Davlan/distilbert-base-multilingual-cased-ner-hrl}"
OUTPUT_DIR="${OUTPUT_DIR:-${PROJECT_ROOT}/resources/models/distilbert-ner}"

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Download and export DistilBERT NER model to ONNX format for local inference.

Options:
  --model-id=<id>     Hugging Face model ID (default: ${MODEL_ID})
  --output-dir=<dir>  Output directory (default: ${OUTPUT_DIR})
  -h, --help          Show this help message

Environment variables:
  MODEL_ID            Override default model ID
  OUTPUT_DIR          Override default output directory

Examples:
  $(basename "$0")
  $(basename "$0") --model-id=dslim/bert-base-NER
  MODEL_ID=custom/model $(basename "$0")
EOF
}

# 解析参数
for arg in "$@"; do
    case $arg in
        --model-id=*)
            MODEL_ID="${arg#*=}"
            shift
            ;;
        --output-dir=*)
            OUTPUT_DIR="${arg#*=}"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $arg"
            usage
            exit 1
            ;;
    esac
done

echo "============================================"
echo "  MedMemo Model Download Script"
echo "============================================"
echo "Model ID:   ${MODEL_ID}"
echo "Output:     ${OUTPUT_DIR}"
echo ""

# 幂等检查
if [ -f "${OUTPUT_DIR}/model.onnx" ] && [ -f "${OUTPUT_DIR}/tokenizer.json" ]; then
    echo "Model already exists at ${OUTPUT_DIR}, skipping download."
    if [ -f "${OUTPUT_DIR}/model.onnx.sha256" ]; then
        echo "SHA-256 checksum file already exists."
    else
        echo "Generating SHA-256 checksum..."
        (cd "$(dirname "${OUTPUT_DIR}/model.onnx")" && sha256sum "$(basename "${OUTPUT_DIR}/model.onnx")" > "model.onnx.sha256")
        echo "Checksum saved to ${OUTPUT_DIR}/model.onnx.sha256"
    fi
    exit 0
fi

# 检查 Python3
if ! command -v python3 &>/dev/null; then
    echo "Error: python3 is required to export ONNX model."
    echo "Please install Python 3.8+ and try again."
    exit 1
fi

# 检查 optimum-cli
if ! python3 -c "import optimum" 2>/dev/null; then
    echo "Installing optimum[onnxruntime]..."
    pip3 install -q "optimum[onnxruntime]" || {
        echo "Error: Failed to install optimum. Please run manually:"
        echo "  pip3 install optimum[onnxruntime]"
        exit 1
    }
fi

if ! command -v optimum-cli &>/dev/null; then
    echo "Error: optimum-cli not found in PATH after installation."
    echo "Please ensure your Python scripts directory is in PATH."
    exit 1
fi

# 创建输出目录
mkdir -p "${OUTPUT_DIR}"

echo ""
echo "Exporting ONNX model (this may take 1-3 minutes)..."
echo ""

optimum-cli export onnx \
    --model "${MODEL_ID}" \
    --task token-classification \
    "${OUTPUT_DIR}"

# 检查导出结果
if [ ! -f "${OUTPUT_DIR}/model.onnx" ]; then
    echo "Error: model.onnx not found after export."
    echo "Exported files:"
    ls -la "${OUTPUT_DIR}"
    exit 1
fi

# 生成 SHA-256 校验文件
echo ""
echo "Generating SHA-256 checksum..."
(cd "$(dirname "${OUTPUT_DIR}/model.onnx")" && sha256sum "$(basename "${OUTPUT_DIR}/model.onnx")" > "model.onnx.sha256")

echo ""
echo "============================================"
echo "  Model downloaded successfully!"
echo "============================================"
echo "Location:   ${OUTPUT_DIR}"
echo "Model:      model.onnx"
echo "Checksum:   ${OUTPUT_DIR}/model.onnx.sha256"
echo ""
echo "Files:"
ls -lh "${OUTPUT_DIR}"
echo ""
echo "Next steps:"
echo "  1. Ensure ONNX Runtime library is present: ./scripts/build/download-onnx.sh"
echo "  2. Build the application: go build ./..."
echo "  3. Run tests: go test ./internal/infrastructure/onnx/ -v"
