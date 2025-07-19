package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"order-service/internal/config"
	"order-service/internal/database"
	"order-service/internal/model"
	"order-service/internal/mq"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// 订单服务
type OrderService struct {
	config      *config.Config
	db          *database.Database
	redisClient *redis.Client
	logger      *logrus.Logger

	// 统计信息
	stats ServiceStats
}

// 服务统计信息
type ServiceStats struct {
	TotalOrders       int64
	SuccessOrders     int64
	FailedOrders      int64
	DuplicateOrders   int64
	CompensationTasks int64
}

// 创建订单服务
func NewOrderService(cfg *config.Config, db *database.Database, redisClient *redis.Client, logger *logrus.Logger) *OrderService {
	return &OrderService{
		config:      cfg,
		db:          db,
		redisClient: redisClient,
		logger:      logger,
	}
}

// 实现消息处理器接口
func (s *OrderService) HandleSeckillOrder(ctx context.Context, message *mq.SeckillOrderMessage) error {
	s.stats.TotalOrders++

	s.logger.WithFields(logrus.Fields{
		"order_id":   message.OrderID,
		"user_id":    message.UserID,
		"product_id": message.ProductID,
		"trace_id":   message.TraceID,
	}).Info("Processing seckill order message")

	// 创建订单请求
	request := &model.CreateOrderRequest{
		OrderID:     message.OrderID,
		UserID:      message.UserID,
		ProductID:   message.ProductID,
		ProductName: fmt.Sprintf("Product-%d", message.ProductID), // 简化处理，实际应该从产品服务获取
		Quantity:    message.Quantity,
		Price:       message.Price,
		OrderType:   model.OrderTypeSeckill,
		TraceID:     message.TraceID,
	}

	// 创建订单
	_, err := s.CreateOrder(ctx, request)
	if err != nil {
		// 如果是重复订单，不算作失败
		if errors.Is(err, ErrDuplicateOrder) {
			s.stats.DuplicateOrders++
			s.logger.WithFields(logrus.Fields{
				"order_id":   message.OrderID,
				"user_id":    message.UserID,
				"product_id": message.ProductID,
			}).Warn("Duplicate order detected, skipping")
			return nil
		}

		s.stats.FailedOrders++

		// 记录失败信息用于补偿
		if err := s.recordOrderFailure(ctx, message, err); err != nil {
			s.logger.Errorf("Failed to record order failure: %v", err)
		}

		return fmt.Errorf("failed to create order: %w", err)
	}

	s.stats.SuccessOrders++
	return nil
}

// 处理库存更新消息
func (s *OrderService) HandleStockUpdate(ctx context.Context, message *mq.StockUpdateMessage) error {
	s.logger.WithFields(logrus.Fields{
		"product_id":      message.ProductID,
		"remaining_stock": message.RemainingStock,
		"trace_id":        message.TraceID,
	}).Debug("Processing stock update message")

	// 这里可以实现库存更新相关的业务逻辑
	// 例如：更新本地缓存、触发补货提醒等

	return nil
}

// 处理用户通知消息
func (s *OrderService) HandleUserNotify(ctx context.Context, message *mq.UserNotifyMessage) error {
	s.logger.WithFields(logrus.Fields{
		"user_id":     message.UserID,
		"product_id":  message.ProductID,
		"notify_type": message.NotifyType,
		"trace_id":    message.TraceID,
	}).Debug("Processing user notify message")

	// 这里可以实现用户通知相关的业务逻辑
	// 例如：发送短信、推送通知等

	return nil
}

// 自定义错误
var (
	ErrDuplicateOrder = errors.New("duplicate order")
	ErrInvalidOrder   = errors.New("invalid order")
)

