#!/bin/bash

# 设置脚本在遇到错误时退出
set -e

echo "正在启动 MiniShop 前端应用..."

# 检查是否安装了 Node.js 和 npm
if ! command -v node &> /dev/null; then
    echo "错误: Node.js 未安装，请先安装 Node.js"
    exit 1
fi

if ! command -v npm &> /dev/null; then
    echo "错误: npm 未安装，请先安装 npm"
    exit 1
fi

# 检查是否在正确的目录
if [ ! -f "package.json" ]; then
    echo "错误: 未找到 package.json 文件，请确保在前端项目根目录运行此脚本"
    exit 1
fi

# 安装依赖
echo "正在安装依赖..."
npm install

# 启动开发服务器
echo "正在启动开发服务器..."
echo ""
echo "🚀 前端应用将在以下地址启动:"
echo "   本地: http://localhost:3000"
echo "   网络: http://$(hostname -I | awk '{print $1}'):3000"
echo ""
echo "💡 提示:"
echo "   - 确保后端 API Gateway 在 http://localhost:8080 运行"
echo "   - 前端会自动代理 API 请求到后端"
echo "   - 支持热重载，修改代码会自动刷新页面"
echo ""

# 启动应用
npm start 