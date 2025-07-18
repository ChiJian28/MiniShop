package service

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"seckill-service/internal/config"
	"seckill-service/internal/flowcontrol"
	"seckill-service/internal/mq"
	"seckill-service/internal/seckill"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// 秒杀服务
type SeckillService struct {
	config         *config.Config
	redisClient    *redis.Client
	seckillCore    *seckill.SeckillCore
	messageQueue   mq.MessageQueue
	limiter        flowcontrol.Limiter
	circuitBreaker *flowcontrol.CircuitBreaker
	requestQueue   *flowcontrol.RequestQueue
	logger         *logrus.Logger

	// 统计信息
	stats ServiceStats
}

// 服务统计信息
type ServiceStats struct {
	TotalRequests       int64
	SuccessRequests     int64
	FailedRequests      int64
	RateLimitedRequests int64
	CircuitBreakerTrips int64
	QueueFullRequests   int64
	SystemBusyRequests  int64
}

// 创建秒杀服务
func NewSeckillService(cfg *config.Config, logger *logrus.Logger) (*SeckillService, error) {
	// 创建 Redis 客户端
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

	// 测试 Redis 连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	// 创建秒杀核心服务
	seckillCore := seckill.NewSeckillCore(redisClient, logger)
	if err := seckillCore.InitScripts(ctx); err != nil {
		return nil, fmt.Errorf("failed to init seckill scripts: %w", err)
	}

	// 创建消息队列
	var messageQueue mq.MessageQueue
	var err error

	// 根据配置选择消息队列
	if cfg.RabbitMQ.URL != "" {
		mqConfig := &mq.RabbitMQConfig{
			URL:        cfg.RabbitMQ.URL,
			Exchange:   cfg.RabbitMQ.Exchange,
			Queue:      cfg.RabbitMQ.Queue,
			RoutingKey: cfg.RabbitMQ.RoutingKey,
			Durable:    cfg.RabbitMQ.Durable,
			AutoDelete: cfg.RabbitMQ.AutoDelete,
		}
		messageQueue, err = mq.NewRabbitMQProducer(mqConfig, logger)
	} else if len(cfg.Kafka.Brokers) > 0 {
		kafkaConfig := &mq.KafkaConfig{
			Brokers:   cfg.Kafka.Brokers,
			Topic:     cfg.Kafka.Topic,
			Partition: cfg.Kafka.Partition,
			Timeout:   cfg.Kafka.Timeout,
		}
		messageQueue = mq.NewKafkaProducer(kafkaConfig, logger)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create message queue: %w", err)
	}

	// 创建限流器
	limiter := flowcontrol.NewTokenBucketLimiter(
		float64(cfg.Seckill.RateLimit.RequestsPerSecond),
		cfg.Seckill.RateLimit.BurstSize,
		logger,
	)

	// 创建熔断器
	circuitBreakerConfig := flowcontrol.CircuitBreakerConfig{
		MaxRequests: uint32(cfg.Seckill.CircuitBreaker.HalfOpenRequests),
		Interval:    60 * time.Second,
		Timeout:     cfg.Seckill.CircuitBreaker.RecoveryTimeout,
		ReadyToTrip: func(counts flowcontrol.Counts) bool {
			return counts.Requests >= 3 && counts.TotalFailures >= uint32(cfg.Seckill.CircuitBreaker.FailureThreshold)
		},
		OnStateChange: func(name string, from, to flowcontrol.CircuitBreakerState) {
			logger.Infof("Circuit breaker %s state changed: %s -> %s", name, from, to)
		},
	}
	circuitBreaker := flowcontrol.NewCircuitBreaker("seckill", circuitBreakerConfig, logger)

	service := &SeckillService{
		config:         cfg,
		redisClient:    redisClient,
		seckillCore:    seckillCore,
		messageQueue:   messageQueue,
		limiter:        limiter,
		circuitBreaker: circuitBreaker,
		logger:         logger,
	}

	// 创建请求队列
	requestQueue := flowcontrol.NewRequestQueue(
		cfg.Seckill.QueueSize,
		10, // 10 个工作协程
		service.processQueueItem,
		logger,
	)
	service.requestQueue = requestQueue

	logger.Info("Seckill service created successfully")
	return service, nil
}

