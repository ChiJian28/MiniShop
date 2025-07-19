# Order Service

Order Service 是 MiniShop 电商系统的订单服务，负责处理秒杀订单的异步创建、管理和失败补偿。

## 功能特性

### 核心功能
- **异步订单创建**: 消费来自 seckill-service 的秒杀成功消息，异步创建订单
- **幂等性保证**: 通过数据库唯一索引和业务逻辑确保订单不重复创建
- **失败补偿**: 自动重试失败的订单创建，支持手动重试和定时任务
- **订单管理**: 提供完整的订单查询、状态更新、取消等功能
- **统计分析**: 提供订单统计数据，支持按日期查询

### 技术特性
- **高并发处理**: 支持大量并发订单创建请求
- **消息队列集成**: 支持 RabbitMQ 和 Kafka 消息队列
- **缓存优化**: 使用 Redis 缓存订单数据，提升查询性能
- **数据库支持**: 支持 PostgreSQL 和 MySQL 数据库
- **健康检查**: 提供服务健康状态监控
- **优雅关闭**: 支持优雅关闭和资源清理

## 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Seckill       │    │   Message       │    │   Order         │
│   Service       │───▶│   Queue         │───▶│   Service       │
│                 │    │ (RabbitMQ/Kafka)│    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
                       ┌─────────────────┐             │
                       │   Database      │◀────────────┘
                       │ (PostgreSQL/    │
                       │  MySQL)         │
                       └─────────────────┘
                                │
                       ┌─────────────────┐
                       │   Redis         │
                       │   Cache         │
                       └─────────────────┘
```

## 数据库设计

### 订单表 (orders)
- `id`: 主键
- `order_id`: 订单号 (UUID)
- `user_id`: 用户ID
- `product_id`: 商品ID
- `quantity`: 购买数量
- `unit_price`: 单价
- `total_amount`: 总金额
- `status`: 订单状态 (pending/completed/cancelled/failed)
- `created_at`: 创建时间
- `updated_at`: 更新时间

### 订单项表 (order_items)
- `id`: 主键
- `order_id`: 订单ID
- `product_id`: 商品ID
- `product_name`: 商品名称
- `quantity`: 数量
- `unit_price`: 单价
- `total_price`: 小计

### 幂等性记录表 (idempotency_records)
- `id`: 主键
- `user_id`: 用户ID
- `product_id`: 商品ID
- `order_id`: 订单ID
- `created_at`: 创建时间
- 唯一索引: (user_id, product_id)

### 失败补偿表 (order_failures)
- `id`: 主键
- `message_data`: 原始消息数据
- `error_message`: 错误信息
- `retry_count`: 重试次数
- `status`: 状态 (pending/processing/success/failed)
- `created_at`: 创建时间
- `updated_at`: 更新时间

## API 接口

### 订单管理
- `GET /api/v1/orders/:orderId` - 获取订单详情
- `PUT /api/v1/orders/:orderId/status` - 更新订单状态
- `PUT /api/v1/orders/:orderId/cancel` - 取消订单
- `GET /api/v1/users/:userId/orders` - 获取用户订单列表

### 统计查询
- `GET /api/v1/stats/orders` - 获取订单统计

### 失败补偿
- `GET /api/v1/failures` - 获取失败订单列表
- `POST /api/v1/failures/:failureId/retry` - 重试失败订单

### 健康检查
- `GET /health` - 服务健康检查

## 配置说明

### 服务配置 (config/config.yaml)
```yaml
server:
  port: 8082
  read_timeout: 30s
  write_timeout: 30s

database:
  driver: postgres
  host: postgres
  port: 5432
  user: postgres
  password: password
  dbname: order_db
  
redis:
  host: redis
  port: 6379
  password: ""
  db: 0

rabbitmq:
  enabled: true
  host: rabbitmq
  port: 5672
  username: admin
  password: password

kafka:
  enabled: false
  brokers: ["kafka:9092"]
```

## 部署指南

### Docker 部署

1. **构建镜像**
```bash
docker build -t order-service .
```

2. **使用 Docker Compose 启动**
```bash
# 快速启动
./scripts/start.sh

# 或手动启动
docker-compose up -d
```

3. **检查服务状态**
```bash
docker-compose ps
docker-compose logs -f order-service
```

### 本地开发

1. **安装依赖**
```bash
go mod download
```

2. **启动依赖服务**
```bash
docker-compose up -d postgres redis rabbitmq
```

3. **运行服务**
```bash
go run cmd/main.go
```

## 测试

### API 测试
```bash
# 运行 API 测试脚本
./scripts/test.sh
```

### 手动测试

1. **健康检查**
```bash
curl http://localhost:8082/health
```

2. **查询用户订单**
```bash
curl "http://localhost:8082/api/v1/users/1/orders?page=1&pageSize=10"
```

3. **获取订单统计**
```bash
curl "http://localhost:8082/api/v1/stats/orders?date=2024-01-01"
```

## 消息队列集成

### RabbitMQ 消息格式
```json
{
  "user_id": 123,
  "product_id": 456,
  "quantity": 1,
  "unit_price": 99.99,
  "seckill_id": "seckill_789",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### Kafka 消息格式
```json
{
  "user_id": 123,
  "product_id": 456,
  "quantity": 1,
  "unit_price": 99.99,
  "seckill_id": "seckill_789",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## 监控和日志

### 健康检查端点
- `/health` - 返回服务健康状态
- 检查数据库连接
- 检查 Redis 连接

### 日志级别
- INFO: 正常业务日志
- WARN: 警告信息
- ERROR: 错误信息

### 指标监控
- 订单创建成功率
- 消息处理延迟
- 失败补偿执行情况
- 数据库连接状态

## 故障排除

### 常见问题

1. **数据库连接失败**
   - 检查数据库服务是否启动
   - 验证连接参数配置
   - 检查网络连接

2. **Redis 连接失败**
   - 检查 Redis 服务状态
   - 验证 Redis 配置
   - 检查防火墙设置

3. **消息队列连接失败**
   - 确认 RabbitMQ/Kafka 服务运行
   - 检查认证信息
   - 验证队列/主题配置

4. **订单重复创建**
   - 检查幂等性索引是否生效
   - 验证消息去重逻辑
   - 查看数据库约束

### 日志查看
```bash
# 查看服务日志
docker-compose logs -f order-service

# 查看特定时间段日志
docker-compose logs --since="2024-01-01T00:00:00" order-service
```

## 性能优化

### 数据库优化
- 为常用查询字段添加索引
- 使用连接池管理数据库连接
- 定期清理过期数据

### 缓存策略
- 缓存热点订单数据
- 使用 Redis 集群提升性能
- 设置合理的缓存过期时间

### 消息队列优化
- 调整消费者并发数
- 使用批量消息处理
- 设置合理的重试策略

## 版本历史

- v1.0.0: 初始版本，支持基本订单管理功能
- v1.1.0: 增加失败补偿机制
- v1.2.0: 支持 Kafka 消息队列
- v1.3.0: 优化性能和缓存策略 