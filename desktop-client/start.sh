#!/bin/bash

# AI Agent Desktop 启动脚本

echo "================================"
echo "  AI Agent Desktop Launcher"
echo "================================"
echo ""

# 检查 Node.js
if ! command -v node &> /dev/null; then
    echo "错误：Node.js 未安装"
    echo "请访问 https://nodejs.org/ 下载并安装 Node.js 16+"
    exit 1
fi

NODE_VERSION=$(node --version | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 16 ]; then
    echo "错误：Node.js 版本过低 (需要 16+)"
    echo "当前版本: $(node --version)"
    exit 1
fi

echo "✓ Node.js 版本: $(node --version)"
echo "✓ npm 版本: $(npm --version)"
echo ""

# 检查 node_modules
if [ ! -d "node_modules" ]; then
    echo "正在安装依赖..."
    echo ""
    
    # 设置镜像源（国内用户）
    read -p "是否使用国内镜像源？(y/n): " use_mirror
    if [ "$use_mirror" = "y" ] || [ "$use_mirror" = "Y" ]; then
        npm config set registry https://registry.npmmirror.com
        npm config set electron_mirror https://npmmirror.com/mirrors/electron/
        echo "已设置国内镜像源"
    fi
    
    npm install
    
    if [ $? -ne 0 ]; then
        echo ""
        echo "安装失败，尝试使用 cnpm..."
        npm install -g cnpm --registry=https://registry.npmmirror.com
        cnpm install
    fi
    
    echo ""
fi

# 启动应用
echo "启动 AI Agent Desktop..."
echo ""
npm start
