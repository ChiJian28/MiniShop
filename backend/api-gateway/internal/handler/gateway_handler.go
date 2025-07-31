package handler

import (
	"net/http"

	"api-gateway/internal/middleware"
	"api-gateway/internal/proxy"

	"github.com/gin-gonic/gin"
)

// GatewayHandler 网关处理器
type GatewayHandler struct {
	proxy       *proxy.ServiceProxy
	rateLimiter *middleware.RateLimiter
	auth        *middleware.AuthMiddleware
}

// NewGatewayHandler 创建网关处理器
func NewGatewayHandler(
	proxy *proxy.ServiceProxy,
	rateLimiter *middleware.RateLimiter,
	auth *middleware.AuthMiddleware,
) *GatewayHandler {
	return &GatewayHandler{
		proxy:       proxy,
		rateLimiter: rateLimiter,
		auth:        auth,
	}
}

// HealthCheck 网关健康检查
func (h *GatewayHandler) HealthCheck(c *gin.Context) {
	// 检查后端服务健康状态
	serviceHealth := h.proxy.HealthCheck(c.Request.Context())

	// 统计健康服务数量
	healthyCount := 0
	totalCount := len(serviceHealth)

	for _, health := range serviceHealth {
		if healthMap, ok := health.(map[string]interface{}); ok {
			if status, exists := healthMap["status"]; exists && status == "healthy" {
				healthyCount++
			}
		}
	}

	// 确定网关整体健康状态
	gatewayStatus := "healthy"
	httpStatus := http.StatusOK

	if healthyCount == 0 {
		gatewayStatus = "critical"
		httpStatus = http.StatusServiceUnavailable
	} else if healthyCount < totalCount {
		gatewayStatus = "degraded"
		httpStatus = http.StatusPartialContent
	}

	c.JSON(httpStatus, gin.H{
		"status":   gatewayStatus,
		"message":  "API Gateway Health Check",
		"services": serviceHealth,
		"summary": gin.H{
			"healthy_services": healthyCount,
			"total_services":   totalCount,
		},
	})
}

// GetStats 获取网关统计信息
func (h *GatewayHandler) GetStats(c *gin.Context) {
	stats := gin.H{
		"services": h.proxy.GetServiceStats(),
	}

	// 添加限流统计
	if h.rateLimiter != nil {
		rateLimitStats, err := h.rateLimiter.GetRateLimitStats(c.Request.Context())
		if err == nil {
			stats["rate_limit"] = rateLimitStats
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": stats,
		"msg":  "获取统计信息成功",
	})
}

// Login 登录接口
func (h *GatewayHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "参数错误: " + err.Error(),
			"code":  400,
		})
		return
	}

	// 这里应该验证用户名密码，简化处理
	if req.Username == "admin" && req.Password == "password" {
		// 生成JWT token
		token, err := h.auth.GenerateJWT(1, req.Username, []string{"admin"})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "生成token失败",
				"code":  500,
			})
			return
		}

		// 生成刷新token
		refreshToken, err := h.auth.GenerateRefreshToken(1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "生成刷新token失败",
				"code":  500,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"token":         token,
				"refresh_token": refreshToken,
				"user_id":       1,
				"username":      req.Username,
				"roles":         []string{"admin"},
			},
			"msg": "登录成功",
		})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "用户名或密码错误",
			"code":  401,
		})
	}
}

// RefreshToken 刷新token
func (h *GatewayHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "参数错误: " + err.Error(),
			"code":  400,
		})
		return
	}

	// 刷新token
	newToken, err := h.auth.RefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "刷新token失败: " + err.Error(),
			"code":  401,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"token": newToken,
		},
		"msg": "刷新token成功",
	})
}

// NotFound 404处理器
func (h *GatewayHandler) NotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"error": "接口不存在",
		"code":  404,
		"path":  c.Request.URL.Path,
	})
}

// MethodNotAllowed 405处理器
func (h *GatewayHandler) MethodNotAllowed(c *gin.Context) {
	c.JSON(http.StatusMethodNotAllowed, gin.H{
		"error":  "请求方法不允许",
		"code":   405,
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
	})
}
