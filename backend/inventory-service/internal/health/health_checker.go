package health

import (
	"context"
	"fmt"
	"time"

	"inventory-service/internal/config"
	"inventory-service/internal/model"
	"inventory-service/internal/service"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// 健康检查管理器
type HealthChecker struct {
	config           *config.Config
	inventoryService *service.InventoryService
	cron             *cron.Cron
	logger           *logrus.Logger

	// 统计信息
	stats HealthStats
}

// 健康检查统计信息
type HealthStats struct {
	TotalChecks      int64
	FoundDifferences int64
	AutoFixed        int64
	AlertsSent       int64
	LastCheckTime    time.Time
}

// 创建健康检查管理器
func NewHealthChecker(cfg *config.Config, inventoryService *service.InventoryService, logger *logrus.Logger) *HealthChecker {
	return &HealthChecker{
		config:           cfg,
		inventoryService: inventoryService,
		cron:             cron.New(),
		logger:           logger,
	}
}

// 启动健康检查
func (hc *HealthChecker) Start(ctx context.Context) error {
	if !hc.config.Inventory.HealthCheck.Enable {
		hc.logger.Info("库存健康检查已禁用")
		return nil
	}

	// 添加定时任务
	cronSpec := fmt.Sprintf("@every %s", hc.config.Inventory.HealthCheck.Interval)
	_, err := hc.cron.AddFunc(cronSpec, func() {
		hc.performHealthCheck(ctx)
	})
	if err != nil {
		return fmt.Errorf("添加定时任务失败: %w", err)
	}

	// 启动定时器
	hc.cron.Start()

	hc.logger.Infof("库存健康检查已启动，检查间隔: %s", hc.config.Inventory.HealthCheck.Interval)
	return nil
}

// 停止健康检查
func (hc *HealthChecker) Stop() {
	if hc.cron != nil {
		hc.cron.Stop()
	}
	hc.logger.Info("库存健康检查已停止")
}

// 执行健康检查
func (hc *HealthChecker) performHealthCheck(ctx context.Context) {
	hc.stats.TotalChecks++
	hc.stats.LastCheckTime = time.Now()

	hc.logger.Info("开始执行库存健康检查")

	// 执行库存差异检查
	healthResponses, err := hc.inventoryService.CheckInventoryHealth(ctx)
	if err != nil {
		hc.logger.WithError(err).Error("库存健康检查失败")
		return
	}

	var differencesFound int64
	var autoFixed int64

	for _, response := range healthResponses {
		if response.Status == "diff_found" {
			differencesFound++

			hc.logger.WithFields(logrus.Fields{
				"product_id":  response.ProductID,
				"db_stock":    response.DBStock,
				"redis_stock": response.RedisStock,
				"diff":        response.Diff,
			}).Warn("发现库存差异")

			// 检查是否需要自动修复
			if hc.config.Inventory.Compensation.AutoFix &&
				abs(response.Diff) <= hc.config.Inventory.Compensation.MaxFixAmount {

				if hc.autoFixDifference(ctx, response) {
					autoFixed++
				}
			}
		}
	}

	hc.stats.FoundDifferences += differencesFound
	hc.stats.AutoFixed += autoFixed

	hc.logger.WithFields(logrus.Fields{
		"total_products":    len(healthResponses),
		"differences_found": differencesFound,
		"auto_fixed":        autoFixed,
	}).Info("库存健康检查完成")
}

// 自动修复库存差异
func (hc *HealthChecker) autoFixDifference(ctx context.Context, response *model.InventoryHealthResponse) bool {
	hc.logger.WithFields(logrus.Fields{
		"product_id": response.ProductID,
		"diff":       response.Diff,
	}).Info("尝试自动修复库存差异")

	// 简单策略：如果Redis库存更高，认为是正确的（因为Redis是实时扣减的）
	fixType := "use_redis"
	if response.Diff < 0 {
		// 如果Redis库存更低，可能是缓存丢失，使用数据库库存
		fixType = "use_db"
	}

	// 这里需要获取差异记录ID，简化处理
	// 在实际实现中，需要先查询最新的差异记录
	diffID := hc.getLatestDiffID(ctx, response.ProductID)
	if diffID == 0 {
		hc.logger.WithField("product_id", response.ProductID).Error("无法获取差异记录ID")
		return false
	}

	err := hc.inventoryService.FixInventoryDiff(ctx, diffID, fixType)
	if err != nil {
		hc.logger.WithError(err).WithField("product_id", response.ProductID).Error("自动修复库存差异失败")
		return false
	}

	hc.logger.WithFields(logrus.Fields{
		"product_id": response.ProductID,
		"fix_type":   fixType,
	}).Info("自动修复库存差异成功")

	return true
}

// 获取最新的差异记录ID（简化实现）
func (hc *HealthChecker) getLatestDiffID(ctx context.Context, productID int64) uint {
	_ = productID
	_ = ctx
	// 这里应该查询数据库获取最新的差异记录ID
	// 简化处理，返回一个模拟的ID
	return 1
}

// 手动触发健康检查
func (hc *HealthChecker) TriggerCheck(ctx context.Context) error {
	hc.logger.Info("手动触发库存健康检查")
	hc.performHealthCheck(ctx)
	return nil
}

// 获取健康检查统计信息
func (hc *HealthChecker) GetStats() HealthStats {
	return hc.stats
}

// 重置统计信息
func (hc *HealthChecker) ResetStats() {
	hc.stats = HealthStats{}
	hc.logger.Info("健康检查统计信息已重置")
}

// 绝对值函数
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
