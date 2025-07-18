package examples

import (
	"context"
	"fmt"
	"log"
	"time"

	"cache-service/internal/redis"
	"cache-service/internal/seckill"
)

// 模拟秒杀场景的测试示例
func SeckillExample() {
	// 创建 Redis 配置
	redisConfig := &redis.Config{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	// 创建 Redis 客户端
	redisClient, err := redis.NewClient(redisConfig, nil)
	if err != nil {
		log.Fatalf("Failed to create redis client: %v", err)
	}
	defer redisClient.Close()

	// 创建秒杀配置
	seckillConfig := &seckill.Config{
		StockKeyPrefix: "seckill:stock:",
		UserKeyPrefix:  "seckill:users:",
		LockKeyPrefix:  "seckill:lock:",
		DefaultTTL:     3600 * time.Second,
	}

	// 创建秒杀缓存
	seckillCache := seckill.NewSeckillCache(redisClient, seckillConfig)

	ctx := context.Background()

	// 1. 预加载秒杀活动
	fmt.Println("=== 预加载秒杀活动 ===")
	activity := &seckill.SeckillActivity{
		ProductID:   1001,
		ProductName: "iPhone 15 Pro",
		Price:       8999.00,
		Stock:       100,
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(2 * time.Hour),
		Status:      "active",
	}

	err = seckillCache.PreloadSeckillActivity(ctx, activity)
	if err != nil {
		log.Fatalf("Failed to preload seckill activity: %v", err)
	}
	fmt.Printf("成功预加载秒杀活动: %+v\n", activity)

	// 2. 获取库存
	fmt.Println("\n=== 获取库存 ===")
	stock, err := seckillCache.GetStock(ctx, 1001)
	if err != nil {
		log.Fatalf("Failed to get stock: %v", err)
	}
	fmt.Printf("当前库存: %d\n", stock)

	// 3. 模拟用户购买
	fmt.Println("\n=== 模拟用户购买 ===")

	// 用户 2001 购买
	fmt.Println("用户 2001 尝试购买...")
	err = seckillCache.SeckillPurchase(ctx, 1001, 2001, 1)
	if err != nil {
		fmt.Printf("用户 2001 购买失败: %v\n", err)
	} else {
		fmt.Println("用户 2001 购买成功!")
	}

	// 用户 2002 购买
	fmt.Println("用户 2002 尝试购买...")
	err = seckillCache.SeckillPurchase(ctx, 1001, 2002, 1)
	if err != nil {
		fmt.Printf("用户 2002 购买失败: %v\n", err)
	} else {
		fmt.Println("用户 2002 购买成功!")
	}

	// 用户 2001 再次购买（应该失败）
	fmt.Println("用户 2001 再次尝试购买...")
	err = seckillCache.SeckillPurchase(ctx, 1001, 2001, 1)
	if err != nil {
		fmt.Printf("用户 2001 再次购买失败: %v\n", err)
	} else {
		fmt.Println("用户 2001 再次购买成功!")
	}

	// 4. 检查购买后的库存
	fmt.Println("\n=== 检查购买后的库存 ===")
	stock, err = seckillCache.GetStock(ctx, 1001)
	if err != nil {
		log.Fatalf("Failed to get stock: %v", err)
	}
	fmt.Printf("购买后库存: %d\n", stock)

	// 5. 检查用户购买状态
	fmt.Println("\n=== 检查用户购买状态 ===")

	purchased, err := seckillCache.IsUserPurchased(ctx, 1001, 2001)
	if err != nil {
		log.Fatalf("Failed to check user purchase status: %v", err)
	}
	fmt.Printf("用户 2001 是否已购买: %v\n", purchased)

	purchased, err = seckillCache.IsUserPurchased(ctx, 1001, 2003)
	if err != nil {
		log.Fatalf("Failed to check user purchase status: %v", err)
	}
	fmt.Printf("用户 2003 是否已购买: %v\n", purchased)

	// 6. 获取用户购买信息
	fmt.Println("\n=== 获取用户购买信息 ===")

	purchaseInfo, err := seckillCache.GetUserPurchaseInfo(ctx, 1001, 2001)
	if err != nil {
		fmt.Printf("获取用户 2001 购买信息失败: %v\n", err)
	} else {
		fmt.Printf("用户 2001 购买信息: %+v\n", purchaseInfo)
	}

	// 7. 获取购买用户数量
	fmt.Println("\n=== 获取购买用户数量 ===")

	count, err := seckillCache.GetPurchaseUserCount(ctx, 1001)
	if err != nil {
		log.Fatalf("Failed to get purchase user count: %v", err)
	}
	fmt.Printf("购买用户数量: %d\n", count)

	// 8. 获取秒杀活动信息
	fmt.Println("\n=== 获取秒杀活动信息 ===")

	activityInfo, err := seckillCache.GetSeckillActivity(ctx, 1001)
	if err != nil {
		log.Fatalf("Failed to get seckill activity: %v", err)
	}
	fmt.Printf("秒杀活动信息: %+v\n", activityInfo)

	// 9. 清理测试数据
	fmt.Println("\n=== 清理测试数据 ===")

	err = seckillCache.CleanupSeckillData(ctx, 1001)
	if err != nil {
		log.Fatalf("Failed to cleanup seckill data: %v", err)
	}
	fmt.Println("测试数据清理完成")
}

// 并发测试示例
func ConcurrentSeckillExample() {
	fmt.Println("\n=== 并发秒杀测试 ===")

	// 创建 Redis 配置
	redisConfig := &redis.Config{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     20,
		MinIdleConns: 10,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	// 创建 Redis 客户端
	redisClient, err := redis.NewClient(redisConfig, nil)
	if err != nil {
		log.Fatalf("Failed to create redis client: %v", err)
	}
	defer redisClient.Close()

	// 创建秒杀配置
	seckillConfig := &seckill.Config{
		StockKeyPrefix: "seckill:stock:",
		UserKeyPrefix:  "seckill:users:",
		LockKeyPrefix:  "seckill:lock:",
		DefaultTTL:     3600 * time.Second,
	}

	// 创建秒杀缓存
	seckillCache := seckill.NewSeckillCache(redisClient, seckillConfig)

	ctx := context.Background()

	// 预加载秒杀活动（库存设为 10）
	activity := &seckill.SeckillActivity{
		ProductID:   2001,
		ProductName: "限量商品",
		Price:       999.00,
		Stock:       10,
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(1 * time.Hour),
		Status:      "active",
	}

	err = seckillCache.PreloadSeckillActivity(ctx, activity)
	if err != nil {
		log.Fatalf("Failed to preload seckill activity: %v", err)
	}

	// 创建 20 个并发用户
	userCount := 20
	successCount := 0
	failCount := 0

	// 使用 channel 来收集结果
	results := make(chan bool, userCount)

	// 启动并发购买
	for i := 0; i < userCount; i++ {
		go func(userID int) {
			err := seckillCache.SeckillPurchase(ctx, 2001, int64(userID+3000), 1)
			if err != nil {
				fmt.Printf("用户 %d 购买失败: %v\n", userID+3000, err)
				results <- false
			} else {
				fmt.Printf("用户 %d 购买成功!\n", userID+3000)
				results <- true
			}
		}(i)
	}

	// 收集结果
	for i := 0; i < userCount; i++ {
		if <-results {
			successCount++
		} else {
			failCount++
		}
	}

	fmt.Printf("\n并发测试结果:\n")
	fmt.Printf("成功购买: %d 人\n", successCount)
	fmt.Printf("购买失败: %d 人\n", failCount)

	// 检查最终库存
	finalStock, err := seckillCache.GetStock(ctx, 2001)
	if err != nil {
		log.Fatalf("Failed to get final stock: %v", err)
	}
	fmt.Printf("最终库存: %d\n", finalStock)

	// 验证数据一致性
	purchaseCount, err := seckillCache.GetPurchaseUserCount(ctx, 2001)
	if err != nil {
		log.Fatalf("Failed to get purchase count: %v", err)
	}
	fmt.Printf("购买用户数量: %d\n", purchaseCount)

	expectedStock := int64(10) - purchaseCount
	if finalStock == expectedStock {
		fmt.Println("✅ 数据一致性检查通过")
	} else {
		fmt.Printf("❌ 数据一致性检查失败: 期望库存 %d，实际库存 %d\n", expectedStock, finalStock)
	}

	// 清理测试数据
	err = seckillCache.CleanupSeckillData(ctx, 2001)
	if err != nil {
		log.Fatalf("Failed to cleanup seckill data: %v", err)
	}
	fmt.Println("并发测试数据清理完成")
}

// 运行所有示例
func RunAllExamples() {
	fmt.Println("开始运行缓存服务示例...")

	// 基础秒杀示例
	SeckillExample()

	// 等待一下再运行并发测试
	time.Sleep(2 * time.Second)

	// 并发秒杀示例
	ConcurrentSeckillExample()

	fmt.Println("\n所有示例运行完成!")
}
