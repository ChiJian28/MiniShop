package handler

import (
	"net/http"
	"strconv"

	"order-service/internal/service"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

// GetOrder 获取订单详情
func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("orderId")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "订单ID不能为空"})
		return
	}

	order, err := h.orderService.GetOrderByID(orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if order == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "订单不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": order,
		"msg":  "获取成功",
	})
}

// GetUserOrders 获取用户订单列表
func (h *OrderHandler) GetUserOrders(c *gin.Context) {
	userIDStr := c.Param("userId")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID不能为空"})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式错误"})
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	status := c.Query("status")

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}

	orders, total, err := h.orderService.GetUserOrders(userID, page, pageSize, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"orders":   orders,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
		"msg": "获取成功",
	})
}

// UpdateOrderStatus 更新订单状态
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("orderId")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "订单ID不能为空"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
		Remark string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	err := h.orderService.UpdateOrderStatus(c.Request.Context(), orderID, req.Status)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "更新成功",
	})
}

// CancelOrder 取消订单
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("orderId")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "订单ID不能为空"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	err := h.orderService.CancelOrder(orderID, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "取消成功",
	})
}

// GetOrderStats 获取订单统计
func (h *OrderHandler) GetOrderStats(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日期参数不能为空"})
		return
	}

	stats, err := h.orderService.GetOrderStats(date)
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

// RetryFailedOrder 重试失败订单
func (h *OrderHandler) RetryFailedOrder(c *gin.Context) {
	failureIDStr := c.Param("failureId")
	if failureIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "失败记录ID不能为空"})
		return
	}

	failureID, err := strconv.ParseUint(failureIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "失败记录ID格式错误"})
		return
	}

	err = h.orderService.RetryFailedOrder(failureID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "重试成功",
	})
}

// GetFailedOrders 获取失败订单列表
func (h *OrderHandler) GetFailedOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	status := c.Query("status")

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}

	failures, total, err := h.orderService.GetFailedOrders(page, pageSize, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"failures": failures,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
		"msg": "获取成功",
	})
}

// HealthCheck 健康检查
func (h *OrderHandler) HealthCheck(c *gin.Context) {
	isHealthy := h.orderService.HealthCheck(c.Request.Context())

	if isHealthy {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"msg":    "服务正常",
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"msg":    "服务异常",
		})
	}
}
