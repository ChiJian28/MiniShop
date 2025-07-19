package compensation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"order-service/internal/config"
	"order-service/internal/database"
	"order-service/internal/model"
	"order-service/internal/mq"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// 补偿管理器
type CompensationManager struct {
	config  *config.Config
	db      *database.Database
	cron    *cron.Cron
	logger  *logrus.Logger
	handler CompensationHandler

	// 统计信息
	stats CompensationStats
}

// 补偿处理器接口
type CompensationHandler interface {
	ProcessFailedOrder(ctx context.Context, failure *model.OrderFailure) error
}

// 补偿统计信息
type CompensationStats struct {
	TotalTasks     int64
	ProcessedTasks int64
	SuccessTasks   int64
	FailedTasks    int64
	RetryTasks     int64
	ExpiredTasks   int64
}

// 创建补偿管理器
func NewCompensationManager(cfg *config.Config, db *database.Database, handler CompensationHandler, logger *logrus.Logger) *CompensationManager {
	return &CompensationManager{
		config:  cfg,
		db:      db,
		cron:    cron.New(),
		logger:  logger,
		handler: handler,
	}
}

// 启动补偿任务
func (cm *CompensationManager) Start(ctx context.Context) error {
	if !cm.config.Order.Compensation.Enable {
		cm.logger.Info("Compensation is disabled")
		return nil
	}

	// 添加定时任务
	cronSpec := fmt.Sprintf("@every %s", cm.config.Order.Compensation.CheckInterval)
	_, err := cm.cron.AddFunc(cronSpec, func() {
		cm.processFailedOrders(ctx)
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	// 启动定时器
	cm.cron.Start()

	cm.logger.Infof("Compensation manager started with interval: %s", cm.config.Order.Compensation.CheckInterval)
	return nil
}

// 停止补偿任务
func (cm *CompensationManager) Stop() {
	if cm.cron != nil {
		cm.cron.Stop()
	}
	cm.logger.Info("Compensation manager stopped")
}

// 处理失败的订单
func (cm *CompensationManager) processFailedOrders(ctx context.Context) {
	cm.logger.Debug("Starting to process failed orders")

	// 查询需要重试的失败订单
	var failures []model.OrderFailure
	err := cm.db.GetDB().Where(
		"status = ? AND next_retry_at <= ? AND retry_count < max_retries AND created_at > ?",
		"pending",
		time.Now(),
		time.Now().Add(-time.Duration(cm.config.Order.Compensation.MaxRetryHours)*time.Hour),
	).Limit(cm.config.Order.Compensation.BatchSize).Find(&failures).Error

	if err != nil {
		cm.logger.Errorf("Failed to query failed orders: %v", err)
		return
	}

	if len(failures) == 0 {
		cm.logger.Debug("No failed orders to process")
		return
	}

	cm.logger.Infof("Found %d failed orders to process", len(failures))

	for _, failure := range failures {
		cm.processFailedOrder(ctx, &failure)
	}

	// 清理过期的失败记录
	cm.cleanupExpiredFailures(ctx)
}

// 处理单个失败订单
func (cm *CompensationManager) processFailedOrder(ctx context.Context, failure *model.OrderFailure) {
	cm.stats.TotalTasks++
	cm.stats.ProcessedTasks++

	cm.logger.WithFields(logrus.Fields{
		"failure_id":  failure.ID,
		"order_id":    failure.OrderID,
		"user_id":     failure.UserID,
		"product_id":  failure.ProductID,
		"retry_count": failure.RetryCount,
	}).Info("Processing failed order")

	// 更新状态为处理中
	err := cm.updateFailureStatus(failure.ID, "processing", "")
	if err != nil {
		cm.logger.Errorf("Failed to update failure status: %v", err)
		return
	}

	// 调用处理器
	err = cm.handler.ProcessFailedOrder(ctx, failure)

	if err != nil {
		cm.handleRetryFailure(failure, err)
	} else {
		cm.handleRetrySuccess(failure)
	}
}

// 处理重试成功
func (cm *CompensationManager) handleRetrySuccess(failure *model.OrderFailure) {
	cm.stats.SuccessTasks++

	err := cm.updateFailureStatus(failure.ID, "success", "")
	if err != nil {
		cm.logger.Errorf("Failed to update failure status to success: %v", err)
	}

	cm.logger.WithFields(logrus.Fields{
		"failure_id":  failure.ID,
		"order_id":    failure.OrderID,
		"retry_count": failure.RetryCount,
	}).Info("Failed order processed successfully")
}

// 处理重试失败
func (cm *CompensationManager) handleRetryFailure(failure *model.OrderFailure, err error) {
	cm.stats.FailedTasks++

	failure.RetryCount++
	failure.ErrorMsg = err.Error()

	if failure.RetryCount >= failure.MaxRetries {
		// 达到最大重试次数，标记为失败
		cm.updateFailureStatus(failure.ID, "failed", err.Error())
		cm.logger.WithFields(logrus.Fields{
			"failure_id":  failure.ID,
			"order_id":    failure.OrderID,
			"retry_count": failure.RetryCount,
			"max_retries": failure.MaxRetries,
		}).Error("Failed order reached max retries")
	} else {
		// 计算下次重试时间
		nextRetry := cm.calculateNextRetryTime(failure.RetryCount)

		err := cm.db.GetDB().Model(failure).Updates(map[string]interface{}{
			"retry_count":   failure.RetryCount,
			"error_msg":     failure.ErrorMsg,
			"next_retry_at": nextRetry,
			"status":        "pending",
			"updated_at":    time.Now(),
		}).Error

		if err != nil {
			cm.logger.Errorf("Failed to update failure for retry: %v", err)
		} else {
			cm.stats.RetryTasks++
			cm.logger.WithFields(logrus.Fields{
				"failure_id":  failure.ID,
				"order_id":    failure.OrderID,
				"retry_count": failure.RetryCount,
				"next_retry":  nextRetry,
			}).Warn("Failed order scheduled for retry")
		}
	}
}

// 更新失败记录状态
func (cm *CompensationManager) updateFailureStatus(failureID uint, status, errorMsg string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}

	return cm.db.GetDB().Model(&model.OrderFailure{}).Where("id = ?", failureID).Updates(updates).Error
}

// 计算下次重试时间
func (cm *CompensationManager) calculateNextRetryTime(retryCount int) time.Time {
	interval := cm.config.Order.Retry.InitialInterval
	for i := 0; i < retryCount; i++ {
		interval = time.Duration(float64(interval) * cm.config.Order.Retry.Multiplier)
		if interval > cm.config.Order.Retry.MaxInterval {
			interval = cm.config.Order.Retry.MaxInterval
			break
		}
	}

	return time.Now().Add(interval)
}

// 清理过期的失败记录
func (cm *CompensationManager) cleanupExpiredFailures(ctx context.Context) {
	expiredTime := time.Now().Add(-time.Duration(cm.config.Order.Compensation.MaxRetryHours) * time.Hour)

	result := cm.db.GetDB().Where("created_at < ? AND status IN (?)", expiredTime, []string{"pending", "processing"}).
		Updates(&model.OrderFailure{Status: "expired"})

	if result.Error != nil {
		cm.logger.Errorf("Failed to cleanup expired failures: %v", result.Error)
	} else if result.RowsAffected > 0 {
		cm.stats.ExpiredTasks += result.RowsAffected
		cm.logger.Infof("Cleaned up %d expired failure records", result.RowsAffected)
	}
}

// 手动重试失败订单
func (cm *CompensationManager) ManualRetry(ctx context.Context, failureID uint) error {
	var failure model.OrderFailure
	err := cm.db.GetDB().Where("id = ?", failureID).First(&failure).Error
	if err != nil {
		return fmt.Errorf("failed to find failure record: %w", err)
	}

	if failure.Status == "success" {
		return fmt.Errorf("failure already processed successfully")
	}

	cm.logger.WithFields(logrus.Fields{
		"failure_id": failureID,
		"order_id":   failure.OrderID,
	}).Info("Manual retry triggered")

	cm.processFailedOrder(ctx, &failure)
	return nil
}

// 获取失败订单列表
func (cm *CompensationManager) GetFailedOrders(ctx context.Context, status string, limit int) ([]model.OrderFailure, error) {
	var failures []model.OrderFailure
	query := cm.db.GetDB()

	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Order("created_at DESC").Limit(limit).Find(&failures).Error
	return failures, err
}

// 获取补偿统计信息
func (cm *CompensationManager) GetStats() CompensationStats {
	return cm.stats
}

// 默认补偿处理器
type DefaultCompensationHandler struct {
	orderService OrderService
	logger       *logrus.Logger
}

// OrderService 接口
type OrderService interface {
	CreateOrder(ctx context.Context, request *model.CreateOrderRequest) (*model.OrderResponse, error)
}

// 创建默认补偿处理器
func NewDefaultCompensationHandler(orderService OrderService, logger *logrus.Logger) *DefaultCompensationHandler {
	return &DefaultCompensationHandler{
		orderService: orderService,
		logger:       logger,
	}
}

// 处理失败订单
func (h *DefaultCompensationHandler) ProcessFailedOrder(ctx context.Context, failure *model.OrderFailure) error {
	// 解析原始消息
	var message mq.SeckillOrderMessage
	err := json.Unmarshal([]byte(failure.MessageData), &message)
	if err != nil {
		return fmt.Errorf("failed to unmarshal message data: %w", err)
	}

	// 创建订单请求
	request := &model.CreateOrderRequest{
		OrderID:     message.OrderID,
		UserID:      message.UserID,
		ProductID:   message.ProductID,
		ProductName: fmt.Sprintf("Product-%d", message.ProductID),
		Quantity:    message.Quantity,
		Price:       message.Price,
		OrderType:   model.OrderTypeSeckill,
		TraceID:     message.TraceID,
	}

	// 重试创建订单
	_, err = h.orderService.CreateOrder(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to retry create order: %w", err)
	}

	h.logger.WithFields(logrus.Fields{
		"failure_id": failure.ID,
		"order_id":   failure.OrderID,
		"user_id":    failure.UserID,
		"product_id": failure.ProductID,
	}).Info("Failed order compensated successfully")

	return nil
}

// 补偿任务信息
type CompensationTask struct {
	ID          uint       `json:"id"`
	OrderID     string     `json:"order_id"`
	UserID      int64      `json:"user_id"`
	ProductID   int64      `json:"product_id"`
	Status      string     `json:"status"`
	RetryCount  int        `json:"retry_count"`
	MaxRetries  int        `json:"max_retries"`
	ErrorMsg    string     `json:"error_msg"`
	NextRetryAt *time.Time `json:"next_retry_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// 转换为任务信息
func (cm *CompensationManager) toTaskInfo(failure *model.OrderFailure) *CompensationTask {
	return &CompensationTask{
		ID:          failure.ID,
		OrderID:     failure.OrderID,
		UserID:      failure.UserID,
		ProductID:   failure.ProductID,
		Status:      failure.Status,
		RetryCount:  failure.RetryCount,
		MaxRetries:  failure.MaxRetries,
		ErrorMsg:    failure.ErrorMsg,
		NextRetryAt: failure.NextRetryAt,
		CreatedAt:   failure.CreatedAt,
		UpdatedAt:   failure.UpdatedAt,
	}
}

// 获取补偿任务列表
func (cm *CompensationManager) GetCompensationTasks(ctx context.Context, status string, page, pageSize int) (*model.PageResult, error) {
	query := cm.db.GetDB().Model(&model.OrderFailure{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count compensation tasks: %w", err)
	}

	// 分页查询
	var failures []model.OrderFailure
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).
		Order("created_at DESC").Find(&failures).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query compensation tasks: %w", err)
	}

	// 转换为任务信息
	var tasks []*CompensationTask
	for _, failure := range failures {
		tasks = append(tasks, cm.toTaskInfo(&failure))
	}

	// 计算总页数
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &model.PageResult{
		Data:       tasks,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}