// 创建订单
func (s *OrderService) CreateOrder(ctx context.Context, request *model.CreateOrderRequest) (*model.OrderResponse, error) {
	// 参数验证
	if err := s.validateOrderRequest(request); err != nil {
		return nil, fmt.Errorf("invalid order request: %w", err)
	}

	// 开始事务
	tx := s.db.BeginTx()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// 幂等性检查
	if err := s.checkIdempotency(ctx, tx, request); err != nil {
		tx.Rollback()
		return nil, err
	}

	// 创建订单
	order := &model.Order{
		OrderID:     request.OrderID,
		UserID:      request.UserID,
		ProductID:   request.ProductID,
		ProductName: request.ProductName,
		Quantity:    request.Quantity,
		Price:       request.Price,
		TotalAmount: request.Price * float64(request.Quantity),
		Status:      model.OrderStatusPending,
		OrderType:   request.OrderType,
		TraceID:     request.TraceID,
		ExpiredAt:   s.calculateExpiredAt(),
	}

	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// 创建订单明细
	orderItem := &model.OrderItem{
		OrderID:     request.OrderID,
		ProductID:   request.ProductID,
		ProductName: request.ProductName,
		Quantity:    request.Quantity,
		Price:       request.Price,
		TotalAmount: request.Price * float64(request.Quantity),
	}

	if err := tx.Create(orderItem).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create order item: %w", err)
	}

	// 记录幂等性
	idempotency := &model.OrderIdempotency{
		UserID:    request.UserID,
		ProductID: request.ProductID,
		OrderID:   request.OrderID,
		TraceID:   request.TraceID,
		Status:    model.OrderStatusPending,
	}

	if err := tx.Create(idempotency).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create idempotency record: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 异步更新统计
	go s.updateOrderStats(order)

	// 缓存订单信息
	s.cacheOrder(ctx, order)

	s.logger.WithFields(logrus.Fields{
		"order_id":   order.OrderID,
		"user_id":    order.UserID,
		"product_id": order.ProductID,
		"amount":     order.TotalAmount,
	}).Info("Order created successfully")

	return s.orderToResponse(order), nil
}

// 验证订单请求
func (s *OrderService) validateOrderRequest(request *model.CreateOrderRequest) error {
	if request.OrderID == "" {
		return fmt.Errorf("order_id is required")
	}
	if request.UserID <= 0 {
		return fmt.Errorf("user_id must be positive")
	}
	if request.ProductID <= 0 {
		return fmt.Errorf("product_id must be positive")
	}
	if request.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if request.Price < 0 {
		return fmt.Errorf("price cannot be negative")
	}
	if request.OrderType == "" {
		return fmt.Errorf("order_type is required")
	}
	return nil
}

// 幂等性检查
func (s *OrderService) checkIdempotency(ctx context.Context, tx *gorm.DB, request *model.CreateOrderRequest) error {
	var existing model.OrderIdempotency
	err := tx.Where("user_id = ? AND product_id = ?", request.UserID, request.ProductID).
		First(&existing).Error

	if err == nil {
		// 找到了已存在的记录
		return ErrDuplicateOrder
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check idempotency: %w", err)
	}

	return nil
}

// 计算订单过期时间
func (s *OrderService) calculateExpiredAt() *time.Time {
	expiredAt := time.Now().Add(s.config.Order.OrderTimeout)
	return &expiredAt
}

// 缓存订单信息
func (s *OrderService) cacheOrder(ctx context.Context, order *model.Order) {
	key := fmt.Sprintf("order:%s", order.OrderID)
	data, err := json.Marshal(order)
	if err != nil {
		s.logger.Errorf("Failed to marshal order for cache: %v", err)
		return
	}

	err = s.redisClient.Set(ctx, key, data, s.config.Order.OrderTimeout).Err()
	if err != nil {
		s.logger.Errorf("Failed to cache order: %v", err)
	}
}

// 记录订单失败信息
func (s *OrderService) recordOrderFailure(ctx context.Context, message *mq.SeckillOrderMessage, err error) error {
	messageData, _ := json.Marshal(message)

	failure := &model.OrderFailure{
		OrderID:     message.OrderID,
		UserID:      message.UserID,
		ProductID:   message.ProductID,
		MessageData: string(messageData),
		ErrorMsg:    err.Error(),
		RetryCount:  0,
		MaxRetries:  s.config.Order.Retry.MaxAttempts,
		NextRetryAt: s.calculateNextRetryTime(0),
		Status:      "pending",
	}

	if err := s.db.GetDB().Create(failure).Error; err != nil {
		return fmt.Errorf("failed to record order failure: %w", err)
	}

	s.stats.CompensationTasks++
	return nil
}

// 计算下次重试时间
func (s *OrderService) calculateNextRetryTime(retryCount int) *time.Time {
	interval := s.config.Order.Retry.InitialInterval
	for i := 0; i < retryCount; i++ {
		interval = time.Duration(float64(interval) * s.config.Order.Retry.Multiplier)
		if interval > s.config.Order.Retry.MaxInterval {
			interval = s.config.Order.Retry.MaxInterval
			break
		}
	}

	nextRetry := time.Now().Add(interval)
	return &nextRetry
}

