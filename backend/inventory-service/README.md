# Inventory Service

Inventory Service 是 MiniShop 电商系统的库存服务，负责管理商品的真实库存，提供库存同步接口，并进行健康检查以确保数据一致性。

## 功能特性

### 核心功能
- **库存同步**: 提供 `/sync-stock` 接口，接收增量并更新数据库中的真实库存
- **健康检查**: 定时对比 Redis 与 DB 库存差异，发现偏差时报警或触发补偿
- **库存管理**: 提供完整的库存查询、创建、更新等管理功能
- **差异修复**: 支持自动和手动修复库存差异
- **预警机制**: 低库存、缺货、差异过大等情况的自动预警

### 技术特性
- **乐观锁**: 使用版本号防止并发更新冲突
- **事务支持**: 库存操作使用数据库事务确保一致性
- **缓存集成**: Redis 缓存提升查询性能
- **操作日志**: 详细记录所有库存变更操作
- **最终一致性**: 与 order-service 配合保证库存数据最终一致

## 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Order         │    │   Inventory     │    │   Cache         │
│   Service       │───▶│   Service       │◀──▶│   Service       │
│                 │    │                 │    │   (Redis)       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                       ┌─────────────────┐
                       │   Database      │
                       │ (PostgreSQL/    │
                       │  MySQL)         │
                       └─────────────────┘
                                │
                       ┌─────────────────┐
                       │   Health        │
                       │   Checker       │
                       └─────────────────┘
```

## 数据库设计

### 库存表 (inventories)
- `id`: 主键
- `product_id`: 商品ID (唯一索引)
- `product_name`: 商品名称
- `stock`: 当前库存
- `reserved`: 预留库存
- `available`: 可用库存 = stock - reserved
- `version`: 乐观锁版本号
- `status`: 库存状态 (active/inactive/locked)
- `min_stock`: 最小库存阈值
- `max_stock`: 最大库存限制

### 库存操作日志表 (inventory_logs)
- `id`: 主键
- `product_id`: 商品ID
- `op_type`: 操作类型 (deduct/add/sync/init)
- `delta`: 变化量
- `before_stock`: 操作前库存
- `after_stock`: 操作后库存
- `order_id`: 关联订单ID
- `reason`: 操作原因
- `operator`: 操作者
- `trace_id`: 追踪ID

### 库存差异记录表 (inventory_diffs)
- `id`: 主键
- `product_id`: 商品ID
- `db_stock`: 数据库库存
- `redis_stock`: Redis库存
- `diff`: 差异量
- `status`: 状态 (pending/fixed/ignored)
- `fixed_at`: 修复时间
- `fixed_by`: 修复人

### 库存预警记录表 (inventory_alerts)
- `id`: 主键
- `product_id`: 商品ID
- `alert_type`: 预警类型 (low_stock/out_of_stock/diff_alert)
- `message`: 预警消息
- `level`: 预警级别 (info/warning/error)
- `status`: 状态 (pending/sent/ignored)

## API 接口

### 核心接口
- `POST /api/v1/sync-stock` - **库存同步接口** (核心功能)

### 库存管理
- `GET /api/v1/inventory/:productId` - 获取单个库存信息
- `POST /api/v1/inventory/batch` - 批量获取库存信息
- `GET /api/v1/inventory` - 获取库存列表
- `POST /api/v1/inventory` - 创建库存记录
- `PUT /api/v1/inventory/:productId` - 更新库存信息

### 健康检查
- `GET /api/v1/health/check` - 库存健康检查
- `POST /api/v1/health/trigger` - 手动触发健康检查
- `POST /api/v1/health/fix/:diffId` - 修复库存差异

### 统计查询
- `GET /api/v1/stats/inventory` - 获取库存统计
- `GET /api/v1/stats/service` - 获取服务统计

### 系统接口
- `GET /health` - 服务健康检查

## 配置说明

### 服务配置 (config/config.yaml)
```yaml
server:
  port: 8083
  grpc_port: 9083

database:
  driver: postgres
  postgres:
    host: postgres
    port: 5432
    user: postgres
    password: postgres
    dbname: inventory_db

redis:
  host: redis
  port: 6379
  db: 2

inventory:
  default_stock: 1000
  low_stock_threshold: 100
  health_check:
    enable: true
    interval: 300s
    tolerance: 10
    alert_threshold: 50
  compensation:
    enable: true
    auto_fix: false
    max_fix_amount: 1000
```

## 部署指南

### Docker 部署

1. **快速启动**
```bash
# 基础服务
./scripts/start.sh

# 包含监控
./scripts/start.sh --monitoring
```

2. **手动部署**
```bash
# 构建镜像
docker build -t inventory-service .