// 启动服务
func (s *SeckillService) Start(ctx context.Context) error {
	// 启动请求队列
	s.requestQueue.Start(ctx)

	s.logger.Info("Seckill service started")
	return nil
}

// 停止服务
func (s *SeckillService) Stop() error {
	if s.redisClient != nil {
		s.redisClient.Close()
	}
	if s.messageQueue != nil {
		s.messageQueue.Close()
	}

	s.logger.Info("Seckill service stopped")
	return nil
}

// 秒杀请求处理
func (s *SeckillService) ProcessSeckill(ctx context.Context, req *seckill.SeckillRequest) (*seckill.SeckillResult, error) {
	// 检查系统负载
	if s.isSystemBusy() {
		s.stats.SystemBusyRequests++
		return &seckill.SeckillResult{
			Code:    seckill.ResultSystemBusy,
			Message: s.config.Seckill.Degradation.ResponseMessage,
			Success: false,
		}, nil
	}

	// 限流检查
	if !s.limiter.Allow() {
		s.stats.RateLimitedRequests++
		return &seckill.SeckillResult{
			Code:    seckill.ResultSystemBusy,
			Message: "请求过于频繁，请稍后重试",
			Success: false,
		}, nil
	}

	// 熔断器检查
	result, err := s.circuitBreaker.ExecuteWithContext(ctx, func(ctx context.Context) (interface{}, error) {
		return s.executeSeckill(ctx, req)
	})

	if err != nil {
		if err == flowcontrol.ErrCircuitBreakerOpen {
			s.stats.CircuitBreakerTrips++
			return &seckill.SeckillResult{
				Code:    seckill.ResultSystemBusy,
				Message: "系统繁忙，请稍后重试",
				Success: false,
			}, nil
		}
		return nil, err
	}

	return result.(*seckill.SeckillResult), nil
}

// 执行秒杀逻辑
func (s *SeckillService) executeSeckill(ctx context.Context, req *seckill.SeckillRequest) (*seckill.SeckillResult, error) {
	s.stats.TotalRequests++

	// 执行秒杀核心逻辑
	result, err := s.seckillCore.ExecuteSeckill(ctx, req)
	if err != nil {
		s.stats.FailedRequests++
		s.logger.Errorf("Seckill execution failed: %v", err)
		return &seckill.SeckillResult{
			Code:    seckill.ResultSystemError,
			Message: "系统错误",
			Success: false,
		}, nil
	}

	// 如果秒杀成功，发送消息到队列
	if result.Success {
		s.stats.SuccessRequests++

		// 生成订单ID
		orderID := s.seckillCore.GenerateOrderID(req.ProductID, req.UserID)
		result.OrderID = orderID

		// 发送订单消息
		if err := s.sendOrderMessage(ctx, req, orderID); err != nil {
			s.logger.Errorf("Failed to send order message: %v", err)
			// 不影响秒杀结果，只记录错误
		}

		// 发送库存更新消息
		if err := s.sendStockUpdateMessage(ctx, req.ProductID, result.RemainingStock); err != nil {
			s.logger.Errorf("Failed to send stock update message: %v", err)
		}

		s.logger.Infof("Seckill success: user=%d, product=%d, order=%s, remaining_stock=%d",
			req.UserID, req.ProductID, orderID, result.RemainingStock)
	} else {
		s.stats.FailedRequests++
		s.logger.Debugf("Seckill failed: user=%d, product=%d, reason=%s",
			req.UserID, req.ProductID, result.Message)
	}

	return result, nil
}

// 发送订单消息
func (s *SeckillService) sendOrderMessage(ctx context.Context, req *seckill.SeckillRequest, orderID string) error {
	if s.messageQueue == nil {
		return nil
	}

	// 获取商品价格（这里简化处理，实际应该从数据库获取）
	price := 99.99 // 默认价格

	message := mq.NewSeckillOrderMessage(
		orderID,
		req.ProductID,
		req.UserID,
		req.Quantity,
		price,
		s.generateTraceID(),
	)

	return s.messageQueue.SendSeckillOrderMessage(ctx, message)
}

// 发送库存更新消息
func (s *SeckillService) sendStockUpdateMessage(ctx context.Context, productID, remainingStock int64) error {
	if s.messageQueue == nil {
		return nil
	}

	message := mq.NewStockUpdateMessage(
		productID,
		remainingStock,
		s.generateTraceID(),
	)

	return s.messageQueue.SendStockUpdateMessage(ctx, message)
}

