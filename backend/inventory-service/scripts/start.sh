#!/bin/bash

# 设置脚本在遇到错误时退出
set -e

echo "正在启动 Inventory Service..."

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
mkdir -p monitoring

# 创建数据库初始化脚本
cat > scripts/init.sql << EOF
-- 创建库存数据库
CREATE DATABASE IF NOT EXISTS inventory_db;

-- 切换到库存数据库
\c inventory_db;

-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 设置时区
SET timezone = 'Asia/Shanghai';
EOF

# 创建 Prometheus 配置
mkdir -p monitoring
cat > monitoring/prometheus.yml << EOF
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'inventory-service'
    static_configs:
      - targets: ['inventory-service:8084']
    metrics_path: /metrics
    scrape_interval: 10s
EOF

# 解析命令行参数
MONITORING=false
PROFILES=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --monitoring)
      MONITORING=true
      shift
      ;;
    *)
      echo "未知参数: $1"
      exit 1
      ;;
  esac
done

# 设置 Docker Compose profiles
if [ "\$MONITORING" = true ]; then
    PROFILES="--profile monitoring"
    echo "启用监控组件 (Prometheus + Grafana)"
fi

# 停止可能正在运行的容器
echo "停止现有容器..."
docker-compose down --remove-orphans

# 清理旧的镜像（可选）
echo "清理旧镜像..."
docker image prune -f

# 构建并启动服务
echo "构建并启动服务..."
docker-compose up --build -d \$PROFILES

# 等待服务启动
echo "等待服务启动..."
sleep 30

# 检查服务状态
echo "检查服务状态..."
docker-compose ps

# 检查服务健康状态
echo "检查服务健康状态..."
for i in {1..10}; do
    if curl -s http://localhost:8083/health > /dev/null; then
        echo "✅ Inventory Service 健康检查通过"
        break
    else
        echo "⏳ 等待 Inventory Service 启动... (\$i/10)"
        sleep 3
    fi
done

# 显示日志
echo "显示最近的日志..."
docker-compose logs --tail=50 inventory-service

echo "Inventory Service 启动完成!"
echo "服务访问地址:"
echo "  - Inventory Service API: http://localhost:8083"
echo "  - Health Check: http://localhost:8083/health"
echo "  - PostgreSQL: localhost:5434"
echo "  - Redis: localhost:6381"
echo "  - Redis Commander: http://localhost:8082"

if [ "\$MONITORING" = true ]; then
    echo "  - Prometheus: http://localhost:9091"
    echo "  - Grafana: http://localhost:3001 (admin/admin)"
fi

echo ""
echo "常用命令:"
echo "  查看实时日志: docker-compose logs -f inventory-service"
echo "  停止服务: docker-compose down"
echo "  重启服务: docker-compose restart inventory-service" 