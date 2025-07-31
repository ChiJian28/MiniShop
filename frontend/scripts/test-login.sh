#!/bin/bash

echo "🔐 MiniShop 登录功能测试"
echo "=========================="

# 检查 Node.js 和 npm
if ! command -v node &> /dev/null; then
    echo "❌ Node.js 未安装，请先安装 Node.js"
    exit 1
fi

if ! command -v npm &> /dev/null; then
    echo "❌ npm 未安装，请先安装 npm"
    exit 1
fi

echo "✅ Node.js 版本: $(node --version)"
echo "✅ npm 版本: $(npm --version)"
echo ""

# 检查依赖
echo "📦 检查依赖..."
if [ ! -d "node_modules" ]; then
    echo "📥 安装依赖..."
    npm install
fi

# 检查关键依赖
if [ ! -d "node_modules/react-router-dom" ]; then
    echo "📥 安装 React Router..."
    npm install react-router-dom@^6.20.0 @types/react-router-dom
fi

echo "✅ 依赖检查完成"
echo ""

# 显示测试信息
echo "🧪 登录功能测试指南"
echo "==================="
echo ""
echo "1. 📱 React 应用测试："
echo "   - 访问: http://localhost:3000"
echo "   - 默认会重定向到 /login（如果未登录）"
echo "   - 或重定向到 /seckill（如果已登录）"
echo ""
echo "2. 🔗 路由测试："
echo "   - http://localhost:3000/login - 登录页面"
echo "   - http://localhost:3000/seckill - 秒杀页面（需要登录）"
echo "   - http://localhost:3000/ - 根路径（重定向）"
echo ""
echo "3. 🧪 静态测试页面："
echo "   - 打开 test-login.html 进行功能测试"
echo ""
echo "4. 👤 测试账号："
echo "   - 演示账号: admin / password"
echo "   - 任意账号: 任何非空用户名密码都可以登录"
echo ""
echo "5. ✨ 测试功能："
echo "   - ✅ 表单验证（空值、长度检查）"
echo "   - ✅ 登录状态管理（Context API）"
echo "   - ✅ 路由保护（Protected Routes）"
echo "   - ✅ Token 持久化（localStorage）"
echo "   - ✅ 自动跳转（登录后跳转到秒杀页）"
echo "   - ✅ Header 状态更新（显示用户名和退出按钮）"
echo ""

# 启动开发服务器
echo "🚀 启动开发服务器..."
echo "   - 前端: http://localhost:3000"
echo "   - 测试页面: ./test-login.html"
echo ""
echo "按 Ctrl+C 停止服务器"
echo ""

# 检查端口是否被占用
if lsof -Pi :3000 -sTCP:LISTEN -t >/dev/null ; then
    echo "⚠️  端口 3000 已被占用，尝试使用其他端口..."
    PORT=3001 npm start
else
    npm start
fi 