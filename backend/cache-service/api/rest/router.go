package rest

import (
	"github.com/gin-gonic/gin"
)

func SetupRouter(handler *Handler) *gin.Engine {
	router := gin.Default()

	// 中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// 健康检查
	router.GET("/health", handler.HealthCheck)

	// API 版本组
	v1 := router.Group("/api/v1")
	{
		// 基础缓存操作
		cache := v1.Group("/cache")
		{
			cache.GET("/:key", handler.Get)
			cache.POST("/", handler.Set)
			cache.DELETE("/", handler.Del)
		}

		// 秒杀相关操作
		seckill := v1.Group("/seckill")
		{
			// 预加载秒杀活动
			seckill.POST("/activity", handler.PreloadSeckillActivity)

			// 获取秒杀活动信息
			seckill.GET("/activity/:productId", handler.GetSeckillActivity)

			// 获取库存
			seckill.GET("/stock/:productId", handler.GetSeckillStock)

			// 秒杀购买
			seckill.POST("/purchase", handler.SeckillPurchase)

			// 检查用户是否已购买
			seckill.GET("/purchased/:productId/:userId", handler.IsUserPurchased)

			// 获取用户购买信息
			seckill.GET("/purchase/:productId/:userId", handler.GetUserPurchaseInfo)

			// 获取购买用户数量
			seckill.GET("/count/:productId", handler.GetPurchaseUserCount)

			// 清理秒杀数据
			seckill.DELETE("/cleanup/:productId", handler.CleanupSeckillData)
		}
	}

	return router
}
