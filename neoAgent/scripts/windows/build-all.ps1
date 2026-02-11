<#
.SYNOPSIS
    NeoScan-Agent Windows 多平台构建脚本

.DESCRIPTION
    在 Windows 环境下为所有目标平台编译 NeoScan-Agent 的 PowerShell 脚本。
    支持编译 Windows、Linux、macOS 的 amd64 和 arm64 架构版本。

.PARAMETER Version
    指定版本号，如果不指定则从 version.go 文件读取

.EXAMPLE
    .\build-all.ps1
    编译所有平台版本（使用 version.go 中的版本号）

.EXAMPLE
    .\build-all.ps1 -Version 2.12.0
    使用指定版本号编译所有平台版本

.NOTES
    作者：NeoScan Team
    版本：1.0.0
    日期：2026-02-11

    输出目录：release/
    输出文件格式：neoScan-Agent-{VERSION}-{OS}-{ARCH}.{EXT}
#>

param(
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$ReleaseDir = Join-Path $ProjectRoot "release"

if (-not (Test-Path $ReleaseDir)) {
    New-Item -ItemType Directory -Path $ReleaseDir | Out-Null
}

if ([string]::IsNullOrEmpty($Version)) {
    $Version = (Select-String -Path "$ProjectRoot\internal\pkg\version\version.go" -Pattern "^Version\s*=" | ForEach-Object { $_.Line -replace '.*"([^"]+)".*', '$1' })
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "NeoScan-Agent Multi-Platform Build" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Version: $Version" -ForegroundColor Green
Write-Host "Release Directory: $ReleaseDir" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan

$Platforms = @(
    @{OS = "windows"; Arch = "amd64"; Output = "neoScan-Agent-$Version-windows-amd64.exe"},
    @{OS = "windows"; Arch = "arm64"; Output = "neoScan-Agent-$Version-windows-arm64.exe"},
    @{OS = "linux"; Arch = "amd64"; Output = "neoScan-Agent-$Version-linux-amd64"},
    @{OS = "linux"; Arch = "arm64"; Output = "neoScan-Agent-$Version-linux-arm64"},
    @{OS = "darwin"; Arch = "amd64"; Output = "neoScan-Agent-$Version-darwin-amd64"},
    @{OS = "darwin"; Arch = "arm64"; Output = "neoScan-Agent-$Version-darwin-arm64"}
)

foreach ($Platform in $Platforms) {
    Write-Host "Building for $($Platform.OS)/$($Platform.Arch)..." -ForegroundColor Yellow

    & "$PSScriptRoot\build.ps1" -TargetOS $Platform.OS -TargetArch $Platform.Arch -Version $Version

    if ($LASTEXITCODE -eq 0) {
        $SourcePath = Join-Path $ProjectRoot "bin"
        if ($Platform.OS -eq "windows") {
            $SourcePath = Join-Path $SourcePath "neoScan-Agent.exe"
        } else {
            $SourcePath = Join-Path $SourcePath "neoScan-Agent"
        }

        $DestPath = Join-Path $ReleaseDir $Platform.Output
        Copy-Item -Path $SourcePath -Destination $DestPath -Force

        Write-Host "Created: $DestPath" -ForegroundColor Green
    } else {
        Write-Host "Failed to build for $($Platform.OS)/$($Platform.Arch)" -ForegroundColor Red
        exit 1
    }
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "All builds completed successfully!" -ForegroundColor Green
Write-Host "Release directory: $ReleaseDir" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
