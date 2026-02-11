#!/bin/bash
#
# NeoScan-Agent macOS 平台构建脚本
#
# 用法说明：
#   ./build.sh [选项]
#
# 选项：
#   -o, --os <操作系统>    目标操作系统（windows, linux, darwin），默认：darwin
#   -a, --arch <架构>      目标架构（amd64, arm64），默认：amd64
#   -v, --version <版本号> 指定版本号，如果不指定则从 version.go 读取
#   -h, --help              显示帮助信息
#
# 示例：
#   ./build.sh                          编译当前平台（macOS amd64）
#   ./build.sh -o windows -a amd64      编译 Windows amd64 版本
#   ./build.sh -o linux -a arm64        编译 Linux ARM64 版本
#   ./build.sh -v 2.12.0              使用指定版本号编译
#
# 作者：NeoScan Team
# 版本：1.0.0
# 日期：2026-02-11
#

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
OUTPUT_DIR="$PROJECT_ROOT/bin"
VERSION=""
TARGET_OS="darwin"
TARGET_ARCH="amd64"
OUTPUT=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -o|--os)
            TARGET_OS="$2"
            shift 2
            ;;
        -a|--arch)
            TARGET_ARCH="$2"
            shift 2
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -h|--help)
            echo "用法：$0 [选项]"
            echo ""
            echo "选项："
            echo "  -o, --os <操作系统>    目标操作系统（windows, linux, darwin），默认：darwin"
            echo "  -a, --arch <架构>      目标架构（amd64, arm64），默认：amd64"
            echo "  -v, --version <版本号> 指定版本号"
            echo "  -h, --help              显示帮助信息"
            echo ""
            echo "示例："
            echo "  $0                          编译当前平台（macOS amd64）"
            echo "  $0 -o windows -a amd64      编译 Windows amd64 版本"
            echo "  $0 -o linux -a arm64        编译 Linux ARM64 版本"
            echo "  $0 -v 2.12.0              使用指定版本号编译"
            exit 0
            ;;
        *)
            echo "未知选项：$1"
            echo "使用 -h 或 --help 查看帮助"
            exit 1
            ;;
    esac
done

mkdir -p "$OUTPUT_DIR"

if [ -z "$VERSION" ]; then
    VERSION=$(grep -E "^\s*Version\s*=" "$PROJECT_ROOT/internal/pkg/version/version.go" | awk -F'"' '{print $2}')
fi

BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION=$(go version | awk '{print $3}')

BINARY_NAME="neoScan-Agent"
if [ "$TARGET_OS" = "windows" ]; then
    BINARY_NAME="$BINARY_NAME.exe"
fi

if [ -z "$OUTPUT" ]; then
    OUTPUT="$OUTPUT_DIR/$BINARY_NAME"
fi

echo "========================================"
echo "NeoScan-Agent Build Script"
echo "========================================"
echo "Version: $VERSION"
echo "Target OS: $TARGET_OS"
echo "Target Arch: $TARGET_ARCH"
echo "Build Time: $BUILD_TIME"
echo "Git Commit: $GIT_COMMIT"
echo "Go Version: $GO_VERSION"
echo "Output: $OUTPUT"
echo "========================================"

LDFLAGS="-w -s -X 'neoagent/internal/pkg/version.BuildTime=$BUILD_TIME' -X 'neoagent/internal/pkg/version.GitCommit=$GIT_COMMIT' -X 'neoagent/internal/pkg/version.GoVersion=$GO_VERSION'"

export GOOS=$TARGET_OS
export GOARCH=$TARGET_ARCH

cd "$PROJECT_ROOT"
go build -ldflags="$LDFLAGS" -o "$OUTPUT" ./cmd/agent

if [ $? -eq 0 ]; then
    echo "========================================"
    echo "Build successful!"
    echo "========================================"
else
    echo "========================================"
    echo "Build failed!"
    echo "========================================"
    exit 1
fi
