package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"api-gateway/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// RateLimiter 限流器
type RateLimiter struct {
	config      *config.RateLimitConfig
	redisClient *redis.Client
	logger      *logrus.Logger

	// 全局限流器
	globalLimiter *rate.Limiter

	// 用户限流器缓存
	userLimiters map[string]*rate.Limiter

	// IP限流器缓存
	ipLimiters map[string]*rate.Limiter
}

// NewRateLimiter 创建限流器
func NewRateLimiter(cfg *config.RateLimitConfig, redisClient *redis.Client, logger *logrus.Logger) *RateLimiter {
	rl := &RateLimiter{
		config:       cfg,
		redisClient:  redisClient,
		logger:       logger,
		userLimiters: make(map[string]*rate.Limiter),
		ipLimiters:   make(map[string]*rate.Limiter),
	}

	// 初始化全局限流器
	if cfg.Enable {
		rl.globalLimiter = rate.NewLimiter(
			rate.Limit(cfg.Global.RequestsPerSecond),
			cfg.Global.Burst,
		)
	}

	return rl
}

// GlobalRateLimit 全局限流中间件
func (rl *RateLimiter) GlobalRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.config.Enable {
			c.Next()
			return
		}

		if !rl.globalLimiter.Allow() {
			rl.logger.WithFields(logrus.Fields{
				"ip":     c.ClientIP(),
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			}).Warn("全局限流触发")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁，请稍后再试",
				"code":  429,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// UserRateLimit 用户限流中间件
func (rl *RateLimiter) UserRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.config.Enable {
			c.Next()
			return
		}

		// 获取用户ID
		userID := rl.getUserID(c)
		if userID == "" {
			c.Next()
			return
		}

		// 使用Redis实现分布式限流
		allowed, err := rl.checkUserRateLimit(c.Request.Context(), userID)
		if err != nil {
			rl.logger.WithError(err).Error("用户限流检查失败")
			c.Next()
			return
		}

		if !allowed {
			rl.logger.WithFields(logrus.Fields{
				"user_id": userID,
				"ip":      c.ClientIP(),
				"path":    c.Request.URL.Path,
			}).Warn("用户限流触发")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "用户请求过于频繁，请稍后再试",
				"code":  429,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// IPRateLimit IP限流中间件
func (rl *RateLimiter) IPRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.config.Enable {
			c.Next()
			return
		}

		ip := c.ClientIP()

		// 使用Redis实现分布式限流
		allowed, err := rl.checkIPRateLimit(c.Request.Context(), ip)
		if err != nil {
			rl.logger.WithError(err).Error("IP限流检查失败")
			c.Next()
			return
		}

		if !allowed {
			rl.logger.WithFields(logrus.Fields{
				"ip":   ip,
				"path": c.Request.URL.Path,
			}).Warn("IP限流触发")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "IP请求过于频繁，请稍后再试",
				"code":  429,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// EndpointRateLimit 接口限流中间件
func (rl *RateLimiter) EndpointRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.config.Enable {
			c.Next()
			return
		}

		path := c.Request.URL.Path

		// 检查是否有针对该接口的限流配置
		endpointConfig, exists := rl.config.Endpoints[path]
		if !exists {
			c.Next()
			return
		}

		// 使用Redis实现分布式限流
		allowed, err := rl.checkEndpointRateLimit(c.Request.Context(), path, endpointConfig)
		if err != nil {
			rl.logger.WithError(err).Error("接口限流检查失败")
			c.Next()
			return
		}

		if !allowed {
			rl.logger.WithFields(logrus.Fields{
				"ip":       c.ClientIP(),
				"path":     path,
				"endpoint": path,
			}).Warn("接口限流触发")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "接口请求过于频繁，请稍后再试",
				"code":  429,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// checkUserRateLimit 检查用户限流
func (rl *RateLimiter) checkUserRateLimit(ctx context.Context, userID string) (bool, error) {
	key := fmt.Sprintf("rate_limit:user:%s", userID)
	return rl.checkRedisRateLimit(ctx, key, rl.config.User.RequestsPerSecond, rl.config.User.Burst, rl.config.User.Window)
}

// checkIPRateLimit 检查IP限流
func (rl *RateLimiter) checkIPRateLimit(ctx context.Context, ip string) (bool, error) {
	key := fmt.Sprintf("rate_limit:ip:%s", ip)
	return rl.checkRedisRateLimit(ctx, key, rl.config.IP.RequestsPerSecond, rl.config.IP.Burst, rl.config.IP.Window)
}

// checkEndpointRateLimit 检查接口限流
func (rl *RateLimiter) checkEndpointRateLimit(ctx context.Context, path string, cfg config.EndpointLimitConfig) (bool, error) {
	key := fmt.Sprintf("rate_limit:endpoint:%s", strings.ReplaceAll(path, "/", ":"))
	return rl.checkRedisRateLimit(ctx, key, cfg.RequestsPerSecond, cfg.Burst, time.Minute)
}

// checkRedisRateLimit 使用Redis实现分布式限流
func (rl *RateLimiter) checkRedisRateLimit(ctx context.Context, key string, rps float64, burst int, window time.Duration) (bool, error) {
	_ = rps

	now := time.Now()
	pipe := rl.redisClient.Pipeline()

	// 使用滑动窗口算法
	windowStart := now.Add(-window)

	// 删除过期的记录
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	// 获取当前窗口内的请求数
	pipe.ZCard(ctx, key)

	// 添加当前请求
	pipe.ZAdd(ctx, key, &redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})

	// 设置过期时间
	pipe.Expire(ctx, key, window)

	results, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	// 获取当前请求数
	count := results[1].(*redis.IntCmd).Val()

	// 检查是否超过限制
	return count <= int64(burst), nil
}

// getUserID 从请求中获取用户ID
func (rl *RateLimiter) getUserID(c *gin.Context) string {
	// 从JWT token中获取用户ID
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok {
			return uid
		}
		if uid, ok := userID.(int64); ok {
			return strconv.FormatInt(uid, 10)
		}
	}

	// 从header中获取用户ID
	if userID := c.GetHeader("X-User-ID"); userID != "" {
		return userID
	}

	return ""
}

// GetRateLimitStats 获取限流统计信息
func (rl *RateLimiter) GetRateLimitStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 全局限流器状态
	if rl.globalLimiter != nil {
		stats["global"] = map[string]interface{}{
			"limit": rl.globalLimiter.Limit(),
			"burst": rl.globalLimiter.Burst(),
		}
	}

	// Redis中的限流统计
	keys, err := rl.redisClient.Keys(ctx, "rate_limit:*").Result()
	if err != nil {
		return stats, err
	}

	stats["active_limits"] = len(keys)

	// 分类统计
	userLimits := 0
	ipLimits := 0
	endpointLimits := 0

	for _, key := range keys {
		if strings.HasPrefix(key, "rate_limit:user:") {
			userLimits++
		} else if strings.HasPrefix(key, "rate_limit:ip:") {
			ipLimits++
		} else if strings.HasPrefix(key, "rate_limit:endpoint:") {
			endpointLimits++
		}
	}

	stats["user_limits"] = userLimits
	stats["ip_limits"] = ipLimits
	stats["endpoint_limits"] = endpointLimits

	return stats, nil
}