// 生成追踪ID
func (s *SeckillService) generateTraceID() string {
	return fmt.Sprintf("trace_%d", time.Now().UnixNano())
}

// 检查系统是否繁忙
func (s *SeckillService) isSystemBusy() bool {
	if !s.config.Seckill.Degradation.Enable {
		return false
	}

	// 检查 CPU 使用率
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// 简化的系统负载检查
	// 实际应该使用更精确的系统监控指标
	return false
}

// 处理队列项
func (s *SeckillService) processQueueItem(ctx context.Context, item *flowcontrol.QueueItem) (interface{}, error) {
	req, ok := item.Request.(*seckill.SeckillRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type")
	}

	return s.executeSeckill(ctx, req)
}

// 异步处理秒杀请求
func (s *SeckillService) ProcessSeckillAsync(ctx context.Context, req *seckill.SeckillRequest) error {
	// 检查系统负载
	if s.isSystemBusy() {
		s.stats.SystemBusyRequests++
		return fmt.Errorf("system is busy")
	}

	// 限流检查
	if !s.limiter.Allow() {
		s.stats.RateLimitedRequests++
		return fmt.Errorf("rate limited")
	}

	// 提交到请求队列
	requestID := fmt.Sprintf("seckill_%d_%d_%d", req.ProductID, req.UserID, time.Now().UnixNano())

	err := s.requestQueue.SubmitAsync(ctx, requestID, req, 30*time.Second, func(result interface{}, err error) {
		if err != nil {
			s.logger.Errorf("Async seckill failed: %v", err)
			return
		}

		seckillResult := result.(*seckill.SeckillResult)
		s.logger.Infof("Async seckill completed: %+v", seckillResult)
	})

	if err != nil {
		if err == flowcontrol.ErrQueueFull {
			s.stats.QueueFullRequests++
		}
		return err
	}

	return nil
}

// 预热活动
func (s *SeckillService) PrewarmActivity(ctx context.Context, activity *seckill.SeckillActivity) error {
	return s.seckillCore.PrewarmActivity(ctx, activity)
}

// 获取秒杀统计信息
func (s *SeckillService) GetSeckillStats(ctx context.Context, productID int64) (*seckill.SeckillStats, error) {
	return s.seckillCore.GetSeckillStats(ctx, productID)
}

// 检查用户购买状态
func (s *SeckillService) IsUserPurchased(ctx context.Context, productID, userID int64) (bool, error) {
	return s.seckillCore.IsUserPurchased(ctx, productID, userID)
}

// 获取用户购买信息
func (s *SeckillService) GetUserPurchaseInfo(ctx context.Context, productID, userID int64) (*seckill.UserPurchaseInfo, error) {
	// 这里需要实现获取用户购买信息的逻辑
	// 暂时返回模拟数据
	return &seckill.UserPurchaseInfo{
		UserID:       userID,
		ProductID:    productID,
		Quantity:     1,
		PurchaseTime: time.Now(),
		Status:       "success",
	}, nil
}

// 清理活动数据
func (s *SeckillService) CleanupActivity(ctx context.Context, productID int64) error {
	return s.seckillCore.CleanupActivity(ctx, productID)
}

// 获取服务统计信息
func (s *SeckillService) GetServiceStats() ServiceStats {
	return s.stats
}

// 获取队列统计信息
func (s *SeckillService) GetQueueStats() flowcontrol.QueueStats {
	return s.requestQueue.GetStats()
}

// 获取熔断器状态
func (s *SeckillService) GetCircuitBreakerState() flowcontrol.CircuitBreakerState {
	return s.circuitBreaker.State()
}

// 获取限流器状态
func (s *SeckillService) GetLimiterTokens() int {
	if tokenBucket, ok := s.limiter.(*flowcontrol.TokenBucketLimiter); ok {
		return tokenBucket.GetTokens()
	}
	return 0
}

// 健康检查
func (s *SeckillService) HealthCheck(ctx context.Context) error {
	// 检查 Redis 连接
	if err := s.redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	// 检查消息队列
	if s.messageQueue != nil {
		// 这里需要根据具体的消息队列实现健康检查
		// 暂时跳过
	}

	return nil
}
