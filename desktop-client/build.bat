@echo off
chcp 65001 >nul
title AI Agent Desktop Builder
echo ========================================
echo    AI Agent Desktop Builder
echo ========================================
echo.

:: 检查 Node.js
node --version >nul 2>&1
if errorlevel 1 (
    echo 错误：Node.js 未安装
    pause
    exit /b 1
)

echo Node.js 版本: 
node --version
echo.

:: 检查依赖
if not exist "node_modules" (
    echo 正在安装依赖...
    npm install
    echo.
)

:: 构建 Windows 版本
echo 构建 Windows 版本...
npm run build:win

echo.
echo ========================================
echo    构建完成！
echo ========================================
echo.
echo 构建产物位于 dist\ 目录
echo.

:: 列出构建产物
if exist "dist" (
    echo 构建文件列表:
    dir /b dist\
)

pause
