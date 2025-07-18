package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"seckill-service/internal/seckill"
	"seckill-service/internal/service"

	"github.com/gin-gonic/gin"
)

// REST API 处理程序
type Handler struct {
	seckillService *service.SeckillService
}

// 创建处理程序
func NewHandler(seckillService *service.SeckillService) *Handler {
	return &Handler{
		seckillService: seckillService,
	}
}

// 秒杀请求
func (h *Handler) SeckillPurchase(c *gin.Context) {
	var req seckill.SeckillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// 参数验证
	if req.ProductID <= 0 || req.UserID <= 0 || req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid parameters",
		})
		return
	}

	// 处理秒杀请求
	result, err := h.seckillService.ProcessSeckill(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal server error",
			"details": err.Error(),
		})
		return
	}

	// 根据结果返回不同的 HTTP 状态码
	var statusCode int
	switch result.Code {
	case seckill.ResultSuccess:
		statusCode = http.StatusOK
	case seckill.ResultSystemBusy:
		statusCode = http.StatusTooManyRequests
	case seckill.ResultInsufficientStock:
		statusCode = http.StatusConflict
	case seckill.ResultUserAlreadyBought:
		statusCode = http.StatusConflict
	default:
		statusCode = http.StatusBadRequest
	}

	c.JSON(statusCode, result)
}

// 异步秒杀请求
func (h *Handler) SeckillPurchaseAsync(c *gin.Context) {
	var req seckill.SeckillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// 参数验证
	if req.ProductID <= 0 || req.UserID <= 0 || req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid parameters",
		})
		return
	}

	// 异步处理秒杀请求
	err := h.seckillService.ProcessSeckillAsync(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Service unavailable",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":    "Request accepted and will be processed asynchronously",
		"product_id": req.ProductID,
		"user_id":    req.UserID,
	})
}

// 预热活动
func (h *Handler) PrewarmActivity(c *gin.Context) {
	var activity seckill.SeckillActivity
	if err := c.ShouldBindJSON(&activity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// 参数验证
	if activity.ProductID <= 0 || activity.Stock <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid parameters",
		})
		return
	}

	err := h.seckillService.PrewarmActivity(c.Request.Context(), &activity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to prewarm activity",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Activity prewarmed successfully",
		"product_id": activity.ProductID,
		"stock":      activity.Stock,
	})
}

// 获取秒杀统计信息
func (h *Handler) GetSeckillStats(c *gin.Context) {
	productIDStr := c.Param("productId")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	stats, err := h.seckillService.GetSeckillStats(c.Request.Context(), productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get seckill stats",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// 检查用户购买状态
func (h *Handler) IsUserPurchased(c *gin.Context) {
	productIDStr := c.Param("productId")
	userIDStr := c.Param("userId")

	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	purchased, err := h.seckillService.IsUserPurchased(c.Request.Context(), productID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to check user purchase status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"product_id": productID,
		"user_id":    userID,
		"purchased":  purchased,
	})
}

// 获取用户购买信息
func (h *Handler) GetUserPurchaseInfo(c *gin.Context) {
	productIDStr := c.Param("productId")
	userIDStr := c.Param("userId")

	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	info, err := h.seckillService.GetUserPurchaseInfo(c.Request.Context(), productID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get user purchase info",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, info)
}

// 清理活动数据
func (h *Handler) CleanupActivity(c *gin.Context) {
	productIDStr := c.Param("productId")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid product ID",
		})
		return
	}

	err = h.seckillService.CleanupActivity(c.Request.Context(), productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to cleanup activity",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Activity cleaned up successfully",
		"product_id": productID,
	})
}

// 获取服务统计信息
func (h *Handler) GetServiceStats(c *gin.Context) {
	stats := h.seckillService.GetServiceStats()
	c.JSON(http.StatusOK, gin.H{
		"service_stats":         stats,
		"queue_stats":           h.seckillService.GetQueueStats(),
		"circuit_breaker_state": h.seckillService.GetCircuitBreakerState().String(),
		"limiter_tokens":        h.seckillService.GetLimiterTokens(),
	})
}

// 健康检查
func (h *Handler) HealthCheck(c *gin.Context) {
	err := h.seckillService.HealthCheck(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "unhealthy",
			"error":     err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "seckill-service",
	})
}

// 中间件：请求日志
func (h *Handler) RequestLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s %s\n",
			param.TimeStamp.Format(time.RFC3339),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
		)
	})
}

// 中间件：CORS
func (h *Handler) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// 中间件：限流
func (h *Handler) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 这里可以添加基于 IP 的限流逻辑
		// 暂时跳过，因为服务层已经有限流
		c.Next()
	}
}

// 中间件：错误处理
func (h *Handler) ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Internal server error",
					"details": fmt.Sprintf("%v", err),
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
