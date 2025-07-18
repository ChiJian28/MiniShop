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

	"seckill-service/api/rest"
	"seckill-service/internal/config"
	"seckill-service/internal/service"

	"github.com/sirupsen/logrus"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("./config")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	logger := logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

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

	// 设置日志输出
	if cfg.Log.File != "" {
		file, err := os.OpenFile(cfg.Log.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
		defer file.Close()
		logger.SetOutput(file)
	}

	logger.Info("Starting seckill service...")

	// 创建秒杀服务
	seckillService, err := service.NewSeckillService(cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to create seckill service: %v", err)
	}
	defer seckillService.Stop()

	// 启动服务
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := seckillService.Start(ctx); err != nil {
		logger.Fatalf("Failed to start seckill service: %v", err)
	}

	// 创建 REST API 处理程序
	handler := rest.NewHandler(seckillService)
	router := rest.SetupRouter(handler)

	// 启动 HTTP 服务器
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	// 在 goroutine 中启动服务器
	go func() {
		logger.Infof("Starting HTTP server on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// 等待中断信号以优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// 取消上下文
	cancel()

	// 创建一个超时上下文用于关闭服务器
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 关闭 HTTP 服务器
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}
