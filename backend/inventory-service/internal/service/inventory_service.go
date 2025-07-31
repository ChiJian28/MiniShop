package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"inventory-service/internal/config"
	"inventory-service/internal/database"
	"inventory-service/internal/model"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// 库存服务
type InventoryService struct {
	config      *config.Config
	db          *database.Database
	redisClient *redis.Client
	logger      *logrus.Logger

	// 统计信息
	stats ServiceStats
}

// 服务统计信息
type ServiceStats struct {
	TotalSyncRequests   int64
	SuccessSyncRequests int64
	FailedSyncRequests  int64
	TotalDiffChecks     int64
	FoundDifferences    int64
	FixedDifferences    int64
}

// 创建库存服务
func NewInventoryService(cfg *config.Config, db *database.Database, redisClient *redis.Client, logger *logrus.Logger) *InventoryService {
	return &InventoryService{
		config:      cfg,
		db:          db,
		redisClient: redisClient,
		logger:      logger,
	}
}

// SyncStock 同步库存 - 核心接口
func (s *InventoryService) SyncStock(ctx context.Context, req *model.SyncStockRequest) (*model.SyncStockResponse, error) {
	s.stats.TotalSyncRequests++

	s.logger.WithFields(logrus.Fields{
		"product_id": req.ProductID,
		"delta":      req.Delta,
		"order_id":   req.OrderID,
		"trace_id":   req.TraceID,
	}).Info("开始同步库存")

	// 参数验证
	if req.ProductID <= 0 {
		return nil, fmt.Errorf("商品ID必须大于0")
	}

	// 开始事务
	tx := s.db.BeginTx()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// 获取当前库存（加行锁）
	var inventory model.Inventory
	err := tx.Where("product_id = ?", req.ProductID).First(&inventory).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 商品不存在，创建初始库存
			inventory = model.Inventory{
				ProductID:   req.ProductID,
				ProductName: fmt.Sprintf("Product-%d", req.ProductID),
				Stock:       s.config.Inventory.DefaultStock,
				Reserved:    0,
				Available:   s.config.Inventory.DefaultStock,
				Status:      model.InventoryStatusActive,
				MinStock:    s.config.Inventory.LowStockThreshold,
				MaxStock:    s.config.Inventory.DefaultStock * 10,
				Version:     0,
			}

			if err := tx.Create(&inventory).Error; err != nil {
				tx.Rollback()
				s.stats.FailedSyncRequests++
				return nil, fmt.Errorf("创建库存记录失败: %w", err)
			}
		} else {
			tx.Rollback()
			s.stats.FailedSyncRequests++
			return nil, fmt.Errorf("获取库存信息失败: %w", err)
		}
	}

	beforeStock := inventory.Stock
	afterStock := beforeStock + req.Delta

	// 检查库存是否足够（如果是扣减操作）
	if req.Delta < 0 && afterStock < 0 {
		tx.Rollback()
		s.stats.FailedSyncRequests++
		return &model.SyncStockResponse{
			ProductID:   req.ProductID,
			BeforeStock: beforeStock,
			AfterStock:  beforeStock,
			Delta:       0,
			Success:     false,
			Message:     "库存不足",
		}, nil
	}

	// 更新库存（使用乐观锁）
	result := tx.Model(&inventory).
		Where("version = ?", inventory.Version).
		Updates(map[string]interface{}{
			"stock":     afterStock,
			"available": afterStock - inventory.Reserved,
			"version":   inventory.Version + 1,
		})

	if result.Error != nil {
		tx.Rollback()
		s.stats.FailedSyncRequests++
		return nil, fmt.Errorf("更新库存失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		s.stats.FailedSyncRequests++
		return nil, fmt.Errorf("库存更新冲突，请重试")
	}

	// 记录操作日志
	log := &model.InventoryLog{
		ProductID:   req.ProductID,
		OpType:      model.InventoryOpTypeSync,
		Delta:       req.Delta,
		BeforeStock: beforeStock,
		AfterStock:  afterStock,
		OrderID:     req.OrderID,
		Reason:      req.Reason,
		Operator:    "system",
		TraceID:     req.TraceID,
	}

	if err := tx.Create(log).Error; err != nil {
		tx.Rollback()
		s.stats.FailedSyncRequests++
		return nil, fmt.Errorf("记录操作日志失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		s.stats.FailedSyncRequests++
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	// 异步更新缓存
	go s.updateInventoryCache(ctx, req.ProductID, afterStock)

	// 检查是否需要发送库存预警
	go s.checkStockAlert(ctx, req.ProductID, afterStock)

	s.stats.SuccessSyncRequests++

	s.logger.WithFields(logrus.Fields{
		"product_id":   req.ProductID,
		"before_stock": beforeStock,
		"after_stock":  afterStock,
		"delta":        req.Delta,
	}).Info("库存同步成功")

	return &model.SyncStockResponse{
		ProductID:   req.ProductID,
		BeforeStock: beforeStock,
		AfterStock:  afterStock,
		Delta:       req.Delta,
		Success:     true,
	}, nil
}

// GetInventory 获取库存信息
func (s *InventoryService) GetInventory(ctx context.Context, productID int64) (*model.InventoryResponse, error) {
	// 先从缓存获取
	cacheKey := fmt.Sprintf("inventory:%d", productID)
	cached, err := s.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var inventory model.Inventory
		if json.Unmarshal([]byte(cached), &inventory) == nil {
			return s.inventoryToResponse(&inventory), nil
		}
	}

	// 从数据库获取
	var inventory model.Inventory
	err = s.db.GetDB().Where("product_id = ?", productID).First(&inventory).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("商品库存不存在")
		}
		return nil, fmt.Errorf("获取库存信息失败: %w", err)
	}

	// 更新缓存
	s.updateInventoryCache(ctx, productID, inventory.Stock)

	return s.inventoryToResponse(&inventory), nil
}