// 更新订单统计
func (s *OrderService) updateOrderStats(order *model.Order) {
	date := time.Now().Truncate(24 * time.Hour)

	var stats model.OrderStats
	err := s.db.GetDB().Where("date = ?", date).First(&stats).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		stats = model.OrderStats{
			Date:        date,
			TotalOrders: 1,
			TotalAmount: order.TotalAmount,
		}
		if order.OrderType == model.OrderTypeSeckill {
			stats.SeckillOrders = 1
		}
		s.db.GetDB().Create(&stats)
	} else if err == nil {
		updates := map[string]interface{}{
			"total_orders": gorm.Expr("total_orders + ?", 1),
			"total_amount": gorm.Expr("total_amount + ?", order.TotalAmount),
		}
		if order.OrderType == model.OrderTypeSeckill {
			updates["seckill_orders"] = gorm.Expr("seckill_orders + ?", 1)
		}
		s.db.GetDB().Model(&stats).Updates(updates)
	}
}

// 转换为响应对象
func (s *OrderService) orderToResponse(order *model.Order) *model.OrderResponse {
	return &model.OrderResponse{
		OrderID:     order.OrderID,
		UserID:      order.UserID,
		ProductID:   order.ProductID,
		ProductName: order.ProductName,
		Quantity:    order.Quantity,
		Price:       order.Price,
		TotalAmount: order.TotalAmount,
		Status:      order.Status,
		OrderType:   order.OrderType,
		CreatedAt:   order.CreatedAt,
		ExpiredAt:   order.ExpiredAt,
		PaidAt:      order.PaidAt,
	}
}

// 获取订单
func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*model.OrderResponse, error) {
	// 先从缓存获取
	key := fmt.Sprintf("order:%s", orderID)
	data, err := s.redisClient.Get(ctx, key).Result()
	if err == nil {
		var order model.Order
		if json.Unmarshal([]byte(data), &order) == nil {
			return s.orderToResponse(&order), nil
		}
	}

	// 从数据库获取
	var order model.Order
	err = s.db.GetDB().Where("order_id = ?", orderID).First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// 更新缓存
	s.cacheOrder(ctx, &order)

	return s.orderToResponse(&order), nil
}

// 更新订单状态
func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID, newStatus string) error {
	var order model.Order
	err := s.db.GetDB().Where("order_id = ?", orderID).First(&order).Error
	if err != nil {
		return fmt.Errorf("failed to find order: %w", err)
	}

	oldStatus := order.Status
	now := time.Now()

	updates := map[string]interface{}{
		"status":     newStatus,
		"updated_at": now,
	}

	// 根据状态设置特定字段
	switch newStatus {
	case model.OrderStatusPaid:
		updates["paid_at"] = &now
	case model.OrderStatusCancelled:
		updates["cancelled_at"] = &now
	}

	err = s.db.GetDB().Model(&order).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// 清除缓存
	key := fmt.Sprintf("order:%s", orderID)
	s.redisClient.Del(ctx, key)

	s.logger.WithFields(logrus.Fields{
		"order_id":   orderID,
		"old_status": oldStatus,
		"new_status": newStatus,
	}).Info("Order status updated")

	return nil
}

// 查询订单列表
func (s *OrderService) ListOrders(ctx context.Context, query *model.OrderQuery) (*model.PageResult, error) {
	db := s.db.GetDB()

	// 构建查询条件
	if query.UserID > 0 {
		db = db.Where("user_id = ?", query.UserID)
	}
	if query.ProductID > 0 {
		db = db.Where("product_id = ?", query.ProductID)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.OrderType != "" {
		db = db.Where("order_type = ?", query.OrderType)
	}
	if query.StartDate != "" {
		db = db.Where("created_at >= ?", query.StartDate)
	}
	if query.EndDate != "" {
		db = db.Where("created_at <= ?", query.EndDate)
	}

	// 计算总数
	var total int64
	if err := db.Model(&model.Order{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count orders: %w", err)
	}

	// 分页查询
	var orders []model.Order
	offset := (query.Page - 1) * query.PageSize
	err := db.Offset(offset).Limit(query.PageSize).
		Order("created_at DESC").Find(&orders).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}

	// 转换为响应对象
	var responses []*model.OrderResponse
	for _, order := range orders {
		responses = append(responses, s.orderToResponse(&order))
	}

	// 计算总页数
	totalPages := int(total) / query.PageSize
	if int(total)%query.PageSize > 0 {
		totalPages++
	}

	return &model.PageResult{
		Data:       responses,
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: totalPages,
	}, nil
}

// 获取服务统计信息
func (s *OrderService) GetServiceStats() ServiceStats {
	return s.stats
}

// GetOrderByID 根据订单ID获取订单详情
func (s *OrderService) GetOrderByID(orderID string) (*model.Order, error) {
	var order model.Order
	err := s.db.GetDB().Preload("Items").Where("order_id = ?", orderID).First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		s.logger.WithError(err).Error("获取订单详情失败")
		return nil, err
	}
	return &order, nil
}

