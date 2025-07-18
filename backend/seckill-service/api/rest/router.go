package rest

import (
	"github.com/gin-gonic/gin"
)

// 设置路由
func SetupRouter(handler *Handler) *gin.Engine {
	router := gin.New()

	// 添加中间件
	router.Use(handler.RequestLogger())
	router.Use(handler.CORS())
	router.Use(handler.ErrorHandler())
	router.Use(handler.RateLimit())

	// 健康检查
	router.GET("/health", handler.HealthCheck)

	// API 版本组
	v1 := router.Group("/api/v1")
	{
		// 秒杀相关路由
		seckill := v1.Group("/seckill")
		{
			// 同步秒杀
			seckill.POST("/purchase", handler.SeckillPurchase)

			// 异步秒杀
			seckill.POST("/purchase/async", handler.SeckillPurchaseAsync)

			// 预热活动
			seckill.POST("/activity/prewarm", handler.PrewarmActivity)

			// 获取秒杀统计信息
			seckill.GET("/stats/:productId", handler.GetSeckillStats)

			// 检查用户购买状态
			seckill.GET("/purchased/:productId/:userId", handler.IsUserPurchased)

			// 获取用户购买信息
			seckill.GET("/purchase/:productId/:userId", handler.GetUserPurchaseInfo)

			// 清理活动数据
			seckill.DELETE("/activity/:productId", handler.CleanupActivity)
		}

		// 系统监控相关路由
		system := v1.Group("/system")
		{
			// 获取服务统计信息
			system.GET("/stats", handler.GetServiceStats)

			// 健康检查
			system.GET("/health", handler.HealthCheck)
		}
	}

	return router
}