// BatchGetInventory 批量获取库存信息
func (s *InventoryService) BatchGetInventory(ctx context.Context, productIDs []int64) ([]*model.InventoryResponse, error) {
	var inventories []model.Inventory
	err := s.db.GetDB().Where("product_id IN ?", productIDs).Find(&inventories).Error
	if err != nil {
		return nil, fmt.Errorf("批量获取库存信息失败: %w", err)
	}

	var responses []*model.InventoryResponse
	for _, inventory := range inventories {
		responses = append(responses, s.inventoryToResponse(&inventory))
	}

	return responses, nil
}

// CheckInventoryHealth 健康检查 - 对比Redis与DB的库存差异
func (s *InventoryService) CheckInventoryHealth(ctx context.Context) ([]*model.InventoryHealthResponse, error) {
	s.stats.TotalDiffChecks++

	s.logger.Info("开始库存健康检查")

	// 获取所有活跃的库存记录
	var inventories []model.Inventory
	err := s.db.GetDB().Where("status = ?", model.InventoryStatusActive).Find(&inventories).Error
	if err != nil {
		return nil, fmt.Errorf("获取库存列表失败: %w", err)
	}

	var healthResponses []*model.InventoryHealthResponse
	var foundDiffs int64

	for _, inventory := range inventories {
		// 获取Redis中的库存
		redisKey := fmt.Sprintf("seckill:stock:%d", inventory.ProductID)
		redisStock, err := s.redisClient.Get(ctx, redisKey).Int64()
		if err != nil {
			if err == redis.Nil {
				redisStock = 0
			} else {
				s.logger.WithError(err).Errorf("获取Redis库存失败: product_id=%d", inventory.ProductID)
				continue
			}
		}

		diff := redisStock - inventory.Stock
		status := "normal"

		// 检查差异是否超过阈值
		if abs(diff) > s.config.Inventory.HealthCheck.Tolerance {
			status = "diff_found"
			foundDiffs++

			// 记录差异
			s.recordInventoryDiff(ctx, inventory.ProductID, inventory.Stock, redisStock, diff)

			// 检查是否需要报警
			if abs(diff) > s.config.Inventory.HealthCheck.AlertThreshold {
				s.createAlert(ctx, inventory.ProductID, "diff_alert",
					fmt.Sprintf("库存差异过大: DB=%d, Redis=%d, 差异=%d", inventory.Stock, redisStock, diff), "error")
			}
		}

		healthResponses = append(healthResponses, &model.InventoryHealthResponse{
			ProductID:  inventory.ProductID,
			DBStock:    inventory.Stock,
			RedisStock: redisStock,
			Diff:       diff,
			Status:     status,
		})
	}

	s.stats.FoundDifferences += foundDiffs

	s.logger.WithFields(logrus.Fields{
		"total_products": len(inventories),
		"found_diffs":    foundDiffs,
	}).Info("库存健康检查完成")

	return healthResponses, nil
}

