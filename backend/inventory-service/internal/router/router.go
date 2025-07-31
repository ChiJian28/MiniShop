package router

import (
	"inventory-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func SetupRouter(inventoryHandler *handler.InventoryHandler) *gin.Engine {
	router := gin.Default()

	// 中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// 健康检查
	router.GET("/health", inventoryHandler.HealthCheck)

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		// 库存同步接口 - 核心接口
		v1.POST("/sync-stock", inventoryHandler.SyncStock)

		// 库存查询接口
		inventory := v1.Group("/inventory")
		{
			inventory.GET("/:productId", inventoryHandler.GetInventory)    // 获取单个库存
			inventory.POST("/batch", inventoryHandler.BatchGetInventory)   // 批量获取库存
			inventory.GET("", inventoryHandler.ListInventories)            // 获取库存列表
			inventory.POST("", inventoryHandler.CreateInventory)           // 创建库存记录
			inventory.PUT("/:productId", inventoryHandler.UpdateInventory) // 更新库存信息
		}

		// 健康检查相关接口
		health := v1.Group("/health")
		{
			health.GET("/check", inventoryHandler.CheckInventoryHealth)    // 库存健康检查
			health.POST("/trigger", inventoryHandler.TriggerHealthCheck)   // 手动触发健康检查
			health.POST("/fix/:diffId", inventoryHandler.FixInventoryDiff) // 修复库存差异
		}

		// 统计相关接口
		stats := v1.Group("/stats")
		{
			stats.GET("/inventory", inventoryHandler.GetInventoryStats) // 获取库存统计
			stats.GET("/service", inventoryHandler.GetServiceStats)     // 获取服务统计
		}
	}

	return router
}
