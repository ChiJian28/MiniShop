#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 服务配置
BASE_URL="http://localhost:8083"
API_PREFIX="/api/v1"

# 测试数据
PRODUCT_ID=1001
USER_ID=2001
QUANTITY=1

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

log_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

# 检查服务是否运行
check_service() {
    log_info "Checking if seckill service is running..."
    
    if ! curl -s "$BASE_URL/health" > /dev/null; then
        log_error "Seckill service is not running or not accessible at $BASE_URL"
        log_error "Please start the service first using: ./scripts/start.sh"
        exit 1
    fi
    
    log_info "✓ Seckill service is running"
}

# 测试健康检查
test_health() {
    log_test "Testing health check..."
    
    response=$(curl -s "$BASE_URL/health")
    if echo "$response" | grep -q "healthy"; then
        log_info "✓ Health check passed"
        echo "Response: $response"
    else
        log_error "✗ Health check failed"
        echo "Response: $response"
    fi
    echo ""
}

# 测试预热活动
test_prewarm() {
    log_test "Testing activity prewarm..."
    
    local payload=$(cat <<EOF
{
    "product_id": $PRODUCT_ID,
    "product_name": "Test Product",
    "price": 99.99,
    "stock": 100,
    "start_time": "2024-01-01T10:00:00Z",
    "end_time": "2024-12-31T23:59:59Z",
    "status": "active"
}
EOF
)
    
    response=$(curl -s -X POST "$BASE_URL$API_PREFIX/seckill/activity/prewarm" \
        -H "Content-Type: application/json" \
        -d "$payload")
    
    if echo "$response" | grep -q "successfully"; then
        log_info "✓ Activity prewarm successful"
    else
        log_error "✗ Activity prewarm failed"
    fi
    echo "Response: $response"
    echo ""
}

# 测试同步秒杀
test_sync_seckill() {
    log_test "Testing synchronous seckill..."
    
    local payload=$(cat <<EOF
{
    "product_id": $PRODUCT_ID,
    "user_id": $USER_ID,
    "quantity": $QUANTITY
}
EOF
)
    
    response=$(curl -s -X POST "$BASE_URL$API_PREFIX/seckill/purchase" \
        -H "Content-Type: application/json" \
        -d "$payload")
    
    if echo "$response" | grep -q "success.*true"; then
        log_info "✓ Synchronous seckill successful"
    else
        log_warn "⚠ Synchronous seckill result (may be expected):"
    fi
    echo "Response: $response"
    echo ""
}

# 测试异步秒杀
test_async_seckill() {
    log_test "Testing asynchronous seckill..."
    
    local user_id=$((USER_ID + 1))
    local payload=$(cat <<EOF
{
    "product_id": $PRODUCT_ID,
    "user_id": $user_id,
    "quantity": $QUANTITY
}
EOF
)
    
    response=$(curl -s -X POST "$BASE_URL$API_PREFIX/seckill/purchase/async" \
        -H "Content-Type: application/json" \
        -d "$payload")
    
    if echo "$response" | grep -q "accepted"; then
        log_info "✓ Asynchronous seckill request accepted"
    else
        log_warn "⚠ Asynchronous seckill result:"
    fi
    echo "Response: $response"
    echo ""
}

# 测试获取统计信息
test_stats() {
    log_test "Testing get seckill stats..."
    
    response=$(curl -s "$BASE_URL$API_PREFIX/seckill/stats/$PRODUCT_ID")
    
    if echo "$response" | grep -q "product_id"; then
        log_info "✓ Get seckill stats successful"
    else
        log_error "✗ Get seckill stats failed"
    fi
    echo "Response: $response"
    echo ""
}

# 测试检查用户购买状态
test_user_purchased() {
    log_test "Testing check user purchased status..."
    
    response=$(curl -s "$BASE_URL$API_PREFIX/seckill/purchased/$PRODUCT_ID/$USER_ID")
    
    if echo "$response" | grep -q "purchased"; then
        log_info "✓ Check user purchased status successful"
    else
        log_error "✗ Check user purchased status failed"
    fi
    echo "Response: $response"
    echo ""
}

# 测试获取用户购买信息
test_user_purchase_info() {
    log_test "Testing get user purchase info..."
    
    response=$(curl -s "$BASE_URL$API_PREFIX/seckill/purchase/$PRODUCT_ID/$USER_ID")
    
    if echo "$response" | grep -q "user_id"; then
        log_info "✓ Get user purchase info successful"
    else
        log_error "✗ Get user purchase info failed"
    fi
    echo "Response: $response"
    echo ""
}

# 测试获取系统统计
test_system_stats() {
    log_test "Testing get system stats..."
    
    response=$(curl -s "$BASE_URL$API_PREFIX/system/stats")
    
    if echo "$response" | grep -q "service_stats"; then
        log_info "✓ Get system stats successful"
    else
        log_error "✗ Get system stats failed"
    fi
    echo "Response: $response"
    echo ""
}

# 压力测试
stress_test() {
    log_test "Running stress test..."
    
    local concurrent_users=10
    local requests_per_user=5
    
    log_info "Starting stress test with $concurrent_users concurrent users, $requests_per_user requests each"
    
    for i in $(seq 1 $concurrent_users); do
        {
            local user_id=$((USER_ID + i))
            for j in $(seq 1 $requests_per_user); do
                local payload=$(cat <<EOF
{
    "product_id": $PRODUCT_ID,
    "user_id": $user_id,
    "quantity": 1
}
EOF
)
                curl -s -X POST "$BASE_URL$API_PREFIX/seckill/purchase" \
                    -H "Content-Type: application/json" \
                    -d "$payload" > /dev/null
            done
        } &
    done
    
    wait
    log_info "✓ Stress test completed"
    echo ""
}

# 清理测试数据
cleanup() {
    log_test "Cleaning up test data..."
    
    response=$(curl -s -X DELETE "$BASE_URL$API_PREFIX/seckill/activity/$PRODUCT_ID")
    
    if echo "$response" | grep -q "successfully"; then
        log_info "✓ Test data cleaned up successfully"
    else
        log_warn "⚠ Cleanup may have failed (this is okay if data doesn't exist)"
    fi
    echo "Response: $response"
    echo ""
}

# 显示帮助信息
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo "Test the seckill service API"
    echo ""
    echo "Options:"
    echo "  --help, -h         Show this help message"
    echo "  --health           Test health check only"
    echo "  --basic            Run basic functionality tests"
    echo "  --stress           Run stress test"
    echo "  --cleanup          Clean up test data"
    echo "  --all              Run all tests (default)"
    echo ""
    echo "Examples:"
    echo "  $0                 # Run all tests"
    echo "  $0 --basic         # Run basic tests only"
    echo "  $0 --stress        # Run stress test only"
}

# 运行基础测试
run_basic_tests() {
    log_info "Running basic API tests..."
    echo ""
    
    test_health
    test_prewarm
    test_sync_seckill
    test_async_seckill
    test_stats
    test_user_purchased
    test_user_purchase_info
    test_system_stats
    
    log_info "Basic tests completed!"
}

# 运行所有测试
run_all_tests() {
    log_info "Running comprehensive API tests..."
    echo ""
    
    run_basic_tests
    stress_test
    cleanup
    
    log_info "All tests completed!"
}

# 主函数
main() {
    check_service
    
    case "${1:-}" in
        --help|-h)
            show_help
            ;;
        --health)
            test_health
            ;;
        --basic)
            run_basic_tests
            ;;
        --stress)
            stress_test
            ;;
        --cleanup)
            cleanup
            ;;
        --all|"")
            run_all_tests
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@" 