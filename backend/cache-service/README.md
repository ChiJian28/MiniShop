# Cache Service

基于 Redis 的缓存服务，专为秒杀系统设计，提供高性能的缓存操作和分布式锁功能。

## 功能特性

### 核心功能
- **Redis 封装层**：统一管理 Redis 连接与操作，屏蔽底层细节
- **通用缓存操作**：提供 GET/SET、列表/集合操作、分布式锁等 API
- **秒杀专用功能**：预加载和管理秒杀活动的库存、用户抢购状态

### 秒杀相关功能
- **库存管理**：`seckill:stock:<productId>` - 商品库存缓存
- **用户状态**：`seckill:users:<productId>` - 用户抢购状态记录
- **分布式锁**：`seckill:lock:<productId>` - 防止超卖的分布式锁
- **活动信息**：`seckill:activity:<productId>` - 秒杀活动详情
- **购买记录**：`seckill:purchase:<productId>:<userId>` - 用户购买信息

## 技术栈

- **Go 1.21**：主要编程语言
- **Gin**：HTTP 框架，提供 RESTful API
- **Redis**：缓存数据库
- **Docker**：容器化部署

## 项目结构

```
cache-service/
├── cmd/
│   └── main.go                 # 程序入口
├── internal/
│   ├── config/                 # 配置管理
│   │   └── config.go
│   ├── redis/                  # Redis 客户端封装
│   │   └── client.go
│   ├── lock/                   # 分布式锁实现
│   │   └── distributed_lock.go
│   ├── seckill/                # 秒杀业务逻辑
│   │   └── seckill.go
│   └── service/                # 服务层
│       └── cache_service.go
├── api/
│   └── rest/                   # REST API
│       ├── handler.go
│       └── router.go
├── config/
│   └── config.yaml             # 配置文件
├── go.mod                      # Go 模块文件
├── Dockerfile                  # Docker 构建文件
└── README.md                   # 项目说明
```

## 配置说明

### config.yaml
```yaml
server:
  port: 8082                    # HTTP 服务端口
  grpc_port: 9082              # gRPC 服务端口

redis:
  host: localhost              # Redis 主机
  port: 6379                   # Redis 端口
  password: ""                 # Redis 密码
  db: 0                        # Redis 数据库
  pool_size: 10                # 连接池大小
  min_idle_conns: 5            # 最小空闲连接数
  max_retries: 3               # 最大重试次数
  dial_timeout: 5s             # 连接超时
  read_timeout: 3s             # 读取超时
  write_timeout: 3s            # 写入超时

seckill:
  stock_key_prefix: "seckill:stock:"    # 库存键前缀
  user_key_prefix: "seckill:users:"     # 用户键前缀
  lock_key_prefix: "seckill:lock:"      # 锁键前缀
  default_ttl: 3600                     # 默认过期时间（秒）
```

## API 接口

### 基础缓存操作

#### 获取缓存
```http
GET /api/v1/cache/{key}
```

#### 设置缓存
```http
POST /api/v1/cache/
Content-Type: application/json

{
  "key": "test_key",
  "value": "test_value",
  "expiration": 3600
}
```

#### 删除缓存
```http
DELETE /api/v1/cache/
Content-Type: application/json

{
  "keys": ["key1", "key2"]
}
```

### 秒杀相关操作

#### 预加载秒杀活动
```http
POST /api/v1/seckill/activity
Content-Type: application/json

{
  "product_id": 1001,
  "product_name": "iPhone 15",
  "price": 5999.00,
  "stock": 100,
  "start_time": "2024-01-01T10:00:00Z",
  "end_time": "2024-01-01T12:00:00Z",
  "status": "active"
}
```

#### 获取库存
```http
GET /api/v1/seckill/stock/{productId}
```

#### 秒杀购买
```http
POST /api/v1/seckill/purchase
Content-Type: application/json

{
  "product_id": 1001,
  "user_id": 2001,
  "quantity": 1
}
```

#### 检查用户是否已购买
```http
GET /api/v1/seckill/purchased/{productId}/{userId}
```

#### 获取用户购买信息
```http
GET /api/v1/seckill/purchase/{productId}/{userId}
```

#### 获取购买用户数量
```http
GET /api/v1/seckill/count/{productId}
```

#### 清理秒杀数据
```http
DELETE /api/v1/seckill/cleanup/{productId}
```

### 健康检查
```http
GET /health
```

## 使用方法

### 本地开发

1. 启动 Redis 服务
```bash
redis-server
```

2. 运行应用
```bash
go run cmd/main.go
```

### Docker 部署

1. 构建镜像
```bash
docker build -t cache-service .
```

2. 运行容器
```bash
docker run -p 8082:8082 \
  -e REDIS_HOST=redis \
  -e REDIS_PORT=6379 \
  cache-service
```

### Docker Compose 部署

```yaml
version: '3.8'
services:
  cache-service:
    build: .
    ports:
      - "8082:8082"
    environment:
      - REDIS_HOST=redis
      - REDIS_PORT=6379
    depends_on:
      - redis
  
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```

## 核心特性

### 分布式锁
- 基于 Redis 的分布式锁实现
- 支持锁的获取、释放、续期
- 防止秒杀过程中的并发问题

### 原子操作
- 使用 Lua 脚本确保库存扣减的原子性
- 防止超卖问题

### 高性能
- Redis 连接池管理
- 批量操作支持
- 管道操作优化

### 可靠性
- 自动重试机制
- 连接健康检查
- 优雅关闭

## 监控指标

服务提供以下监控指标：
- Redis 连接状态
- 缓存命中率
- 接口响应时间
- 错误率统计

## 注意事项

1. **Redis 配置**：确保 Redis 服务正常运行，并根据实际需求调整连接池参数
2. **内存管理**：合理设置 TTL，避免内存溢出
3. **并发控制**：在高并发场景下，适当调整分布式锁的超时时间
4. **数据一致性**：秒杀结束后，需要将缓存数据同步到数据库

## 扩展功能

- 支持 Redis 集群模式
- 添加 Prometheus 监控指标
- 实现缓存预热策略
- 支持多级缓存架构 