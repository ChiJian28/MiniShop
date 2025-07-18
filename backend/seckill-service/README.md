# Seckill Service

基于 Go 语言实现的高性能秒杀服务，专为大规模并发场景设计，具备完善的流控、降级和消息队列功能。

## 🚀 功能特性

### 核心功能
- **原子性库存扣减**：使用 Redis Lua 脚本确保库存操作的原子性
- **用户去重**：防止用户重复购买，避免超卖问题
- **消息队列**：支持 RabbitMQ 和 Kafka，异步处理订单创建
- **流控降级**：多级限流、熔断器、请求队列等保护机制

### 高级特性
- **多种限流算法**：令牌桶、滑动窗口、固定窗口
- **熔断器**：自动故障检测和恢复
- **请求队列**：高并发下的排队处理
- **系统监控**：实时统计和健康检查
- **分布式锁**：基于 Redis 的分布式锁实现

## 🏗️ 技术栈

- **Go 1.21**：主要编程语言
- **Gin**：HTTP 框架，提供 RESTful API
- **Redis**：缓存和分布式锁
- **RabbitMQ/Kafka**：消息队列
- **Docker**：容器化部署
- **Prometheus + Grafana**：监控告警

## 📁 项目结构

```
seckill-service/
├── cmd/
│   └── main.go                     # 程序入口
├── internal/
│   ├── config/                     # 配置管理
│   │   └── config.go
│   ├── seckill/                    # 秒杀核心逻辑
│   │   ├── lua_scripts.go          # Lua 脚本
│   │   └── seckill_core.go         # 核心业务逻辑
│   ├── mq/                         # 消息队列
│   │   ├── message.go              # 消息定义
│   │   ├── rabbitmq.go             # RabbitMQ 实现
│   │   └── kafka.go                # Kafka 实现
│   ├── flowcontrol/                # 流控组件
│   │   ├── limiter.go              # 限流器
│   │   ├── circuit_breaker.go      # 熔断器
│   │   └── queue.go                # 请求队列
│   └── service/                    # 服务层
│       └── seckill_service.go      # 主服务
├── api/
│   └── rest/                       # REST API
│       ├── handler.go              # 请求处理
│       └── router.go               # 路由配置
├── config/
│   └── config.yaml                 # 配置文件
├── docker-compose.yml              # Docker Compose 配置
├── Dockerfile                      # Docker 构建文件
└── README.md                       # 项目说明
```

## ⚙️ 配置说明

### 主要配置项

```yaml
server:
  port: 8083                        # HTTP 服务端口
  grpc_port: 9083                   # gRPC 服务端口

redis:
  host: localhost                   # Redis 主机
  port: 6379                        # Redis 端口
  pool_size: 20                     # 连接池大小

seckill:
  max_concurrent_requests: 1000     # 最大并发请求数
  queue_size: 5000                  # 排队队列大小
  
  rate_limit:
    requests_per_second: 500        # 每秒请求数限制
    burst_size: 1000                # 突发请求数
  
  circuit_breaker:
    failure_threshold: 10           # 失败阈值
    recovery_timeout: 60s           # 恢复超时时间
  
  degradation:
    enable: true                    # 是否启用降级
    threshold: 0.8                  # 降级阈值
    response_message: "系统繁忙，请稍后重试"
```

## 🔧 API 接口

### 秒杀相关

#### 同步秒杀
```http
POST /api/v1/seckill/purchase
Content-Type: application/json

{
  "product_id": 1001,
  "user_id": 2001,
  "quantity": 1
}
```

#### 异步秒杀
```http
POST /api/v1/seckill/purchase/async
Content-Type: application/json

{
  "product_id": 1001,
  "user_id": 2001,
  "quantity": 1
}
```

#### 预热活动
```http
POST /api/v1/seckill/activity/prewarm
Content-Type: application/json

{
  "product_id": 1001,
  "product_name": "iPhone 15 Pro",
  "price": 8999.00,
  "stock": 100,
  "start_time": "2024-01-01T10:00:00Z",
  "end_time": "2024-01-01T12:00:00Z",
  "status": "active"
}
```

#### 获取统计信息
```http
GET /api/v1/seckill/stats/{productId}
```

#### 检查用户购买状态
```http
GET /api/v1/seckill/purchased/{productId}/{userId}
```

### 系统监控

#### 服务统计
```http
GET /api/v1/system/stats
```

#### 健康检查
```http
GET /health
```

## 🚀 快速开始

