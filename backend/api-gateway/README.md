# API Gateway

API Gateway 是 MiniShop 秒杀系统的统一入口，负责请求路由、认证授权、限流保护、跨域处理等功能。

## 功能特性

### 核心功能
- **反向代理**: 将请求路由到相应的后端微服务
- **认证授权**: JWT Token 认证 + 签名校验双重保护
- **限流保护**: 全局、用户、IP、接口多级限流策略
- **跨域支持**: 完整的 CORS 中间件支持
- **健康检查**: 网关和后端服务的健康状态监控
- **监控统计**: 请求统计、性能指标、服务状态

### 技术特性
- **高性能**: 基于 Gin 框架，支持高并发请求
- **分布式限流**: 基于 Redis 的滑动窗口限流算法
- **服务发现**: 支持多个后端服务的负载均衡
- **链路追踪**: 自动注入 Trace ID 用于请求追踪
- **优雅关闭**: 支持平滑重启和关闭

## 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│     Client      │    │   API Gateway   │    │   Backend       │
│   (Frontend/    │───▶│                 │───▶│   Services      │
│    Mobile)      │    │  - 认证授权     │    │                 │
│                 │    │  - 限流保护     │    │  - cache-service│
│                 │    │  - 路由代理     │    │  - seckill-service│
│                 │    │  - 监控统计     │    │  - order-service│
│                 │    │                 │    │  - inventory-service│
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                       ┌─────────────────┐
                       │     Redis       │
                       │  (限流缓存)     │
                       └─────────────────┘
