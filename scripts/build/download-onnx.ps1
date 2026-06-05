# ONNX Runtime 跨平台动态库下载脚本 (PowerShell)
#
# 用法:
#   .\scripts\build\download-onnx.ps1              # 下载全部平台
#   .\scripts\build\download-onnx.ps1 -Platform windows
#   $env:ONNX_VERSION = "1.20.0"; .\scripts\build\download-onnx.ps1
#
# 支持平台: linux, darwin, windows, all (默认)

param(
    [ValidateSet("linux", "darwin", "windows", "all")]
    [string]$Platform = "all"
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Resolve-Path (Join-Path $ScriptDir "..\..")

$OnnxVersion = if ($env:ONNX_VERSION) { $env:ONNX_VERSION } else { "1.21.0" }
$BaseUrl = "https://github.com/microsoft/onnxruntime/releases/download/v${OnnxVersion}"

function Download-Linux {
    $OutDir = Join-Path $ProjectRoot "resources\lib\linux"
    $Archive = "onnxruntime-linux-x64-${OnnxVersion}.tgz"
    $Url = "${BaseUrl}/${Archive}"

    if (Test-Path (Join-Path $OutDir "libonnxruntime.so.1")) {
        Write-Host "[linux] ONNX Runtime already exists, skipping"
        return
    }

    Write-Host "[linux] Downloading ONNX Runtime ${OnnxVersion}..."
    New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
    $TmpArchive = Join-Path $env:TEMP $Archive
    Invoke-WebRequest -Uri $Url -OutFile $TmpArchive -UseBasicParsing

    $TmpExtract = Join-Path $env:TEMP "onnxruntime-linux-x64-${OnnxVersion}"
    if (Get-Command tar -ErrorAction SilentlyContinue) {
        tar -xzf $TmpArchive -C $env:TEMP
    } else {
        throw "[linux] 'tar' not found. Please install tar or use WSL."
    }

    Copy-Item -Path "$TmpExtract\lib\*" -Destination $OutDir -Recurse -Force
    Remove-Item -Path $TmpArchive -Force -ErrorAction SilentlyContinue
    Remove-Item -Path $TmpExtract -Recurse -Force -ErrorAction SilentlyContinue
    Write-Host "[linux] Done → $OutDir"
}

function Download-Darwin {
    $OutDir = Join-Path $ProjectRoot "resources\lib\darwin"
    $Archive = "onnxruntime-osx-universal2-${OnnxVersion}.tgz"
    $Url = "${BaseUrl}/${Archive}"

    if (Test-Path (Join-Path $OutDir "libonnxruntime.dylib")) {
        Write-Host "[darwin] ONNX Runtime already exists, skipping"
        return
    }

    Write-Host "[darwin] Downloading ONNX Runtime ${OnnxVersion}..."
    New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
    $TmpArchive = Join-Path $env:TEMP $Archive
    Invoke-WebRequest -Uri $Url -OutFile $TmpArchive -UseBasicParsing

    $TmpExtract = Join-Path $env:TEMP "onnxruntime-osx-universal2-${OnnxVersion}"
    tar -xzf $TmpArchive -C $env:TEMP

    Copy-Item -Path "$TmpExtract\lib\*" -Destination $OutDir -Recurse -Force
    Remove-Item -Path $TmpArchive -Force -ErrorAction SilentlyContinue
    Remove-Item -Path $TmpExtract -Recurse -Force -ErrorAction SilentlyContinue
    Write-Host "[darwin] Done → $OutDir"
}

function Download-Windows {
    $OutDir = Join-Path $ProjectRoot "resources\lib\windows"
    $Archive = "onnxruntime-win-x64-${OnnxVersion}.zip"
    $Url = "${BaseUrl}/${Archive}"

    if (Test-Path (Join-Path $OutDir "onnxruntime.dll")) {
        Write-Host "[windows] ONNX Runtime already exists, skipping"
        return
    }

    Write-Host "[windows] Downloading ONNX Runtime ${OnnxVersion}..."
    New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
    $TmpArchive = Join-Path $env:TEMP $Archive
    Invoke-WebRequest -Uri $Url -OutFile $TmpArchive -UseBasicParsing

    $TmpExtract = Join-Path $env:TEMP "onnxruntime-win-x64-${OnnxVersion}"
    Expand-Archive -Path $TmpArchive -DestinationPath $env:TEMP -Force

    Copy-Item -Path "$TmpExtract\lib\onnxruntime.dll" -Destination $OutDir -Force
    if (Test-Path "$TmpExtract\lib\onnxruntime_providers_shared.dll") {
        Copy-Item -Path "$TmpExtract\lib\onnxruntime_providers_shared.dll" -Destination $OutDir -Force
    }

    # 复制头文件（CGO 编译 hugot C wrapper 可能需要）
    $IncludeDir = Join-Path $ProjectRoot "resources\include\onnxruntime"
    if (Test-Path "$TmpExtract\include") {
        New-Item -ItemType Directory -Force -Path $IncludeDir | Out-Null
        Copy-Item -Path "$TmpExtract\include\*" -Destination $IncludeDir -Recurse -Force
        Write-Host "[windows] Headers copied to -> $IncludeDir"
    }

    Remove-Item -Path $TmpArchive -Force -ErrorAction SilentlyContinue
    Remove-Item -Path $TmpExtract -Recurse -Force -ErrorAction SilentlyContinue
    Write-Host "[windows] Done → $OutDir"
}

Write-Host "=== ONNX Runtime Downloader ==="
Write-Host "Version:  $OnnxVersion"
Write-Host "Platform: $Platform"
Write-Host ""

switch ($Platform) {
    "linux"   { Download-Linux }
    "darwin"  { Download-Darwin }
    "windows" { Download-Windows }
    "all" {
        Download-Linux
        Write-Host ""
        Download-Darwin
        Write-Host ""
        Download-Windows
    }
}

Write-Host ""
Write-Host "=== All done ==="
