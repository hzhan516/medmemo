# download-model.ps1 — 下载并导出 DistilBERT NER ONNX 模型 (Windows)
# 用法: .\scripts\build\download-model.ps1 [-ModelId <id>] [-OutputDir <dir>]
# 环境变量: MODEL_ID, OUTPUT_DIR

param(
    [string]$ModelId = $env:MODEL_ID,
    [string]$OutputDir = $env:OUTPUT_DIR
)

# 默认值
if (-not $ModelId) { $ModelId = "Davlan/distilbert-base-multilingual-cased-ner-hrl" }
if (-not $OutputDir) {
    $ProjectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
    $OutputDir = Join-Path $ProjectRoot "resources\models\distilbert-ner"
}

function Show-Usage {
    Write-Host @"
Usage: $(Split-Path -Leaf $PSCommandPath) [OPTIONS]

Download and export DistilBERT NER model to ONNX format for local inference.

Options:
  -ModelId <id>     Hugging Face model ID (default: $ModelId)
  -OutputDir <dir>  Output directory (default: $OutputDir)

Environment variables:
  MODEL_ID            Override default model ID
  OUTPUT_DIR          Override default output directory

Examples:
  .\$(Split-Path -Leaf $PSCommandPath)
  .\$(Split-Path -Leaf $PSCommandPath) -ModelId dslim/bert-base-NER
"@
}

if ($args -contains "-h" -or $args -contains "--help") {
    Show-Usage
    exit 0
}

Write-Host "============================================"
Write-Host "  MedMemo Model Download Script"
Write-Host "============================================"
Write-Host "Model ID:   $ModelId"
Write-Host "Output:     $OutputDir"
Write-Host ""

# 幂等检查
$ModelFile = Join-Path $OutputDir "model.onnx"
$TokenizerFile = Join-Path $OutputDir "tokenizer.json"
$ShaFile = Join-Path $OutputDir "model.onnx.sha256"

if ((Test-Path $ModelFile) -and (Test-Path $TokenizerFile)) {
    Write-Host "Model already exists at $OutputDir, skipping download."
    if (Test-Path $ShaFile) {
        Write-Host "SHA-256 checksum file already exists."
    } else {
        Write-Host "Generating SHA-256 checksum..."
        $Hash = Get-FileHash -Path $ModelFile -Algorithm SHA256
        $Hash.Hash | Out-File -FilePath $ShaFile -Encoding utf8
        Write-Host "Checksum saved to $ShaFile"
    }
    exit 0
}

# 检查 Python3
$Python = Get-Command python3 -ErrorAction SilentlyContinue
if (-not $Python) {
    $Python = Get-Command python -ErrorAction SilentlyContinue
}
if (-not $Python) {
    Write-Host "Error: python3 or python is required to export ONNX model."
    Write-Host "Please install Python 3.8+ and try again."
    exit 1
}

# 检查 optimum
$OptimumCheck = & $Python.Source -c "import optimum" 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "Installing optimum[onnxruntime]..."
    & $Python.Source -m pip install -q "optimum[onnxruntime]"
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Error: Failed to install optimum. Please run manually:"
        Write-Host "  pip3 install optimum[onnxruntime]"
        exit 1
    }
}

# 检查 optimum-cli
$OptimumCli = Get-Command optimum-cli -ErrorAction SilentlyContinue
if (-not $OptimumCli) {
    # 尝试通过 python -m 调用
    $OptimumModuleCheck = & $Python.Source -m optimum.exporters.onnx --help 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Error: optimum-cli not found in PATH after installation."
        Write-Host "Please ensure your Python scripts directory is in PATH."
        exit 1
    }
}

# 创建输出目录
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

Write-Host ""
Write-Host "Exporting ONNX model (this may take 1-3 minutes)..."
Write-Host ""

& $Python.Source -m optimum.exporters.onnx `
    --model $ModelId `
    --task token-classification `
    $OutputDir

if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: ONNX export failed."
    exit 1
}

# 检查导出结果
if (-not (Test-Path $ModelFile)) {
    Write-Host "Error: model.onnx not found after export."
    Write-Host "Exported files:"
    Get-ChildItem $OutputDir
    exit 1
}

# 生成 SHA-256 校验文件
Write-Host ""
Write-Host "Generating SHA-256 checksum..."
$Hash = Get-FileHash -Path $ModelFile -Algorithm SHA256
$Hash.Hash | Out-File -FilePath $ShaFile -Encoding utf8

Write-Host ""
Write-Host "============================================"
Write-Host "  Model downloaded successfully!"
Write-Host "============================================"
Write-Host "Location:   $OutputDir"
Write-Host "Model:      model.onnx"
Write-Host "Checksum:   $ShaFile"
Write-Host ""
Write-Host "Files:"
Get-ChildItem $OutputDir | Select-Object Name, @{N="Size";E={"{0:N1} MB" -f ($_.Length / 1MB)}}
Write-Host ""
Write-Host "Next steps:"
Write-Host "  1. Ensure ONNX Runtime library is present: .\scripts\build\download-onnx.ps1"
Write-Host "  2. Build the application: go build ./..."
Write-Host "  3. Run tests: go test ./internal/infrastructure/onnx/ -v"
