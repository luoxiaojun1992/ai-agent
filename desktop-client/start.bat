@echo off
chcp 65001 >nul
title AI Agent Desktop Launcher
echo ========================================
echo    AI Agent Desktop Launcher
echo ========================================
echo.

:: 检查 Node.js
node --version >nul 2>&1
if errorlevel 1 (
    echo 错误：Node.js 未安装
    echo 请访问 https://nodejs.org/ 下载并安装 Node.js 16+
    pause
    exit /b 1
)

for /f "tokens=1 delims=v." %%a in ('node --version') do set NODE_MAJOR=%%a
if %NODE_MAJOR% LSS 16 (
    echo 错误：Node.js 版本过低 (需要 16+)
    echo 当前版本: 
    node --version
    pause
    exit /b 1
)

echo [OK] Node.js 版本: 
node --version
echo [OK] npm 版本: 
npm --version
echo.

:: 检查 node_modules
if not exist "node_modules" (
    echo 正在安装依赖...
    echo.
    
    set /p use_mirror="是否使用国内镜像源？(y/n): "
    if /i "%use_mirror%"=="y" (
        npm config set registry https://registry.npmmirror.com
        npm config set electron_mirror https://npmmirror.com/mirrors/electron/
        echo 已设置国内镜像源
    )
    
    npm install
    
    if errorlevel 1 (
        echo.
        echo 安装失败，尝试使用 cnpm...
        npm install -g cnpm --registry=https://registry.npmmirror.com
        cnpm install
    )
    
    echo.
)

:: 启动应用
echo 启动 AI Agent Desktop...
echo.
npm start

pause
