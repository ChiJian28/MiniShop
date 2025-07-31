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

	"api-gateway/internal/config"
	"api-gateway/internal/handler"
	"api-gateway/internal/middleware"
	"api-gateway/internal/proxy"
	"api-gateway/internal/router"

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
	logger := initLogger(cfg)
	logger.Info("正在启动 API Gateway...")

	// 初始化 Redis 客户端
	redisClient := initRedis(cfg, logger)
	defer redisClient.Close()

	// 初始化中间件
	rateLimiter := middleware.NewRateLimiter(&cfg.RateLimit, redisClient, logger)
	authMiddleware := middleware.NewAuthMiddleware(&cfg.Auth, logger)
	corsMiddleware := middleware.NewCORSMiddleware(&cfg.CORS)

	// 初始化服务代理
	serviceProxy := proxy.NewServiceProxy(cfg, logger)

	// 初始化处理器
	gatewayHandler := handler.NewGatewayHandler(serviceProxy, rateLimiter, authMiddleware)

	// 设置路由
	mainRouter := router.SetupRouter(cfg, gatewayHandler, serviceProxy, corsMiddleware, rateLimiter, authMiddleware)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mainRouter,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// 启动监控服务器（如果启用）
	var monitoringServer *http.Server
	if cfg.Monitoring.Enable {
		monitoringRouter := router.SetupMonitoringRouter(cfg)
		monitoringServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Monitoring.MetricsPort),
			Handler: monitoringRouter,
		}

		go func() {
			logger.WithField("port", cfg.Monitoring.MetricsPort).Info("启动监控服务器")
			if err := monitoringServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.WithError(err).Error("监控服务器启动失败")
			}
		}()
	}

	// 在 goroutine 中启动主服务器
	go func() {
		logger.WithField("port", cfg.Server.Port).Info("启动 API Gateway 服务器")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("API Gateway 服务器启动失败")
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭 API Gateway...")

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 关闭主服务器
	if err := server.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("主服务器关闭失败")
	}

	// 关闭监控服务器
	if monitoringServer != nil {
		if err := monitoringServer.Shutdown(ctx); err != nil {
			logger.WithError(err).Error("监控服务器关闭失败")
		}
	}

	logger.Info("API Gateway 已关闭")
}

// initLogger 初始化日志
func initLogger(cfg *config.Config) *logrus.Logger {
	logger := logrus.New()

	// 设置日志级别
	switch cfg.Log.Level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	// 设置日志格式
	if cfg.Log.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	// 设置日志输出文件（如果配置了）
	if cfg.Log.File != "" {
		file, err := os.OpenFile(cfg.Log.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logger.WithError(err).Warn("无法打开日志文件，使用标准输出")
		} else {
			logger.SetOutput(file)
		}
	}

	return logger
}

// initRedis 初始化Redis客户端
func initRedis(cfg *config.Config, logger *logrus.Logger) *redis.Client {
	client := redis.NewClient(&redis.Options{
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

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.WithError(err).Fatal("Redis连接失败")
	}

	logger.Info("Redis连接成功")
	return client
}
