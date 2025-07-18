package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cache-service/api/rest"
	"cache-service/internal/config"
	"cache-service/internal/redis"
	"cache-service/internal/seckill"
	"cache-service/internal/service"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("./config")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建 Redis 配置
	redisConfig := &redis.Config{
		Host:         cfg.Redis.Host,
		Port:         cfg.Redis.Port,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	}

	// 创建秒杀配置
	seckillConfig := &seckill.Config{
		StockKeyPrefix: cfg.Seckill.StockKeyPrefix,
		UserKeyPrefix:  cfg.Seckill.UserKeyPrefix,
		LockKeyPrefix:  cfg.Seckill.LockKeyPrefix,
		DefaultTTL:     time.Duration(cfg.Seckill.DefaultTTL) * time.Second,
	}

	// 创建服务配置
	serviceConfig := &service.Config{
		Redis:   redisConfig,
		Seckill: seckillConfig,
	}

	// 创建缓存服务
	cacheService, err := service.NewCacheService(serviceConfig)
	if err != nil {
		log.Fatalf("Failed to create cache service: %v", err)
	}
	defer cacheService.Close()

	// 创建 REST API 处理程序
	handler := rest.NewHandler(cacheService)
	router := rest.SetupRouter(handler)

	// 启动 HTTP 服务器
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	// 在 goroutine 中启动服务器
	go func() {
		log.Printf("Starting HTTP server on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号以优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 创建一个超时上下文用于关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭服务器
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