// GetUserOrders 获取用户订单列表
func (s *OrderService) GetUserOrders(userID uint64, page, pageSize int, status string) ([]*model.Order, int64, error) {
	var orders []*model.Order
	var total int64

	query := s.db.GetDB().Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	err := query.Model(&model.Order{}).Count(&total).Error
	if err != nil {
		s.logger.WithError(err).Error("获取订单总数失败")
		return nil, 0, err
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	err = query.Preload("Items").
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&orders).Error
	if err != nil {
		s.logger.WithError(err).Error("获取用户订单列表失败")
		return nil, 0, err
	}

	return orders, total, nil
}

// CancelOrder 取消订单
func (s *OrderService) CancelOrder(orderID, reason string) error {
	tx := s.db.BeginTx()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取订单
	var order model.Order
	err := tx.Where("order_id = ?", orderID).First(&order).Error
	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("订单不存在")
		}
		s.logger.WithError(err).Error("获取订单失败")
		return err
	}

	// 检查订单状态
	if order.Status == "cancelled" {
		tx.Rollback()
		return errors.New("订单已取消")
	}
	if order.Status == "completed" {
		tx.Rollback()
		return errors.New("已完成的订单无法取消")
	}

	// 更新订单状态
	err = tx.Model(&order).Updates(map[string]interface{}{
		"status":     "cancelled",
		"remark":     reason,
		"updated_at": time.Now(),
	}).Error
	if err != nil {
		tx.Rollback()
		s.logger.WithError(err).Error("更新订单状态失败")
		return err
	}

	// 更新缓存
	s.cacheOrder(context.Background(), &order) // Assuming context.Background() is acceptable for this call

	tx.Commit()
	s.logger.WithField("order_id", orderID).Info("订单取消成功")
	return nil
}

// GetOrderStats 获取订单统计
func (s *OrderService) GetOrderStats(date string) (*model.OrderStats, error) {
	var stats model.OrderStats
	err := s.db.GetDB().Where("date = ?", date).First(&stats).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果没有统计数据，实时计算
			return s.calculateOrderStats(date)
		}
		s.logger.WithError(err).Error("获取订单统计失败")
		return nil, err
	}
	return &stats, nil
}

// calculateOrderStats 实时计算订单统计
func (s *OrderService) calculateOrderStats(dateStr string) (*model.OrderStats, error) {
	startTime := dateStr + " 00:00:00"
	endTime := dateStr + " 23:59:59"

	var stats model.OrderStats
	// 这里需要解析日期字符串为time.Time类型，暂时简化处理
	// stats.Date = date

	// 总订单数
	err := s.db.GetDB().Model(&model.Order{}).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Count(&stats.TotalOrders).Error
	if err != nil {
		return nil, err
	}

	// 成功订单数 - 需要检查OrderStats结构体的实际字段名
	var successCount int64
	err = s.db.GetDB().Model(&model.Order{}).
		Where("created_at BETWEEN ? AND ? AND status = ?", startTime, endTime, "completed").
		Count(&successCount).Error
	if err != nil {
		return nil, err
	}

	// 失败订单数 - 需要检查OrderStats结构体的实际字段名
	var failedCount int64
	err = s.db.GetDB().Model(&model.Order{}).
		Where("created_at BETWEEN ? AND ? AND status = ?", startTime, endTime, "failed").
		Count(&failedCount).Error
	if err != nil {
		return nil, err
	}

	// 总金额
	err = s.db.GetDB().Model(&model.Order{}).
		Where("created_at BETWEEN ? AND ? AND status = ?", startTime, endTime, "completed").
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&stats.TotalAmount).Error
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// RetryFailedOrder 重试失败订单
func (s *OrderService) RetryFailedOrder(failureID uint64) error {
	var failure model.OrderFailure
	err := s.db.GetDB().Where("id = ?", failureID).First(&failure).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("失败记录不存在")
		}
		s.logger.WithError(err).Error("获取失败记录失败")
		return err
	}

	if failure.Status == "success" {
		return errors.New("该记录已处理成功")
	}

	// 解析消息数据
	var msg mq.SeckillOrderMessage
	err = json.Unmarshal([]byte(failure.MessageData), &msg)
	if err != nil {
		s.logger.WithError(err).Error("解析消息数据失败")
		return err
	}

	// 重试处理订单
	err = s.HandleSeckillOrder(context.Background(), &msg)
	if err != nil {
		// 更新重试次数
		s.db.GetDB().Model(&failure).Updates(map[string]interface{}{
			"retry_count": failure.RetryCount + 1,
			"last_error":  err.Error(),
			"updated_at":  time.Now(),
		})
		return err
	}

	// 标记为成功
	s.db.GetDB().Model(&failure).Updates(map[string]interface{}{
		"status":     "success",
		"updated_at": time.Now(),
	})

	s.logger.WithField("failure_id", failureID).Info("失败订单重试成功")
	return nil
}

