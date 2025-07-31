#!/bin/bash

# 设置脚本在遇到错误时退出
set -e

echo "正在启动 MiniShop 秒杀系统..."

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
mkdir -p monitoring
mkdir -p nginx/conf.d

# 解析命令行参数
MONITORING=false
NGINX=false
TOOLS=false
PROFILES=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --monitoring)
      MONITORING=true
      shift
      ;;
    --nginx)
      NGINX=true
      shift
      ;;
    --tools)
      TOOLS=true
      shift
      ;;
    --all)
      MONITORING=true
      NGINX=true
      TOOLS=true
      shift
      ;;
    *)
      echo "未知参数: $1"
      echo "用法: $0 [--monitoring] [--nginx] [--tools] [--all]"
      exit 1
      ;;
  esac
done

# 设置 Docker Compose profiles
if [ "$MONITORING" = true ]; then
    PROFILES="$PROFILES --profile monitoring"
    echo "启用监控组件 (Prometheus + Grafana)"
fi

if [ "$NGINX" = true ]; then
    PROFILES="$PROFILES --profile with-nginx"
    echo "启用 Nginx 负载均衡器"
fi

if [ "$TOOLS" = true ]; then
    PROFILES="$PROFILES --profile tools"
    echo "启用管理工具 (Redis Commander)"
fi

# 创建 Prometheus 配置
if [ "$MONITORING" = true ]; then
    cat > monitoring/prometheus.yml << EOF
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'api-gateway'
    static_configs:
      - targets: ['api-gateway:9090']
    metrics_path: /metrics
    scrape_interval: 10s

  - job_name: 'cache-service'
    static_configs:
      - targets: ['cache-service:8081']
    metrics_path: /metrics
    scrape_interval: 15s

  - job_name: 'seckill-service'
    static_configs:
      - targets: ['seckill-service:8082']
    metrics_path: /metrics
    scrape_interval: 10s

  - job_name: 'order-service'
    static_configs:
      - targets: ['order-service:8084']
    metrics_path: /metrics
    scrape_interval: 15s

  - job_name: 'inventory-service'
    static_configs:
      - targets: ['inventory-service:8083']
    metrics_path: /metrics
    scrape_interval: 15s
EOF
fi

# 创建 Nginx 配置
if [ "$NGINX" = true ]; then
    cat > nginx/nginx.conf << EOF
events {
    worker_connections 1024;
}

http {
    upstream api_gateway {
        server api-gateway:8080;
    }

    server {
        listen 80;
        server_name localhost;

        location / {
            proxy_pass http://api_gateway;
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;
        }
    }
}
EOF
fi

# 停止可能正在运行的容器
echo "停止现有容器..."
docker-compose down --remove-orphans

# 清理旧的镜像（可选）
echo "清理旧镜像..."
docker image prune -f

# 构建并启动服务
echo "构建并启动服务..."
docker-compose up --build -d $PROFILES

# 等待服务启动
echo "等待服务启动..."
sleep 60

# 检查服务状态
echo "检查服务状态..."
docker-compose ps

# 检查 API Gateway 健康状态
echo "检查 API Gateway 健康状态..."
for i in {1..10}; do
    if curl -s http://localhost:8080/health > /dev/null; then
        echo "✅ API Gateway 健康检查通过"
        break
    else
        echo "⏳ 等待 API Gateway 启动... ($i/10)"
        sleep 5
    fi
done

# 显示服务访问地址
echo ""
echo "🎉 MiniShop 秒杀系统启动完成!"
echo ""
echo "服务访问地址:"
echo "  - API Gateway: http://localhost:8080"
echo "  - Health Check: http://localhost:8080/health"
echo "  - Stats: http://localhost:8080/stats"
echo ""
echo "后端服务:"
echo "  - Cache Service: http://localhost:8081"
echo "  - Seckill Service: http://localhost:8082"
echo "  - Inventory Service: http://localhost:8083"
echo "  - Order Service: http://localhost:8084"
echo ""
echo "数据库和中间件:"
echo "  - Redis: localhost:6379"
echo "  - PostgreSQL (Order): localhost:5434"
echo "  - PostgreSQL (Inventory): localhost:5433"
echo "  - RabbitMQ: localhost:5672"
echo "  - RabbitMQ Management: http://localhost:15672 (admin/password)"
echo "  - Kafka: localhost:9092"

if [ "$NGINX" = true ]; then
    echo ""
    echo "负载均衡:"
    echo "  - Nginx: http://localhost:80"
fi

if [ "$MONITORING" = true ]; then
    echo ""
    echo "监控组件:"
    echo "  - Prometheus: http://localhost:9091"
    echo "  - Grafana: http://localhost:3000 (admin/admin)"
fi

if [ "$TOOLS" = true ]; then
    echo ""
    echo "管理工具:"
    echo "  - Redis Commander: http://localhost:8085"
fi

echo ""
echo "API 示例:"
echo "  # 登录获取 token"
echo "  curl -X POST http://localhost:8080/api/v1/auth/login \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"username\":\"admin\",\"password\":\"password\"}'"
echo ""
echo "  # 秒杀接口"
echo "  curl -X POST http://localhost:8080/api/v1/seckill/purchase \\"
echo "    -H 'Authorization: Bearer <token>' \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"product_id\":1001,\"user_id\":1}'"
echo ""
echo "常用命令:"
echo "  查看实时日志: docker-compose logs -f api-gateway"
echo "  停止所有服务: docker-compose down"
echo "  重启服务: docker-compose restart api-gateway" 