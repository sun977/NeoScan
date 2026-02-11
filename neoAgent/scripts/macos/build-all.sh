#!/bin/bash
#
# NeoScan-Agent macOS 多平台构建脚本
#
# 用法说明：
#   ./build-all.sh [选项]
#
# 选项：
#   -v, --version <版本号> 指定版本号，如果不指定则从 version.go 读取
#   -h, --help              显示帮助信息
#
# 示例：
#   ./build-all.sh                   编译所有平台版本（使用 version.go 中的版本号）
#   ./build-all.sh -v 2.12.0         使用指定版本号编译所有平台版本
#
# 输出目录：release/
# 输出文件格式：neoAgent-{VERSION}-{OS}-{ARCH}.{EXT}
#
# 作者：NeoScan Team
# 版本：1.0.0
# 日期：2026-02-11
#

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
RELEASE_DIR="$PROJECT_ROOT/release"
VERSION=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -h|--help)
            echo "用法：$0 [选项]"
            echo ""
            echo "选项："
            echo "  -v, --version <版本号> 指定版本号"
            echo "  -h, --help              显示帮助信息"
            echo ""
            echo "示例："
            echo "  $0                        编译所有平台版本"
            echo "  $0 -v 2.12.0            使用指定版本号编译"
            echo ""
            echo "输出目录：$RELEASE_DIR"
            echo "输出文件格式：neoAgent-{VERSION}-{OS}-{ARCH}.{EXT}"
            exit 0
            ;;
        *)
            echo "未知选项：$1"
            echo "使用 -h 或 --help 查看帮助"
            exit 1
            ;;
    esac
done

mkdir -p "$RELEASE_DIR"

if [ -z "$VERSION" ]; then
    VERSION=$(grep -E "^Version\s*=" "$PROJECT_ROOT/internal/pkg/version/version.go" | awk -F'"' '{print $2}')
fi

echo "========================================"
echo "NeoScan-Agent Multi-Platform Build"
echo "========================================"
echo "Version: $VERSION"
echo "Release Directory: $RELEASE_DIR"
echo "========================================"

PLATFORMS=(
    "windows:amd64:neoAgent-$VERSION-windows-amd64.exe"
    "windows:arm64:neoAgent-$VERSION-windows-arm64.exe"
    "linux:amd64:neoAgent-$VERSION-linux-amd64"
    "linux:arm64:neoAgent-$VERSION-linux-arm64"
    "darwin:amd64:neoAgent-$VERSION-darwin-amd64"
    "darwin:arm64:neoAgent-$VERSION-darwin-arm64"
)

for PLATFORM in "${PLATFORMS[@]}"; do
    IFS=':' read -r OS ARCH OUTPUT <<< "$PLATFORM"

    echo "Building for $OS/$ARCH..."

    "$BASH_SOURCE" -o "$OS" -a "$ARCH" -v "$VERSION"

    if [ $? -eq 0 ]; then
        SOURCE_PATH="$PROJECT_ROOT/bin"
        if [ "$OS" = "windows" ]; then
            SOURCE_PATH="$SOURCE_PATH/neoAgent.exe"
        else
            SOURCE_PATH="$SOURCE_PATH/neoAgent"
        fi

        DEST_PATH="$RELEASE_DIR/$OUTPUT"
        cp -f "$SOURCE_PATH" "$DEST_PATH"

        echo "Created: $DEST_PATH"
    else
        echo "Failed to build for $OS/$ARCH"
        exit 1
    fi
done

echo "========================================"
echo "All builds completed successfully!"
echo "Release directory: $RELEASE_DIR"
echo "========================================"