### 使用 Docker Compose（推荐）

1. **启动所有服务**
```bash
docker-compose up -d
```

2. **仅启动基础服务**
```bash
docker-compose up -d seckill-service redis rabbitmq
```

3. **启动 Kafka 版本**
```bash
docker-compose --profile kafka up -d
```

4. **启动监控服务**
```bash
docker-compose --profile monitoring up -d
```

### 本地开发

1. **安装依赖**
```bash
go mod download
```

2. **启动 Redis 和 RabbitMQ**
```bash
docker-compose up -d redis rabbitmq
```

3. **运行服务**
```bash
go run cmd/main.go
```

## 📊 监控和管理

### 服务端口
- **Seckill Service**: http://localhost:8083
- **Redis Commander**: http://localhost:8081
- **RabbitMQ Management**: http://localhost:15672 (guest/guest)
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)

### 健康检查
```bash
curl http://localhost:8083/health
```

### 获取服务统计
```bash
curl http://localhost:8083/api/v1/system/stats
```

## 🔥 核心算法

### 1. 原子性库存扣减 Lua 脚本
```lua
-- 检查库存
local current_stock = redis.call('GET', stock_key)
if current_stock < quantity then
    return -2  -- 库存不足
end

-- 检查用户是否已购买
local user_bought = redis.call('SISMEMBER', users_key, user_id)
if user_bought == 1 then
    return -3  -- 用户已购买
end

-- 扣减库存并记录用户
local new_stock = current_stock - quantity
redis.call('SET', stock_key, new_stock)
redis.call('SADD', users_key, user_id)

return {1, new_stock}  -- 成功
```

### 2. 多级流控策略
- **限流器**：令牌桶算法，平滑处理突发流量
- **熔断器**：自动检测故障并快速失败
- **请求队列**：高并发下的排队处理
- **系统降级**：基于系统负载的自动降级

### 3. 消息队列异步处理
- **订单创建**：秒杀成功后异步创建订单
- **库存同步**：实时同步库存变化
- **用户通知**：异步发送购买结果通知

## 🧪 测试

### 压力测试
```bash
# 使用 wrk 进行压力测试
wrk -t12 -c400 -d30s --script=test/seckill.lua http://localhost:8083/api/v1/seckill/purchase
```

### 功能测试
```bash
# 预热活动
curl -X POST http://localhost:8083/api/v1/seckill/activity/prewarm \
  -H "Content-Type: application/json" \
  -d '{"product_id":1001,"product_name":"Test Product","price":99.99,"stock":100,"start_time":"2024-01-01T10:00:00Z","end_time":"2024-01-01T12:00:00Z","status":"active"}'

# 秒杀请求
curl -X POST http://localhost:8083/api/v1/seckill/purchase \
  -H "Content-Type: application/json" \
  -d '{"product_id":1001,"user_id":2001,"quantity":1}'
```

## 🔧 性能优化

### 1. Redis 优化
- 使用连接池减少连接开销
- Lua 脚本减少网络往返
- 合理设置过期时间

### 2. 消息队列优化
- 批量发送消息
- 消息持久化配置
- 消费者并发处理

### 3. 服务优化
- 协程池管理
- 内存复用
- 热点数据缓存

## 📈 监控指标

### 业务指标
- 秒杀成功率
- 平均响应时间
- 并发用户数
- 库存准确性

### 系统指标
- CPU 使用率
- 内存使用率
- 网络 I/O
- Redis 连接数

### 自定义指标
- 限流触发次数
- 熔断器状态
- 队列长度
- 消息处理延迟

## 🚨 故障处理

### 常见问题
1. **Redis 连接失败**：检查 Redis 服务状态和网络连接
2. **消息队列堆积**：增加消费者数量或优化处理逻辑
3. **库存不一致**：检查 Lua 脚本执行和事务处理
4. **服务响应慢**：检查限流配置和系统资源

### 应急处理
1. **启用降级模式**：返回系统繁忙提示
2. **增加限流强度**：降低并发请求数
3. **重启服务**：清理异常状态
4. **数据修复**：手动修复库存数据

## 📚 扩展功能

- **分布式部署**：支持多实例部署
- **数据持久化**：集成数据库存储
- **用户认证**：JWT 令牌验证
- **API 网关**：统一入口和路由
- **链路追踪**：分布式链路追踪
- **配置中心**：动态配置管理

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交代码
4. 发起 Pull Request

## �� 许可证

MIT License 