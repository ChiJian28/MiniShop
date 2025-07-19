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

	"order-service/internal/compensation"
	"order-service/internal/config"
	"order-service/internal/database"
	"order-service/internal/handler"
	"order-service/internal/mq"
	"order-service/internal/router"
	"order-service/internal/service"

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

	// 初始化 Redis 客户端
	redisClient := database.NewRedisClient(&cfg.Redis)

	// 初始化订单服务
	orderService := service.NewOrderService(cfg, db, redisClient, logger)

	// 初始化失败补偿管理器
	compensationManager := compensation.NewCompensationManager(cfg, db, orderService, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go compensationManager.Start(ctx) // 启动定时任务

	// 初始化消息队列消费者
	var consumers []interface{}

	// RabbitMQ 消费者 (简化处理，假设总是启用)
	rabbitConsumer, err := mq.NewRabbitMQConsumer(&cfg.RabbitMQ, orderService, logger)
	if err != nil {
		logger.WithError(err).Fatal("初始化RabbitMQ消费者失败")
	}
	consumers = append(consumers, rabbitConsumer)

	// Kafka 消费者 (简化处理，假设总是启用)
	kafkaConsumer := mq.NewKafkaConsumer(&cfg.Kafka, orderService, logger)
	consumers = append(consumers, kafkaConsumer)

	// 启动消费者 (暂时注释掉，因为接口不明确)
	// for _, consumer := range consumers {
	// 	go func(c interface{}) {
	// 		// 这里需要根据实际的消费者接口来实现
	// 		// if starter, ok := c.(interface{ Start(context.Context) error }); ok {
	// 		// 	if err := starter.Start(context.Background()); err != nil {
	// 		// 		logger.WithError(err).Error("消费者启动失败")
	// 		// 	}
	// 		// }
	// 	}(consumer)
	// }

	// 初始化HTTP处理器和路由
	orderHandler := handler.NewOrderHandler(orderService)
	r := router.SetupRouter(orderHandler)

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

	// 停止消费者 (暂时注释掉)
	// for _, consumer := range consumers {
	// 	// 这里需要根据实际的消费者接口来实现
	// 	// if stopper, ok := consumer.(interface{ Stop() }); ok {
	// 	// 	stopper.Stop()
	// 	// }
	// }

	// 关闭HTTP服务器
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("服务器关闭失败")
	}

	logger.Info("服务器已关闭")
}
