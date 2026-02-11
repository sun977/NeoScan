#!/bin/bash
#
# NeoScan-Agent macOS 发布脚本
#
# 用法说明：
#   ./release.sh <版本号>
#
# 参数：
#   版本号（必需）   指定版本号，格式：MAJOR.MINOR.PATCH（例如：2.12.0）
#
# 示例：
#   ./release.sh 2.12.0       发布版本 2.12.0
#
# 发布流程：
#   1. 更新 version.go 文件中的版本号
#   2. 提交代码到 Git
#   3. 创建 Git Tag
#   4. 构建所有平台版本
#   5. 提示用户推送代码和 Tag
#
# 作者：NeoScan Team
# 版本：1.0.0
# 日期：2026-02-11
#

set -e

if [ -z "$1" ]; then
    echo "错误：请指定版本号"
    echo "用法：$0 <版本号>"
    echo "示例：$0 2.12.0"
    exit 1
fi

VERSION="$1"
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
VERSION_FILE="$PROJECT_ROOT/internal/pkg/version/version.go"

echo "========================================"
echo "NeoScan-Agent Release Script"
echo "========================================"
echo "Version: $VERSION"
echo "========================================"

if [ ! -f "$VERSION_FILE" ]; then
    echo "错误：版本文件不存在：$VERSION_FILE"
    exit 1
fi

sed -i '' "s/[[:space:]]*Version[[:space:]]*=[[:space:]]*\"[^\"]*\"/Version = \"$VERSION\"/" "$VERSION_FILE"

echo "Step 1/4: Updated version file to $VERSION"

git add internal/pkg/version/version.go
git commit -m "chore: bump version to $VERSION"

if [ $? -ne 0 ]; then
    echo "警告：Git 提交失败或没有可提交的内容"
else
    echo "Step 2/4: Committed version update"
fi

git tag "v$VERSION" -m "Release version $VERSION"

if [ $? -ne 0 ]; then
    echo "错误：创建 Git Tag 失败"
    exit 1
fi

echo "Step 3/4: Created Git tag v$VERSION"

"$BASH_SOURCE" build-all.sh -v "$VERSION"

if [ $? -ne 0 ]; then
    echo "错误：构建失败"
    exit 1
fi

echo "Step 4/4: Built all platforms"

echo "========================================"
echo "Release prepared successfully!"
echo "========================================"
echo "Next steps:"
echo "1. Review changes: git status"
echo "2. Push to remote: git push origin main"
echo "3. Push tag: git push origin v$VERSION"
echo "========================================"
