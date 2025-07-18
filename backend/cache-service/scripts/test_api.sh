#!/bin/bash

# Cache Service API 测试脚本

BASE_URL="http://localhost:8082"

echo "=== Cache Service API 测试 ==="

# 1. 健康检查
echo "1. 健康检查..."
curl -s -X GET "$BASE_URL/health" | jq .
echo ""

# 2. 基础缓存操作测试
echo "2. 基础缓存操作测试..."

# 设置缓存
echo "设置缓存..."
curl -s -X POST "$BASE_URL/api/v1/cache/" \
  -H "Content-Type: application/json" \
  -d '{"key": "test_key", "value": "test_value", "expiration": 3600}' | jq .

# 获取缓存
echo "获取缓存..."
curl -s -X GET "$BASE_URL/api/v1/cache/test_key" | jq .

echo ""

# 3. 秒杀功能测试
echo "3. 秒杀功能测试..."

# 预加载秒杀活动
echo "预加载秒杀活动..."
curl -s -X POST "$BASE_URL/api/v1/seckill/activity" \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 1001,
    "product_name": "iPhone 15 Pro",
    "price": 8999.00,
    "stock": 100,
    "start_time": "2024-01-01T10:00:00Z",
    "end_time": "2024-01-01T12:00:00Z",
    "status": "active"
  }' | jq .

# 获取库存
echo "获取库存..."
curl -s -X GET "$BASE_URL/api/v1/seckill/stock/1001" | jq .

# 用户购买
echo "用户 2001 购买..."
curl -s -X POST "$BASE_URL/api/v1/seckill/purchase" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 1001, "user_id": 2001, "quantity": 1}' | jq .

# 用户 2002 购买
echo "用户 2002 购买..."
curl -s -X POST "$BASE_URL/api/v1/seckill/purchase" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 1001, "user_id": 2002, "quantity": 1}' | jq .

# 用户 2001 再次购买（应该失败）
echo "用户 2001 再次购买..."
curl -s -X POST "$BASE_URL/api/v1/seckill/purchase" \
  -H "Content-Type: application/json" \
  -d '{"product_id": 1001, "user_id": 2001, "quantity": 1}' | jq .

# 检查购买后的库存
echo "检查购买后的库存..."
curl -s -X GET "$BASE_URL/api/v1/seckill/stock/1001" | jq .

# 检查用户购买状态
echo "检查用户 2001 购买状态..."
curl -s -X GET "$BASE_URL/api/v1/seckill/purchased/1001/2001" | jq .

echo "检查用户 2003 购买状态..."
curl -s -X GET "$BASE_URL/api/v1/seckill/purchased/1001/2003" | jq .

# 获取用户购买信息
echo "获取用户 2001 购买信息..."
curl -s -X GET "$BASE_URL/api/v1/seckill/purchase/1001/2001" | jq .

# 获取购买用户数量
echo "获取购买用户数量..."
curl -s -X GET "$BASE_URL/api/v1/seckill/count/1001" | jq .

# 获取秒杀活动信息
echo "获取秒杀活动信息..."
curl -s -X GET "$BASE_URL/api/v1/seckill/activity/1001" | jq .

echo ""

# 4. 清理测试数据
echo "4. 清理测试数据..."
curl -s -X DELETE "$BASE_URL/api/v1/seckill/cleanup/1001" | jq .

# 删除基础缓存
curl -s -X DELETE "$BASE_URL/api/v1/cache/" \
  -H "Content-Type: application/json" \
  -d '{"keys": ["test_key"]}' | jq .

echo ""
echo "=== API 测试完成 ===" 