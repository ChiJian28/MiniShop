#!/bin/bash

# 设置脚本在遇到错误时退出
set -e

echo "正在启动 Order Service..."

# 检查是否安装了 Docker 和 Docker Compose
if ! command -v docker &> /dev/null; then
    echo "错误: Docker 未安装，请先安装 Docker"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "错误: Docker Compose 未安装，请先安装 Docker Compose"
    exit 1
fi

# 创建必要的目录
mkdir -p logs
mkdir -p scripts

# 创建数据库初始化脚本
cat > scripts/init.sql << EOF
-- 创建订单数据库
CREATE DATABASE IF NOT EXISTS order_db;

-- 切换到订单数据库
\c order_db;

-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 设置时区
SET timezone = 'Asia/Shanghai';
EOF

# 停止可能正在运行的容器
echo "停止现有容器..."
docker-compose down --remove-orphans

# 清理旧的镜像（可选）
echo "清理旧镜像..."
docker image prune -f

# 构建并启动服务
echo "构建并启动服务..."
docker-compose up --build -d

# 等待服务启动
echo "等待服务启动..."
sleep 30

# 检查服务状态
echo "检查服务状态..."
docker-compose ps

# 显示日志
echo "显示最近的日志..."
docker-compose logs --tail=50 order-service

echo "Order Service 启动完成!"
echo "服务访问地址:"
echo "  - Order Service API: http://localhost:8082"
echo "  - Health Check: http://localhost:8082/health"
echo "  - PostgreSQL: localhost:5433"
echo "  - Redis: localhost:6380"
echo "  - RabbitMQ Management: http://localhost:15673 (admin/password)"
echo "  - Kafka: localhost:9093"
echo ""
echo "查看实时日志: docker-compose logs -f order-service"
echo "停止服务: docker-compose down" 