// FixInventoryDiff 修复库存差异
func (s *InventoryService) FixInventoryDiff(ctx context.Context, diffID uint, fixType string) error {
	var diff model.InventoryDiff
	err := s.db.GetDB().Where("id = ? AND status = ?", diffID, "pending").First(&diff).Error
	if err != nil {
		return fmt.Errorf("获取差异记录失败: %w", err)
	}

	switch fixType {
	case "use_db":
		// 使用数据库库存作为准确值，更新Redis
		redisKey := fmt.Sprintf("seckill:stock:%d", diff.ProductID)
		err = s.redisClient.Set(ctx, redisKey, diff.DBStock, 0).Err()
		if err != nil {
			return fmt.Errorf("更新Redis库存失败: %w", err)
		}
	case "use_redis":
		// 使用Redis库存作为准确值，更新数据库
		_, err := s.SyncStock(ctx, &model.SyncStockRequest{
			ProductID: diff.ProductID,
			Delta:     diff.RedisStock - diff.DBStock,
			Reason:    "健康检查修复",
			TraceID:   fmt.Sprintf("health-fix-%d", diffID),
		})
		if err != nil {
			return fmt.Errorf("更新数据库库存失败: %w", err)
		}
	default:
		return fmt.Errorf("不支持的修复类型: %s", fixType)
	}

	// 更新差异记录状态
	now := time.Now()
	err = s.db.GetDB().Model(&diff).Updates(map[string]interface{}{
		"status":   "fixed",
		"fixed_at": &now,
		"fixed_by": "system",
		"remark":   fmt.Sprintf("使用%s修复", fixType),
	}).Error

	if err != nil {
		return fmt.Errorf("更新差异记录失败: %w", err)
	}

	s.stats.FixedDifferences++

	s.logger.WithFields(logrus.Fields{
		"diff_id":    diffID,
		"product_id": diff.ProductID,
		"fix_type":   fixType,
	}).Info("库存差异修复成功")

	return nil
}

// GetInventoryStats 获取库存统计
func (s *InventoryService) GetInventoryStats(ctx context.Context) (*model.InventoryStats, error) {
	var stats model.InventoryStats

	// 总商品数
	err := s.db.GetDB().Model(&model.Inventory{}).Count(&stats.TotalProducts).Error
	if err != nil {
		return nil, fmt.Errorf("获取总商品数失败: %w", err)
	}

	// 低库存商品数
	err = s.db.GetDB().Model(&model.Inventory{}).
		Where("stock <= min_stock AND status = ?", model.InventoryStatusActive).
		Count(&stats.LowStockProducts).Error
	if err != nil {
		return nil, fmt.Errorf("获取低库存商品数失败: %w", err)
	}

	// 缺货商品数
	err = s.db.GetDB().Model(&model.Inventory{}).
		Where("stock <= 0 AND status = ?", model.InventoryStatusActive).
		Count(&stats.OutOfStockProducts).Error
	if err != nil {
		return nil, fmt.Errorf("获取缺货商品数失败: %w", err)
	}

	// 总库存、预留库存、可用库存
	var result struct {
		TotalStock     int64
		TotalReserved  int64
		TotalAvailable int64
	}

	err = s.db.GetDB().Model(&model.Inventory{}).
		Where("status = ?", model.InventoryStatusActive).
		Select("SUM(stock) as total_stock, SUM(reserved) as total_reserved, SUM(available) as total_available").
		Scan(&result).Error

	if err != nil {
		return nil, fmt.Errorf("获取库存统计失败: %w", err)
	}

	stats.TotalStock = result.TotalStock
	stats.TotalReserved = result.TotalReserved
	stats.TotalAvailable = result.TotalAvailable

	return &stats, nil
}

