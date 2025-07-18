#!/bin/bash

# 启动 cache-service 的脚本

echo "正在启动 Cache Service..."

# 检查是否安装了 Docker 和 Docker Compose
if ! command -v docker &> /dev/null; then
    echo "错误: Docker 未安装"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "错误: Docker Compose 未安装"
    exit 1
fi

# 构建并启动服务
echo "构建并启动服务..."
docker-compose up --build -d

# 等待服务启动
echo "等待服务启动..."
sleep 10

# 检查服务状态
echo "检查服务状态..."
docker-compose ps

# 检查健康状态
echo "检查 Cache Service 健康状态..."
curl -s http://localhost:8082/health | jq .

echo ""
echo "服务启动完成!"
echo "- Cache Service: http://localhost:8082"
echo "- Redis Commander: http://localhost:8081"
echo "- 健康检查: http://localhost:8082/health"
echo ""
echo "查看日志: docker-compose logs -f cache-service"
echo "停止服务: docker-compose down" 