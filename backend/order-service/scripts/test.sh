#!/bin/bash

# 设置脚本在遇到错误时退出
set -e

# 服务地址
BASE_URL="http://localhost:8082"

echo "开始测试 Order Service API..."

# 测试健康检查
echo "1. 测试健康检查..."
curl -s -X GET "$BASE_URL/health" | jq '.' || echo "健康检查失败"

echo ""

# 测试获取订单详情（订单不存在）
echo "2. 测试获取不存在的订单..."
curl -s -X GET "$BASE_URL/api/v1/orders/non-existent-order" | jq '.' || echo "请求失败"

echo ""

# 测试获取用户订单列表
echo "3. 测试获取用户订单列表..."
curl -s -X GET "$BASE_URL/api/v1/users/1/orders?page=1&pageSize=10" | jq '.' || echo "请求失败"

echo ""

# 测试获取订单统计
echo "4. 测试获取订单统计..."
curl -s -X GET "$BASE_URL/api/v1/stats/orders?date=2024-01-01" | jq '.' || echo "请求失败"

echo ""

# 测试获取失败订单列表
echo "5. 测试获取失败订单列表..."
curl -s -X GET "$BASE_URL/api/v1/failures?page=1&pageSize=10" | jq '.' || echo "请求失败"

echo ""

# 测试更新订单状态（订单不存在）
echo "6. 测试更新不存在订单的状态..."
curl -s -X PUT "$BASE_URL/api/v1/orders/non-existent-order/status" \
  -H "Content-Type: application/json" \
  -d '{"status":"completed","remark":"测试更新"}' | jq '.' || echo "请求失败"

echo ""

# 测试取消订单（订单不存在）
echo "7. 测试取消不存在的订单..."
curl -s -X PUT "$BASE_URL/api/v1/orders/non-existent-order/cancel" \
  -H "Content-Type: application/json" \
  -d '{"reason":"测试取消"}' | jq '.' || echo "请求失败"

echo ""

# 测试重试失败订单（记录不存在）
echo "8. 测试重试不存在的失败订单..."
curl -s -X POST "$BASE_URL/api/v1/failures/999/retry" | jq '.' || echo "请求失败"

echo ""

echo "API 测试完成!"
echo ""
echo "注意: 以上测试主要验证 API 接口的可用性和错误处理"
echo "在实际使用中，需要先通过消息队列创建订单，然后再测试相关功能"
echo ""
echo "模拟秒杀订单消息的方法:"
echo "1. 启动 RabbitMQ 或 Kafka"
echo "2. 发送秒杀成功消息到相应队列"
echo "3. Order Service 会自动消费消息并创建订单"
echo "4. 然后可以通过 API 查询订单信息" 