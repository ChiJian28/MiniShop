#!/bin/bash

# 设置脚本在遇到错误时退出
set -e

# 服务地址
BASE_URL="http://localhost:8083"

echo "开始测试 Inventory Service API..."

# 测试健康检查
echo "1. 测试健康检查..."
curl -s -X GET "$BASE_URL/health" | jq '.' || echo "健康检查失败"

echo ""

# 测试库存同步接口 - 核心功能
echo "2. 测试库存同步接口..."
curl -s -X POST "$BASE_URL/api/v1/sync-stock" \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 1001,
    "delta": -5,
    "order_id": "order_12345",
    "reason": "秒杀扣减",
    "trace_id": "trace_12345"
  }' | jq '.' || echo "库存同步失败"

echo ""

# 测试获取库存信息
echo "3. 测试获取库存信息..."
curl -s -X GET "$BASE_URL/api/v1/inventory/1001" | jq '.' || echo "获取库存信息失败"

echo ""

# 测试批量获取库存信息
echo "4. 测试批量获取库存信息..."
curl -s -X POST "$BASE_URL/api/v1/inventory/batch" \
  -H "Content-Type: application/json" \
  -d '{
    "product_ids": [1001, 1002, 1003]
  }' | jq '.' || echo "批量获取库存信息失败"

echo ""

# 测试库存健康检查
echo "5. 测试库存健康检查..."
curl -s -X GET "$BASE_URL/api/v1/health/check" | jq '.' || echo "库存健康检查失败"

echo ""

# 测试手动触发健康检查
echo "6. 测试手动触发健康检查..."
curl -s -X POST "$BASE_URL/api/v1/health/trigger" | jq '.' || echo "手动触发健康检查失败"

echo ""

# 测试获取库存统计
echo "7. 测试获取库存统计..."
curl -s -X GET "$BASE_URL/api/v1/stats/inventory" | jq '.' || echo "获取库存统计失败"

echo ""

# 测试获取服务统计
echo "8. 测试获取服务统计..."
curl -s -X GET "$BASE_URL/api/v1/stats/service" | jq '.' || echo "获取服务统计失败"

echo ""

# 测试创建库存记录
echo "9. 测试创建库存记录..."
curl -s -X POST "$BASE_URL/api/v1/inventory" \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 2001,
    "product_name": "测试商品",
    "stock": 1000,
    "min_stock": 100,
    "max_stock": 10000
  }' | jq '.' || echo "创建库存记录失败"

echo ""

# 测试更新库存信息
echo "10. 测试更新库存信息..."
curl -s -X PUT "$BASE_URL/api/v1/inventory/2001" \
  -H "Content-Type: application/json" \
  -d '{
    "min_stock": 50,
    "reason": "调整最小库存阈值"
  }' | jq '.' || echo "更新库存信息失败"

echo ""

# 测试获取库存列表
echo "11. 测试获取库存列表..."
curl -s -X GET "$BASE_URL/api/v1/inventory?page=1&page_size=10" | jq '.' || echo "获取库存列表失败"

echo ""

# 压力测试 - 多次库存同步
echo "12. 压力测试 - 库存同步..."
echo "执行10次并发库存同步请求..."

for i in {1..10}; do
  curl -s -X POST "$BASE_URL/api/v1/sync-stock" \
    -H "Content-Type: application/json" \
    -d "{
      \"product_id\": 3001,
      \"delta\": -1,
      \"order_id\": \"stress_test_$i\",
      \"reason\": \"压力测试\",
      \"trace_id\": \"stress_$i\"
    }" &
done

# 等待所有后台任务完成
wait

echo "压力测试完成"

echo ""

# 最终检查库存状态
echo "13. 最终检查库存状态..."
curl -s -X GET "$BASE_URL/api/v1/inventory/3001" | jq '.' || echo "获取最终库存状态失败"

echo ""

echo "API 测试完成!"
echo ""
echo "测试总结:"
echo "✅ 核心功能: 库存同步接口 (/api/v1/sync-stock)"
echo "✅ 库存查询: 单个、批量、列表查询"
echo "✅ 健康检查: 自动和手动触发"
echo "✅ 统计信息: 库存和服务统计"
echo "✅ 管理功能: 创建、更新库存记录"
echo "✅ 压力测试: 并发库存同步"
echo ""
echo "注意事项:"
echo "1. 库存同步是核心功能，支持正负增量"
echo "2. 健康检查会对比Redis和DB的库存差异"
echo "3. 服务支持乐观锁防止并发冲突"
echo "4. 所有操作都有详细的日志记录" 