// GetFailedOrders 获取失败订单列表
func (s *OrderService) GetFailedOrders(page, pageSize int, status string) ([]*model.OrderFailure, int64, error) {
	var failures []*model.OrderFailure
	var total int64

	query := s.db.GetDB().Model(&model.OrderFailure{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	err := query.Count(&total).Error
	if err != nil {
		s.logger.WithError(err).Error("获取失败订单总数失败")
		return nil, 0, err
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	err = query.Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&failures).Error
	if err != nil {
		s.logger.WithError(err).Error("获取失败订单列表失败")
		return nil, 0, err
	}

	return failures, total, nil
}

// HealthCheck 健康检查
func (s *OrderService) HealthCheck(ctx context.Context) bool {
	// 检查数据库连接
	sqlDB, err := s.db.GetDB().DB()
	if err != nil {
		s.logger.WithError(err).Error("获取数据库连接失败")
		return false
	}

	err = sqlDB.PingContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("数据库连接检查失败")
		return false
	}

	// 检查Redis连接
	_, err = s.redisClient.Ping(ctx).Result()
	if err != nil {
		s.logger.WithError(err).Error("Redis连接检查失败")
		return false
	}

	return true
}

// ProcessFailedOrder 实现CompensationHandler接口，处理失败的订单
func (s *OrderService) ProcessFailedOrder(ctx context.Context, failure *model.OrderFailure) error {
	s.logger.WithFields(logrus.Fields{
		"order_id":     failure.OrderID,
		"failure_type": failure.FailureType,
		"retry_count":  failure.RetryCount,
	}).Info("开始处理失败订单")

	// 根据失败类型进行不同的处理
	switch failure.FailureType {
	case model.FailureTypeInventoryLock:
		return s.processInventoryLockFailure(ctx, failure)
	case model.FailureTypePayment:
		return s.processPaymentFailure(ctx, failure)
	case model.FailureTypeOrderCreation:
		return s.processOrderCreationFailure(ctx, failure)
	default:
		return fmt.Errorf("未知的失败类型: %s", failure.FailureType)
	}
}

// processInventoryLockFailure 处理库存锁定失败
func (s *OrderService) processInventoryLockFailure(ctx context.Context, failure *model.OrderFailure) error {
	// 重试库存锁定逻辑
	s.logger.WithField("order_id", failure.OrderID).Info("重试库存锁定")
	// 这里可以实现具体的重试逻辑
	return nil
}

// processPaymentFailure 处理支付失败
func (s *OrderService) processPaymentFailure(ctx context.Context, failure *model.OrderFailure) error {
	// 处理支付失败，可能需要退款或重试
	s.logger.WithField("order_id", failure.OrderID).Info("处理支付失败")
	// 这里可以实现具体的支付失败处理逻辑
	return nil
}

// processOrderCreationFailure 处理订单创建失败
func (s *OrderService) processOrderCreationFailure(ctx context.Context, failure *model.OrderFailure) error {
	// 处理订单创建失败
	s.logger.WithField("order_id", failure.OrderID).Info("处理订单创建失败")
	// 这里可以实现具体的订单创建失败处理逻辑
	return nil
}
