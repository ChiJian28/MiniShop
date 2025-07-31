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

	"inventory-service/internal/config"
	"inventory-service/internal/database"
	"inventory-service/internal/handler"
	"inventory-service/internal/health"
	"inventory-service/internal/router"
	"inventory-service/internal/service"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("./config")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})

	// 初始化数据库
	db, err := database.NewDatabase(&cfg.Database)
	if err != nil {
		logger.WithError(err).Fatal("初始化数据库失败")
	}
	defer db.Close()

	// 自动迁移数据库表
	if err := db.AutoMigrate(); err != nil {
		logger.WithError(err).Fatal("数据库迁移失败")
	}

	// 初始化 Redis 客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})
	defer redisClient.Close()

	// 测试 Redis 连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.WithError(err).Fatal("Redis连接失败")
	}

	// 初始化库存服务
	inventoryService := service.NewInventoryService(cfg, db, redisClient, logger)

	// 初始化健康检查管理器
	healthChecker := health.NewHealthChecker(cfg, inventoryService, logger)

	// 启动健康检查
	healthCtx, healthCancel := context.WithCancel(context.Background())
	defer healthCancel()

	go func() {
		if err := healthChecker.Start(healthCtx); err != nil {
			logger.WithError(err).Error("健康检查启动失败")
		}
	}()
	defer healthChecker.Stop()

	// 初始化HTTP处理器和路由
	inventoryHandler := handler.NewInventoryHandler(inventoryService, healthChecker)
	r := router.SetupRouter(inventoryHandler)

	// 启动HTTP服务器
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	// 在 goroutine 中启动服务器
	go func() {
		logger.WithField("port", cfg.Server.Port).Info("启动HTTP服务器")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("HTTP服务器启动失败")
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭服务器...")

	// 关闭HTTP服务器
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("服务器关闭失败")
	}

	logger.Info("服务器已关闭")
}
