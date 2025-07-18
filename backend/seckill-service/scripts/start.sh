#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查 Docker 是否安装
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
}

# 检查端口是否被占用
check_ports() {
    local ports=(8083 6379 5672 15672 8081)
    local occupied_ports=()
    
    for port in "${ports[@]}"; do
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            occupied_ports+=($port)
        fi
    done
    
    if [ ${#occupied_ports[@]} -gt 0 ]; then
        log_warn "The following ports are already in use: ${occupied_ports[*]}"
        log_warn "Please stop the services using these ports or modify the configuration."
        read -p "Do you want to continue? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
}

# 创建必要的目录
create_directories() {
    log_info "Creating necessary directories..."
    mkdir -p logs
    mkdir -p data/redis
    mkdir -p data/rabbitmq
}

# 启动服务
start_services() {
    log_info "Starting seckill service..."
    
    # 检查是否有 docker-compose.yml 文件
    if [ ! -f "docker-compose.yml" ]; then
        log_error "docker-compose.yml not found in current directory"
        exit 1
    fi
    
    # 启动基础服务
    log_info "Starting basic services (Redis, RabbitMQ)..."
    docker-compose up -d redis rabbitmq redis-commander
    
    # 等待服务启动
    log_info "Waiting for services to start..."
    sleep 10
    
    # 启动秒杀服务
    log_info "Starting seckill service..."
    docker-compose up -d seckill-service
    
    # 检查服务状态
    log_info "Checking service status..."
    docker-compose ps
}

# 健康检查
health_check() {
    log_info "Performing health check..."
    
    # 等待服务完全启动
    sleep 5
    
    # 检查 Redis
    if docker-compose exec redis redis-cli ping | grep -q "PONG"; then
        log_info "✓ Redis is healthy"
    else
        log_error "✗ Redis is not responding"
    fi
    
    # 检查 RabbitMQ
    if docker-compose exec rabbitmq rabbitmq-diagnostics ping | grep -q "Ping succeeded"; then
        log_info "✓ RabbitMQ is healthy"
    else
        log_error "✗ RabbitMQ is not responding"
    fi
    
    # 检查秒杀服务
    if curl -s http://localhost:8083/health | grep -q "healthy"; then
        log_info "✓ Seckill service is healthy"
    else
        log_error "✗ Seckill service is not responding"
    fi
}

# 显示服务信息
show_services() {
    log_info "Services are running on the following ports:"
    echo "  - Seckill Service: http://localhost:8083"
    echo "  - Redis Commander: http://localhost:8081"
    echo "  - RabbitMQ Management: http://localhost:15672 (guest/guest)"
    echo ""
    log_info "API Examples:"
    echo "  - Health Check: curl http://localhost:8083/health"
    echo "  - Service Stats: curl http://localhost:8083/api/v1/system/stats"
    echo ""
    log_info "To stop services: docker-compose down"
    log_info "To view logs: docker-compose logs -f seckill-service"
}

# 主函数
main() {
    log_info "Starting Seckill Service..."
    
    # 检查依赖
    check_docker
    check_ports
    
    # 创建目录
    create_directories
    
    # 启动服务
    start_services
    
    # 健康检查
    health_check
    
    # 显示服务信息
    show_services
    
    log_info "Seckill service started successfully!"
}

# 处理参数
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [OPTIONS]"
        echo "Start the seckill service with Docker Compose"
        echo ""
        echo "Options:"
        echo "  --help, -h     Show this help message"
        echo "  --kafka        Start with Kafka instead of RabbitMQ"
        echo "  --monitoring   Start with monitoring services (Prometheus, Grafana)"
        echo "  --dev          Start in development mode"
        exit 0
        ;;
    --kafka)
        log_info "Starting with Kafka..."
        docker-compose --profile kafka up -d
        ;;
    --monitoring)
        log_info "Starting with monitoring services..."
        docker-compose --profile monitoring up -d
        ;;
    --dev)
        log_info "Starting in development mode..."
        docker-compose up -d redis rabbitmq
        log_info "External services started. Run 'go run cmd/main.go' to start the application."
        ;;
    *)
        main
        ;;
esac 