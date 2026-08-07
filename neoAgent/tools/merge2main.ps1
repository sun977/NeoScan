# 作者：Sun977
# 用途：用于 dev 分支到 main 分支的代码合并脚本（Windows PowerShell 版本，对应 merge2main.sh）

$ErrorActionPreference = "Stop"

# 获取当前分支名称
$currentBranch = git branch --show-current

if ([string]::IsNullOrEmpty($currentBranch)) {
    Write-Host "错误: 无法获取当前分支，请确保在Git仓库中运行此脚本"
    exit 1
}

Write-Host "当前分支: $currentBranch"

# 检查当前分支是否为dev分支（或以dev开头的分支）
if ($currentBranch -notmatch '^dev') {
    Write-Host "警告: 当前不在dev分支上 ($currentBranch)，请确认是否继续？(y/N)"
    $confirm = Read-Host
    if ($confirm -notmatch '^[Yy]$') {
        Write-Host "操作已取消"
        exit 0
    }
}

$devBranch = $currentBranch

# 1. 从dev分支切换到main分支
Write-Host "步骤1: 切换到main分支..."
git checkout main

if ($LASTEXITCODE -ne 0) {
    Write-Host "错误: 切换到main分支失败"
    # 尝试切换回原分支
    git checkout $devBranch 2>$null
    exit 1
}

# 2. 在main分支上合并dev分支代码
Write-Host "步骤2: 在main分支上合并 $devBranch 分支..."
git merge $devBranch

if ($LASTEXITCODE -ne 0) {
    Write-Host "错误: 合并失败，请解决冲突后再重试"
    git checkout $devBranch
    exit 1
}

# 3. 在main分支上提交到远程仓库
Write-Host "步骤3: 提交main分支到远程仓库..."
git push origin main

if ($LASTEXITCODE -ne 0) {
    Write-Host "错误: 推送main分支失败"
    # 回滚合并操作
    git reset --hard HEAD~1
    git checkout $devBranch
    exit 1
}

# 4. 从main分支切换回dev分支
Write-Host "步骤4: 切换回 $devBranch 分支..."
git checkout $devBranch

if ($LASTEXITCODE -ne 0) {
    Write-Host "错误: 切换回 $devBranch 分支失败"
    exit 1
}

Write-Host "成功完成合并流程！"
Write-Host "已将 $devBranch 的更改合并到main并推送到远程，现在您可以继续在 $devBranch 上开发"
