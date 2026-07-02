#Requires -Version 5.1
<#
.SYNOPSIS
    收集 Windows 运行时 DLL 到 build/bin，供 NSIS 打包进安装程序。

.DESCRIPTION
    Wails 构建出的 MedMemo.exe 依赖 MinGW 运行时 DLL。本脚本在 wails build 之后运行，
    将已知必需的 DLL 从 MSYS2 mingw64 目录复制到 build/bin，避免用户端出现缺少 DLL 错误。
    不会从 C:\Windows\System32 复制任何系统 DLL。

.EXAMPLE
    .\scripts\build\collect-windows-runtime-dlls.ps1
#>

[CmdletBinding()]
param(
    [string]$BinaryPath = "build/bin/MedMemo.exe",
    [string]$OutputDir = "build/bin",
    [string]$MinGWBinDir = "C:/msys64/mingw64/bin"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$requiredDlls = @(
    "libgcc_s_seh-1.dll",
    "libstdc++-6.dll",
    "libwinpthread-1.dll"
)

if (-not (Test-Path $BinaryPath)) {
    Write-Error "Binary not found: $BinaryPath"
    exit 1
}

if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

if (-not (Test-Path $MinGWBinDir)) {
    Write-Error "MinGW bin directory not found: $MinGWBinDir"
    exit 1
}

$missing = @()
foreach ($dll in $requiredDlls) {
    $source = Join-Path $MinGWBinDir $dll
    $dest = Join-Path $OutputDir $dll

    if (Test-Path $source) {
        Copy-Item -Path $source -Destination $dest -Force
        Write-Host "Copied $dll -> $dest"
    } else {
        $missing += $dll
        Write-Warning "Missing required DLL: $source"
    }
}

if ($missing.Count -gt 0) {
    Write-Error "Failed to collect required runtime DLLs: $($missing -join ', ')"
    exit 1
}

Write-Host "Runtime DLL collection completed successfully."
