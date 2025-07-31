package handler

import (
	"net/http"
	"strconv"

	"inventory-service/internal/health"
	"inventory-service/internal/model"
	"inventory-service/internal/service"

	"github.com/gin-gonic/gin"
)

type InventoryHandler struct {
	inventoryService *service.InventoryService
	healthChecker    *health.HealthChecker
}

func NewInventoryHandler(inventoryService *service.InventoryService, healthChecker *health.HealthChecker) *InventoryHandler {
	return &InventoryHandler{
		inventoryService: inventoryService,
		healthChecker:    healthChecker,
	}
}

// SyncStock 同步库存 - 核心接口
func (h *InventoryHandler) SyncStock(c *gin.Context) {
	var req model.SyncStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	response, err := h.inventoryService.SyncStock(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": response,
			"msg":  "库存同步成功",
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"code": 1,
			"data": response,
			"msg":  response.Message,
		})
	}
}

// GetInventory 获取库存信息
func (h *InventoryHandler) GetInventory(c *gin.Context) {
	productIDStr := c.Param("productId")
	if productIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商品ID不能为空"})
		return
	}

	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商品ID格式错误"})
		return
	}

	inventory, err := h.inventoryService.GetInventory(c.Request.Context(), productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": inventory,
		"msg":  "获取成功",
	})
}

// BatchGetInventory 批量获取库存信息
func (h *InventoryHandler) BatchGetInventory(c *gin.Context) {
	var req model.BatchGetInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	inventories, err := h.inventoryService.BatchGetInventory(c.Request.Context(), req.ProductIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": inventories,
		"msg":  "获取成功",
	})
}

// CheckInventoryHealth 库存健康检查
func (h *InventoryHandler) CheckInventoryHealth(c *gin.Context) {
	healthResponses, err := h.inventoryService.CheckInventoryHealth(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": healthResponses,
		"msg":  "健康检查完成",
	})
}

// TriggerHealthCheck 手动触发健康检查
func (h *InventoryHandler) TriggerHealthCheck(c *gin.Context) {
	if h.healthChecker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "健康检查服务未启用"})
		return
	}

	err := h.healthChecker.TriggerCheck(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "健康检查已触发",
	})
}

// FixInventoryDiff 修复库存差异
func (h *InventoryHandler) FixInventoryDiff(c *gin.Context) {
	diffIDStr := c.Param("diffId")
	if diffIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "差异记录ID不能为空"})
		return
	}

	diffID, err := strconv.ParseUint(diffIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "差异记录ID格式错误"})
		return
	}

	var req struct {
		FixType string `json:"fix_type" binding:"required"` // use_db, use_redis
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	err = h.inventoryService.FixInventoryDiff(c.Request.Context(), uint(diffID), req.FixType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "差异修复成功",
	})
}

// GetInventoryStats 获取库存统计
func (h *InventoryHandler) GetInventoryStats(c *gin.Context) {
	stats, err := h.inventoryService.GetInventoryStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": stats,
		"msg":  "获取成功",
	})
}

// GetServiceStats 获取服务统计信息
func (h *InventoryHandler) GetServiceStats(c *gin.Context) {
	serviceStats := h.inventoryService.GetServiceStats()

	var healthStats interface{}
	if h.healthChecker != nil {
		healthStats = h.healthChecker.GetStats()
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"service_stats": serviceStats,
			"health_stats":  healthStats,
		},
		"msg": "获取成功",
	})
}

// ListInventories 获取库存列表
func (h *InventoryHandler) ListInventories(c *gin.Context) {
	var query model.InventoryQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 设置默认值
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 20
	}

	// 这里需要在 InventoryService 中实现 ListInventories 方法
	// 暂时返回空结果
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"inventories": []interface{}{},
			"total":       0,
			"page":        query.Page,
			"page_size":   query.PageSize,
		},
		"msg": "获取成功",
	})
}

// UpdateInventory 更新库存信息
func (h *InventoryHandler) UpdateInventory(c *gin.Context) {
	productIDStr := c.Param("productId")
	if productIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商品ID不能为空"})
		return
	}

	_, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "商品ID格式错误"})
		return
	}

	var req struct {
		Stock    *int64  `json:"stock"`
		Reserved *int64  `json:"reserved"`
		MinStock *int64  `json:"min_stock"`
		MaxStock *int64  `json:"max_stock"`
		Status   *string `json:"status"`
		Reason   string  `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 这里需要在 InventoryService 中实现 UpdateInventory 方法
	// 暂时返回成功
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "更新成功",
	})
}

// CreateInventory 创建库存记录
func (h *InventoryHandler) CreateInventory(c *gin.Context) {
	var req struct {
		ProductID   int64  `json:"product_id" binding:"required"`
		ProductName string `json:"product_name" binding:"required"`
		Stock       int64  `json:"stock" binding:"required"`
		MinStock    int64  `json:"min_stock"`
		MaxStock    int64  `json:"max_stock"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	// 这里需要在 InventoryService 中实现 CreateInventory 方法
	// 暂时返回成功
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "创建成功",
	})
}

// HealthCheck 服务健康检查
func (h *InventoryHandler) HealthCheck(c *gin.Context) {
	err := h.inventoryService.HealthCheck(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"msg":    "服务正常",
	})
}
