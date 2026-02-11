<#
.SYNOPSIS
    NeoScan-Agent Windows 平台构建脚本

.DESCRIPTION
    用于在 Windows 环境下编译 NeoScan-Agent 的 PowerShell 脚本。
    支持指定目标操作系统和架构，自动注入版本信息。

.PARAMETER TargetOS
    目标操作系统，可选值：windows, linux, darwin
    默认值：windows

.PARAMETER TargetArch
    目标架构，可选值：amd64, arm64
    默认值：amd64

.PARAMETER Output
    输出文件名
    默认值：neoScan-Agent.exe（Windows）或 neoScan-Agent（Linux/macOS）

.PARAMETER Version
    指定版本号，如果不指定则从 version.go 文件读取

.EXAMPLE
    .\build.ps1
    编译当前平台（Windows amd64）

.EXAMPLE
    .\build.ps1 -TargetOS linux -TargetArch amd64
    编译 Linux amd64 版本

.EXAMPLE
    .\build.ps1 -TargetOS darwin -TargetArch arm64
    编译 macOS ARM64 版本（Apple Silicon）

.EXAMPLE
    .\build.ps1 -Version 2.12.0
    使用指定版本号编译

.NOTES
    作者：NeoScan Team
    版本：1.0.0
    日期：2026-02-11
#>

param(
    [string]$TargetOS = "windows",
    [string]$TargetArch = "amd64",
    [string]$Output = "",
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$OutputDir = Join-Path $ProjectRoot "bin"

if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

if ([string]::IsNullOrEmpty($Version)) {
    $Version = (Select-String -Path "$ProjectRoot\internal\pkg\version\version.go" -Pattern "^\s*Version\s*=" | ForEach-Object { $_.Line -replace '.*"([^"]+)".*', '$1' })
}

$BuildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$GitCommit = git rev-parse --short HEAD 2>$null
if ($LASTEXITCODE -ne 0) { $GitCommit = "unknown" }
$GoVersion = (go version | ForEach-Object { $_.Split()[2] })

$BinaryName = "neoScan-Agent"
if ($TargetOS -eq "windows") {
    $BinaryName = "$BinaryName.exe"
}

if ([string]::IsNullOrEmpty($Output)) {
    $Output = Join-Path $OutputDir $BinaryName
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "NeoScan-Agent Build Script" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Version: $Version" -ForegroundColor Green
Write-Host "Target OS: $TargetOS" -ForegroundColor Green
Write-Host "Target Arch: $TargetArch" -ForegroundColor Green
Write-Host "Build Time: $BuildTime" -ForegroundColor Green
Write-Host "Git Commit: $GitCommit" -ForegroundColor Green
Write-Host "Go Version: $GoVersion" -ForegroundColor Green
Write-Host "Output: $Output" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan

$LdFlags = "-w -s -X 'neoagent/internal/pkg/version.BuildTime=$BuildTime' -X 'neoagent/internal/pkg/version.GitCommit=$GitCommit' -X 'neoagent/internal/pkg/version.GoVersion=$GoVersion'"

$Env:GOOS = $TargetOS
$Env:GOARCH = $TargetArch

Push-Location $ProjectRoot
try {
    go build -ldflags="$LdFlags" -o $Output ./cmd/agent

    if ($LASTEXITCODE -eq 0) {
        Write-Host "========================================" -ForegroundColor Cyan
        Write-Host "Build successful!" -ForegroundColor Green
        Write-Host "========================================" -ForegroundColor Cyan
    } else {
        Write-Host "========================================" -ForegroundColor Cyan
        Write-Host "Build failed!" -ForegroundColor Red
        Write-Host "========================================" -ForegroundColor Cyan
        exit 1
    }
} finally {
    Pop-Location
}
