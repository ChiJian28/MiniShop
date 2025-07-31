package router

import (
	"api-gateway/internal/config"
	"api-gateway/internal/handler"
	"api-gateway/internal/middleware"
	"api-gateway/internal/proxy"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter(
	cfg *config.Config,
	gatewayHandler *handler.GatewayHandler,
	serviceProxy *proxy.ServiceProxy,
	corsMiddleware *middleware.CORSMiddleware,
	rateLimiter *middleware.RateLimiter,
	authMiddleware *middleware.AuthMiddleware,
) *gin.Engine {
	// 设置Gin模式
	if cfg.Log.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// 基础中间件
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS中间件
	if corsMiddleware != nil {
		router.Use(corsMiddleware.CORS())
	}

	// 全局限流中间件
	if rateLimiter != nil {
		router.Use(rateLimiter.GlobalRateLimit())
		router.Use(rateLimiter.IPRateLimit())
	}

	// 错误处理
	router.NoRoute(gatewayHandler.NotFound)
	router.NoMethod(gatewayHandler.MethodNotAllowed)

	// 健康检查和管理接口（不需要认证）
	router.GET("/health", gatewayHandler.HealthCheck)
	router.GET("/stats", gatewayHandler.GetStats)

	// API路由组
	api := router.Group("/api/v1")

	// 认证相关接口（不需要认证中间件）
	auth := api.Group("/auth")
	{
		auth.POST("/login", gatewayHandler.Login)
		auth.POST("/refresh", gatewayHandler.RefreshToken)
	}

	// 需要认证的API路由
	protectedAPI := api.Group("")
	
	// 应用认证中间件（除了白名单路径）
	if authMiddleware != nil {
		// JWT认证
		protectedAPI.Use(authMiddleware.JWTAuth())
		// 签名校验
		protectedAPI.Use(authMiddleware.SignatureAuth())
	}

	// 用户限流和接口限流
	if rateLimiter != nil {
		protectedAPI.Use(rateLimiter.UserRateLimit())
		protectedAPI.Use(rateLimiter.EndpointRateLimit())
	}

	// 代理到后端服务（除了认证路径）
	protectedAPI.Any("/cache/*path", serviceProxy.ProxyHandler())
	protectedAPI.Any("/seckill/*path", serviceProxy.ProxyHandler())
	protectedAPI.Any("/order/*path", serviceProxy.ProxyHandler())
	protectedAPI.Any("/inventory/*path", serviceProxy.ProxyHandler())

	return router
}

// SetupMonitoringRouter 设置监控路由
func SetupMonitoringRouter(cfg *config.Config) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	// Prometheus metrics
	// 这里可以添加Prometheus metrics处理器
	router.GET("/metrics", func(c *gin.Context) {
		// TODO: 实现Prometheus metrics
		c.String(200, "# Metrics endpoint")
	})

	return router
}
