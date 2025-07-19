package router

import (
	"order-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func SetupRouter(orderHandler *handler.OrderHandler) *gin.Engine {
	router := gin.Default()

	// 中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// 健康检查
	router.GET("/health", orderHandler.HealthCheck)

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		// 订单相关路由
		orders := v1.Group("/orders")
		{
			orders.GET("/:orderId", orderHandler.GetOrder)                 // 获取订单详情
			orders.PUT("/:orderId/status", orderHandler.UpdateOrderStatus) // 更新订单状态
			orders.PUT("/:orderId/cancel", orderHandler.CancelOrder)       // 取消订单
		}

		// 用户订单路由
		users := v1.Group("/users")
		{
			users.GET("/:userId/orders", orderHandler.GetUserOrders) // 获取用户订单列表
		}

		// 统计相关路由
		stats := v1.Group("/stats")
		{
			stats.GET("/orders", orderHandler.GetOrderStats) // 获取订单统计
		}

		// 失败补偿相关路由
		failures := v1.Group("/failures")
		{
			failures.GET("", orderHandler.GetFailedOrders)                    // 获取失败订单列表
			failures.POST("/:failureId/retry", orderHandler.RetryFailedOrder) // 重试失败订单
		}
	}

	return router
}
