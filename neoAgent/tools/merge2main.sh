#!/bin/bash

# 作者：Sun977
# 用途：用于 dev 分支到 main 分支的代码合并脚本

# 获取当前分支名称
current_branch=$(git branch --show-current)

if [ "$current_branch" == "" ]; then
    echo "错误: 无法获取当前分支，请确保在Git仓库中运行此脚本"
    exit 1
fi

echo "当前分支: $current_branch"

# 检查当前分支是否为dev分支（或以dev开头的分支）
if [[ ! "$current_branch" =~ ^dev ]]; then
    echo "警告: 当前不在dev分支上 ($current_branch)，请确认是否继续？(y/N)"
    read -r confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        echo "操作已取消"
        exit 0
    fi
fi

dev_branch="$current_branch"

# 1. 从dev分支切换到main分支
echo "步骤1: 切换到main分支..."
git checkout main

if [ $? -ne 0 ]; then
    echo "错误: 切换到main分支失败"
    # 尝试切换回原分支
    git checkout "$dev_branch" 2>/dev/null
    exit 1
fi

# 2. 在main分支上合并dev分支代码
echo "步骤2: 在main分支上合并 $dev_branch 分支..."
git merge "$dev_branch"

if [ $? -ne 0 ]; then
    echo "错误: 合并失败，请解决冲突后再重试"
    git checkout "$dev_branch"
    exit 1
fi

# 3. 在main分支上提交到远程仓库
echo "步骤3: 提交main分支到远程仓库..."
git push origin main

if [ $? -ne 0 ]; then
    echo "错误: 推送main分支失败"
    # 回滚合并操作
    git reset --hard HEAD~1
    git checkout "$dev_branch"
    exit 1
fi

# 4. 从main分支切换回dev分支
echo "步骤4: 切换回 $dev_branch 分支..."
git checkout "$dev_branch"

if [ $? -ne 0 ]; then
    echo "错误: 切换回 $dev_branch 分支失败"
    exit 1
fi

echo "成功完成合并流程！"
echo "已将 $dev_branch 的更改合并到main并推送到远程，现在您可以继续在 $dev_branch 上开发"
