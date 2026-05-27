# daulet/tokenizers Windows 静态库下载脚本 (PowerShell)
#
# 用法:
#   .\scripts\build\download-tokenizers.ps1              # 下载默认版本
#   $env:TOKENIZERS_VERSION = "v1.27.0"; .\scripts\build\download-tokenizers.ps1
#
# 注意: 官方 release 不包含 Windows 预编译库。本脚本从 MedMemo 的 GitHub release
# 下载由 CI 交叉编译的 Windows 静态库（MinGW x86_64-pc-windows-gnu）。
# 如果本地 release 不存在，会尝试从 Hugging Face 等镜像下载。

param(
    [string]$Version = "",
    [string]$BaseUrl = ""
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Resolve-Path (Join-Path $ScriptDir "..\..")
$OutDir = Join-Path $ProjectRoot "resources\lib\windows"

# 默认版本
if (-not $Version) {
    $Version = if ($env:TOKENIZERS_VERSION) { $env:TOKENIZERS_VERSION } else { "v1.27.0" }
}

# 默认下载源: MedMemo 的 GitHub release asset
# 格式: https://github.com/hzhan516/medmemo/releases/download/tokenizers-<version>/libtokenizers.windows-amd64.tar.gz
if (-not $BaseUrl) {
    $BaseUrl = if ($env:TOKENIZERS_BASE_URL) { $env:TOKENIZERS_BASE_URL } else { "https://github.com/hzhan516/medmemo/releases/download" }
}

$ArchiveName = "libtokenizers.windows-amd64.tar.gz"
$ReleaseTag = "tokenizers-$Version"
$Url = "$BaseUrl/$ReleaseTag/$ArchiveName"

# 检查是否已存在
if (Test-Path (Join-Path $OutDir "libtokenizers.a")) {
    Write-Host "[tokenizers] libtokenizers.a already exists, skipping"
    exit 0
}

Write-Host "=== Tokenizers Windows Library Downloader ==="
Write-Host "Version:  $Version"
Write-Host "URL:      $Url"
Write-Host "Output:   $OutDir"
Write-Host ""

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$TmpArchive = Join-Path $env:TEMP $ArchiveName

# 下载
try {
    Write-Host "[tokenizers] Downloading Windows tokenizers library..."
    Invoke-WebRequest -Uri $Url -OutFile $TmpArchive -UseBasicParsing
} catch {
    Write-Error "[tokenizers] Download failed: $_`n`nPossible reasons:`n1. The CI-built Windows library has not been published yet.`n2. Network issue or GitHub rate limit.`n`nYou can manually build the library from source (requires Rust + MinGW):`n  git clone https://github.com/daulet/tokenizers.git`n  cd tokenizers/crates/tokenizers`n  cargo build --release --target x86_64-pc-windows-gnu`n  # Copy target/x86_64-pc-windows-gnu/release/libtokenizers_ffi.a to resources/lib/windows/libtokenizers.a"
    exit 1
}

# 解压 (PowerShell 5.1+ 没有原生 tar.gz 支持，需要 tar 命令)
if (-not (Get-Command tar -ErrorAction SilentlyContinue)) {
    throw "[tokenizers] 'tar' command not found. Required to extract .tar.gz archive."
}

Write-Host "[tokenizers] Extracting archive..."
tar -xzf $TmpArchive -C $OutDir

# 清理
Remove-Item -Path $TmpArchive -Force -ErrorAction SilentlyContinue

# 验证
if (Test-Path (Join-Path $OutDir "libtokenizers.a")) {
    $Size = (Get-Item (Join-Path $OutDir "libtokenizers.a")).Length
    Write-Host "[tokenizers] Done → $OutDir\libtokenizers.a ($Size bytes)"
} else {
    throw "[tokenizers] Extraction failed: libtokenizers.a not found in output directory"
}

Write-Host ""
Write-Host "=== All done ==="