// 辅助方法

// updateInventoryCache 更新库存缓存
func (s *InventoryService) updateInventoryCache(ctx context.Context, productID int64, stock int64) {
	_ = productID
	_ = stock
	_ = ctx
	cacheKey := fmt.Sprintf("inventory:%d", productID)

	// 获取完整的库存信息
	var inventory model.Inventory
	err := s.db.GetDB().Where("product_id = ?", productID).First(&inventory).Error
	if err != nil {
		s.logger.WithError(err).Errorf("获取库存信息失败: product_id=%d", productID)
		return
	}

	data, err := json.Marshal(inventory)
	if err != nil {
		s.logger.WithError(err).Errorf("序列化库存信息失败: product_id=%d", productID)
		return
	}

	err = s.redisClient.Set(ctx, cacheKey, data, time.Hour).Err()
	if err != nil {
		s.logger.WithError(err).Errorf("更新库存缓存失败: product_id=%d", productID)
	}
}

// checkStockAlert 检查库存预警
func (s *InventoryService) checkStockAlert(ctx context.Context, productID int64, stock int64) {
	var inventory model.Inventory
	err := s.db.GetDB().Where("product_id = ?", productID).First(&inventory).Error
	if err != nil {
		return
	}

	// 检查缺货
	if stock <= 0 {
		s.createAlert(ctx, productID, "out_of_stock",
			fmt.Sprintf("商品%d已缺货", productID), "error")
	} else if stock <= inventory.MinStock {
		// 检查低库存
		s.createAlert(ctx, productID, "low_stock",
			fmt.Sprintf("商品%d库存不足，当前库存: %d，最小库存: %d", productID, stock, inventory.MinStock), "warning")
	}
}

// createAlert 创建预警
func (s *InventoryService) createAlert(ctx context.Context, productID int64, alertType, message, level string) {
	_ = ctx
	alert := &model.InventoryAlert{
		ProductID: productID,
		AlertType: alertType,
		Message:   message,
		Level:     level,
		Status:    "pending",
	}

	err := s.db.GetDB().Create(alert).Error
	if err != nil {
		s.logger.WithError(err).Errorf("创建预警失败: product_id=%d", productID)
	}
}

// recordInventoryDiff 记录库存差异
func (s *InventoryService) recordInventoryDiff(ctx context.Context, productID, dbStock, redisStock, diff int64) {
	diffRecord := &model.InventoryDiff{
		ProductID:  productID,
		DBStock:    dbStock,
		RedisStock: redisStock,
		Diff:       diff,
		Status:     "pending",
	}

	err := s.db.GetDB().WithContext(ctx).Create(diffRecord).Error

	if err != nil {
		s.logger.WithError(err).Errorf("记录库存差异失败: product_id=%d", productID)
	}
}

// inventoryToResponse 转换为响应对象
func (s *InventoryService) inventoryToResponse(inventory *model.Inventory) *model.InventoryResponse {
	return &model.InventoryResponse{
		ProductID:   inventory.ProductID,
		ProductName: inventory.ProductName,
		Stock:       inventory.Stock,
		Reserved:    inventory.Reserved,
		Available:   inventory.Available,
		Status:      inventory.Status,
		MinStock:    inventory.MinStock,
		MaxStock:    inventory.MaxStock,
		Version:     inventory.Version,
		CreatedAt:   inventory.CreatedAt,
		UpdatedAt:   inventory.UpdatedAt,
	}
}

// abs 绝对值函数
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// GetServiceStats 获取服务统计信息
func (s *InventoryService) GetServiceStats() ServiceStats {
	return s.stats
}

// HealthCheck 健康检查
func (s *InventoryService) HealthCheck(ctx context.Context) error {
	// 检查数据库连接
	if err := s.db.HealthCheck(); err != nil {
		return fmt.Errorf("数据库健康检查失败: %w", err)
	}

	// 检查 Redis 连接
	if err := s.redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis健康检查失败: %w", err)
	}

	return nil
}