# 启动服务
docker-compose up -d
```

3. **检查服务状态**
```bash
docker-compose ps
docker-compose logs -f inventory-service
```

### 本地开发

1. **安装依赖**
```bash
go mod download
```

2. **启动依赖服务**
```bash
docker-compose up -d postgres redis
```

3. **运行服务**
```bash
go run cmd/main.go
```

## 使用示例

### 库存同步 (核心功能)

```bash
# 扣减库存 (order-service 调用)
curl -X POST http://localhost:8083/api/v1/sync-stock \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 1001,
    "delta": -5,
    "order_id": "order_12345",
    "reason": "订单扣减",
    "trace_id": "trace_12345"
  }'

# 增加库存 (补货)
curl -X POST http://localhost:8083/api/v1/sync-stock \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 1001,
    "delta": 100,
    "reason": "库存补充",
    "trace_id": "restock_001"
  }'
```

### 库存查询

```bash
# 获取单个库存
curl http://localhost:8083/api/v1/inventory/1001

# 批量获取库存
curl -X POST http://localhost:8083/api/v1/inventory/batch \
  -H "Content-Type: application/json" \
  -d '{"product_ids": [1001, 1002, 1003]}'
```

### 健康检查

```bash
# 执行健康检查
curl http://localhost:8083/api/v1/health/check

# 手动触发检查
curl -X POST http://localhost:8083/api/v1/health/trigger

# 修复差异
curl -X POST http://localhost:8083/api/v1/health/fix/1 \
  -H "Content-Type: application/json" \
  -d '{"fix_type": "use_redis"}'
```

## 测试

### API 测试
```bash
# 运行完整测试套件
./scripts/test.sh
```

### 手动测试

1. **健康检查**
```bash
curl http://localhost:8083/health
```

2. **库存同步测试**
```bash
curl -X POST http://localhost:8083/api/v1/sync-stock \
  -H "Content-Type: application/json" \
  -d '{"product_id": 1001, "delta": -1, "reason": "测试"}'
```

## 与其他服务的集成

### Order Service 集成

Order Service 在处理订单后调用库存同步接口：

```go
// order-service 中的示例代码
func (s *OrderService) syncInventory(productID int64, quantity int64, orderID string) error {
    syncReq := &SyncStockRequest{
        ProductID: productID,
        Delta:     -quantity, // 扣减库存
        OrderID:   orderID,
        Reason:    "订单创建",
        TraceID:   generateTraceID(),
    }
    
    resp, err := inventoryClient.SyncStock(ctx, syncReq)
    if err != nil {
        return err
    }
    
    if !resp.Success {
        return fmt.Errorf("库存同步失败: %s", resp.Message)
    }
    
    return nil
}
```

### Cache Service 集成

健康检查时对比 Redis 缓存中的库存：

```bash
# Redis 中的库存键格式
seckill:stock:1001  # 商品1001的库存
```

## 监控和日志

### 健康检查监控
- 定时检查库存差异
- 自动生成预警
- 支持自动修复

### 关键指标
- 库存同步成功率
- 库存差异发现数量
- 自动修复成功率
- API 响应时间

### 日志级别
- INFO: 正常业务操作
- WARN: 库存预警、差异发现
- ERROR: 系统错误、修复失败

## 故障排除

### 常见问题

1. **库存同步失败**
   - 检查数据库连接
   - 验证商品ID是否存在
   - 查看乐观锁冲突

2. **健康检查发现差异**
   - 检查 Redis 连接
   - 验证缓存键格式
   - 查看同步时序问题

3. **库存不足错误**
   - 确认当前库存数量
   - 检查预留库存设置
   - 验证并发控制

### 调试命令

```bash
# 查看服务日志
docker-compose logs -f inventory-service

# 检查数据库连接
docker-compose exec postgres psql -U postgres -d inventory_db

# 检查 Redis 连接
docker-compose exec redis redis-cli
```

## 性能优化

### 数据库优化
- 商品ID唯一索引
- 复合索引优化查询
- 连接池配置

### 缓存策略
- 库存信息缓存
- 查询结果缓存
- 缓存预热

### 并发控制
- 乐观锁版本控制
- 事务隔离级别
- 连接池管理

## 扩展功能

### 预留库存
- 支持库存预留和释放
- 订单超时自动释放
- 预留库存统计

### 批量操作
- 批量库存同步
- 批量健康检查
- 批量差异修复

### 历史追踪
- 库存变更历史
- 操作审计日志
- 数据恢复支持

## 版本历史

- v1.0.0: 初始版本，支持基本库存管理
- v1.1.0: 增加健康检查功能
- v1.2.0: 支持自动差异修复
- v1.3.0: 优化并发性能和监控 