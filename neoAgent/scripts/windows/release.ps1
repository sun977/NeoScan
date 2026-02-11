<#
.SYNOPSIS
    NeoScan-Agent Windows 发布脚本

.DESCRIPTION
    用于发布新版本的 PowerShell 脚本。
    自动更新版本号、创建 Git Tag、构建所有平台版本。

.PARAMETER Version
    指定版本号（必需）
    格式：MAJOR.MINOR.PATCH（例如：2.12.0）

.EXAMPLE
    .\release.ps1 -Version 2.12.0
    发布版本 2.12.0

.NOTES
    作者：NeoScan Team
    版本：1.0.0
    日期：2026-02-11

    发布流程：
    1. 更新 version.go 文件中的版本号
    2. 提交代码到 Git
    3. 创建 Git Tag
    4. 构建所有平台版本
    5. 提示用户推送代码和 Tag
#>

param(
    [Parameter(Mandatory=$true)]
    [string]$Version
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "NeoScan-Agent Release Script" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Version: $Version" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan

$VersionFile = Join-Path $ProjectRoot "internal\pkg\version\version.go"

if (-not (Test-Path $VersionFile)) {
    Write-Host "Error: Version file not found: $VersionFile" -ForegroundColor Red
    exit 1
}

$Content = Get-Content $VersionFile -Raw
$Content = $Content -replace 'Version\s*=\s*"[^"]*"', "Version = `"$Version`""
Set-Content -Path $VersionFile -Value $Content -NoNewline

Write-Host "Step 1/4: Updated version file to $Version" -ForegroundColor Green

git add internal/pkg/version/version.go
git commit -m "chore: bump version to $Version"

if ($LASTEXITCODE -ne 0) {
    Write-Host "Warning: Git commit failed or nothing to commit" -ForegroundColor Yellow
} else {
    Write-Host "Step 2/4: Committed version update" -ForegroundColor Green
}

git tag "v$Version" -m "Release version $Version"

if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Failed to create Git tag" -ForegroundColor Red
    exit 1
}

Write-Host "Step 3/4: Created Git tag v$Version" -ForegroundColor Green

& "$PSScriptRoot\build-all.ps1" -Version $Version

if ($LASTEXITCODE -ne 0) {
    Write-Host "Error: Build failed" -ForegroundColor Red
    exit 1
}

Write-Host "Step 4/4: Built all platforms" -ForegroundColor Green

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Release prepared successfully!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "1. Review changes: git status" -ForegroundColor White
Write-Host "2. Push to remote: git push origin main" -ForegroundColor White
Write-Host "3. Push tag: git push origin v$Version" -ForegroundColor White
Write-Host "========================================" -ForegroundColor Cyan