```

## 路由配置

### 路径映射
- `/api/v1/cache/*` → `cache-service:8081`
- `/api/v1/seckill/*` → `seckill-service:8082`
- `/api/v1/order/*` → `order-service:8084`
- `/api/v1/inventory/*` → `inventory-service:8083`

### 特殊路径
- `/health` - 网关健康检查
- `/stats` - 网关统计信息
- `/api/v1/auth/login` - 用户登录
- `/api/v1/auth/refresh` - 刷新 Token
- `/metrics` - Prometheus 监控指标

## 中间件说明

### 1. 认证中间件 (AuthMiddleware)
- **JWT 认证**: 验证 Bearer Token
- **签名校验**: HMAC-SHA256 签名验证
- **白名单**: 支持路径白名单，无需认证

#### JWT Token 格式
```json
{
  "user_id": 1,
  "username": "admin",
  "roles": ["admin"],
  "iat": 1640995200,
  "exp": 1640998800
}
```

#### 签名校验流程
1. 客户端发送请求时携带 `timestamp`、`nonce`、`signature` 头部
2. 网关按规则重新计算签名
3. 对比客户端签名和服务端计算的签名
4. 验证时间戳是否在有效期内

### 2. 限流中间件 (RateLimiter)
- **全局限流**: 整个网关的总请求限制
- **用户限流**: 基于用户ID的限流
- **IP限流**: 基于客户端IP的限流
- **接口限流**: 针对特定接口的限流

#### 限流算法
使用基于 Redis 的滑动窗口算法：
```
时间窗口: [now-window, now]
限流键: rate_limit:{type}:{identifier}
算法: 清理过期记录 → 统计当前请求数 → 添加新请求 → 判断是否超限
```

### 3. CORS 中间件 (CORSMiddleware)
- 支持跨域请求
- 可配置允许的域名、方法、头部
- 支持预检请求 (OPTIONS)

## 配置说明

### 服务配置 (config/config.yaml)
```yaml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

# 后端服务配置
services:
  cache-service:
    url: "http://cache-service:8081"
    timeout: 10s
  seckill-service:
    url: "http://seckill-service:8082"
    timeout: 15s
  # ... 其他服务

# 限流配置
rate_limit:
  enable: true
  global:
    requests_per_second: 1000
    burst: 2000
  user:
    requests_per_second: 10
    burst: 20
  # ... 其他限流配置

# 认证配置
auth:
  enable: true
  jwt_secret: "your-jwt-secret"
  signature:
    enable: true
    secret: "your-signature-secret"
  whitelist:
    - "/health"
    - "/api/v1/auth/login"
```

## 部署指南

### Docker 部署

1. **快速启动整个系统**
```bash
# 基础系统
./scripts/start.sh

# 包含监控
./scripts/start.sh --monitoring

# 包含 Nginx 负载均衡
./scripts/start.sh --nginx

# 完整系统（包含所有组件）
./scripts/start.sh --all
```

2. **单独启动 API Gateway**
```bash
# 构建镜像
docker build -t api-gateway .

# 启动容器
docker run -d \
  --name api-gateway \
  -p 8080:8080 \
  -p 9090:9090 \
  -v ./config:/root/config \
  api-gateway
```

### 本地开发

1. **安装依赖**
```bash
go mod download
```

2. **启动依赖服务**
```bash
docker-compose up -d redis
```

3. **运行网关**
```bash
go run cmd/main.go
```

## API 接口

### 认证接口

#### 用户登录
```bash
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password"
}
```

响应：
```json
{
  "code": 0,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "user_id": 1,
    "username": "admin",
    "roles": ["admin"]
  },
  "msg": "登录成功"
}
```

#### 刷新 Token
```bash
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

### 管理接口

#### 健康检查
```bash
GET /health
```

响应：
```json
{
  "status": "healthy",
  "message": "API Gateway Health Check",
  "services": {
    "cache-service": {
      "status": "healthy",
      "status_code": 200,
      "duration": "5ms",
      "url": "http://cache-service:8081/health"
    }
  },
  "summary": {
    "healthy_services": 4,
    "total_services": 4
  }
}
```

#### 统计信息
```bash
GET /stats
```

### 代理接口

所有 `/api/v1/*` 路径的请求都会被代理到相应的后端服务：

```bash
# 秒杀接口 → seckill-service
POST /api/v1/seckill/purchase
Authorization: Bearer <token>

# 库存查询 → inventory-service
GET /api/v1/inventory/1001

# 订单查询 → order-service
GET /api/v1/order/orders/order_123

# 缓存操作 → cache-service
GET /api/v1/cache/seckill/stock/1001
```

## 使用示例

### 1. 完整的秒杀流程

```bash
# 1. 用户登录获取 token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}' | \
  jq -r '.data.token')

# 2. 查看商品库存
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/cache/seckill/stock/1001

# 3. 参与秒杀
curl -X POST http://localhost:8080/api/v1/seckill/purchase \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"product_id":1001,"user_id":1}'

# 4. 查看订单状态
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/order/users/1/orders
```

### 2. 带签名的请求

```bash
# 生成签名参数
timestamp=$(date +%s)
nonce="random123"

# 计算签名（实际应用中需要按规则计算）
signature="calculated_signature_here"

# 发送带签名的请求
curl -X POST http://localhost:8080/api/v1/seckill/purchase \
  -H "Authorization: Bearer $TOKEN" \
  -H "timestamp: $timestamp" \
  -H "nonce: $nonce" \
  -H "signature: $signature" \
  -H "Content-Type: application/json" \
  -d '{"product_id":1001,"user_id":1}'
```

## 监控和日志

### 健康检查
- 网关自身健康状态
- 后端服务健康状态
- 依赖组件（Redis）健康状态

### 性能指标
- 请求总数和成功率
- 响应时间分布
- 限流触发次数
- 各服务代理统计

### 日志级别
- DEBUG: 详细的调试信息
- INFO: 正常的业务操作
- WARN: 限流触发、认证失败
- ERROR: 系统错误、服务不可用

### Prometheus 监控
```bash
# 访问监控指标
curl http://localhost:9090/metrics
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
curl http://localhost:8080/health
```

2. **登录测试**
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'
```

3. **限流测试**
```bash
# 快速发送多个请求测试限流
for i in {1..20}; do
  curl -s http://localhost:8080/health &
done
wait
```

## 性能优化

### 连接池配置
- HTTP 客户端连接池
- Redis 连接池
- 合理的超时设置

### 缓存策略
- JWT Token 缓存
- 限流状态缓存
- 服务健康状态缓存

### 并发处理
- Goroutine 池
- 异步日志写入
- 非阻塞限流检查

## 安全考虑

### 认证安全
- JWT Secret 定期轮换
- Token 过期时间设置
- 刷新 Token 机制

### 签名安全
- HMAC-SHA256 签名算法
- 时间戳防重放攻击
- Nonce 防重复请求

### 限流保护
- 多级限流策略
- 恶意请求识别
- 自动封禁机制

## 故障排除

### 常见问题

1. **服务不可用 (502)**
   - 检查后端服务是否启动
   - 验证服务配置和网络连通性
   - 查看服务健康检查状态

2. **认证失败 (401)**
   - 检查 JWT Secret 配置
   - 验证 Token 格式和有效期
   - 确认签名计算是否正确

3. **限流触发 (429)**
   - 检查限流配置是否合理
   - 查看 Redis 连接状态
   - 分析请求频率和来源

### 调试命令

```bash
# 查看网关日志
docker-compose logs -f api-gateway

# 检查 Redis 连接
docker-compose exec redis redis-cli ping

# 查看服务状态
curl http://localhost:8080/health

# 查看统计信息
curl http://localhost:8080/stats
```

## 扩展功能

### 负载均衡
- 支持多个后端服务实例
- 轮询、随机、加权等策略
- 健康检查和故障切换

### 缓存增强
- 响应缓存
- 接口结果缓存
- 缓存穿透保护

### 安全增强
- IP 白名单/黑名单
- 请求频率异常检测
- DDoS 攻击防护

## 版本历史

- v1.0.0: 初始版本，基础代理功能
- v1.1.0: 增加认证和限流
- v1.2.0: 支持签名校验和 CORS
- v1.3.0: 完善监控和健康检查
- v1.4.0: 优化性能和错误处理

---

API Gateway 作为整个秒杀系统的统一入口，提供了完整的安全、性能和可观测性保障，是高并发场景下的重要组件。 