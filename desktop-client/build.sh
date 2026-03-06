#!/bin/bash

# AI Agent Desktop 构建脚本

set -e

echo "================================"
echo "  AI Agent Desktop Builder"
echo "================================"
echo ""

# 检测操作系统
OS="unknown"
case "$(uname -s)" in
    Linux*)     OS="linux";;
    Darwin*)    OS="mac";;
    CYGWIN*|MINGW*|MSYS*) OS="win";;
esac

echo "检测到操作系统: $OS"
echo ""

# 检查 Node.js
if ! command -v node &> /dev/null; then
    echo "错误：Node.js 未安装"
    exit 1
fi

echo "Node.js 版本: $(node --version)"
echo ""

# 检查依赖
if [ ! -d "node_modules" ]; then
    echo "正在安装依赖..."
    npm install
    echo ""
fi

# 构建
if [ "$OS" = "win" ]; then
    echo "构建 Windows 版本..."
    npm run build:win
elif [ "$OS" = "mac" ]; then
    echo "构建 macOS 版本..."
    npm run build:mac
else
    echo "构建 Linux 版本..."
    npm run build:linux
fi

echo ""
echo "================================"
echo "  构建完成！"
echo "================================"
echo ""
echo "构建产物位于 dist/ 目录"
echo ""

# 列出构建产物
if [ -d "dist" ]; then
    echo "构建文件列表:"
    ls -lh dist/
fi
