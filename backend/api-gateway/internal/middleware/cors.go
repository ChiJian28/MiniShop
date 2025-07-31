package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"api-gateway/internal/config"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware CORS中间件
type CORSMiddleware struct {
	config *config.CORSConfig
}

// NewCORSMiddleware 创建CORS中间件
func NewCORSMiddleware(cfg *config.CORSConfig) *CORSMiddleware {
	return &CORSMiddleware{
		config: cfg,
	}
}

// CORS 跨域中间件
func (cm *CORSMiddleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cm.config.Enable {
			c.Next()
			return
		}

		origin := c.Request.Header.Get("Origin")

		// 检查允许的源
		if cm.isOriginAllowed(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if cm.isAllowAll() {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		// 设置允许的方法
		if len(cm.config.AllowedMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(cm.config.AllowedMethods, ", "))
		}

		// 设置允许的头部
		if len(cm.config.AllowedHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(cm.config.AllowedHeaders, ", "))
		}

		// 设置暴露的头部
		if len(cm.config.ExposedHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(cm.config.ExposedHeaders, ", "))
		}

		// 设置是否允许凭证
		if cm.config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// 设置预检请求的缓存时间
		if cm.config.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", strconv.Itoa(cm.config.MaxAge))
		}

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isOriginAllowed 检查源是否被允许
func (cm *CORSMiddleware) isOriginAllowed(origin string) bool {
	for _, allowedOrigin := range cm.config.AllowedOrigins {
		if allowedOrigin == "*" {
			return true
		}
		if allowedOrigin == origin {
			return true
		}
		// 支持通配符匹配
		if strings.Contains(allowedOrigin, "*") {
			if cm.matchWildcard(allowedOrigin, origin) {
				return true
			}
		}
	}
	return false
}

// isAllowAll 检查是否允许所有源
func (cm *CORSMiddleware) isAllowAll() bool {
	for _, origin := range cm.config.AllowedOrigins {
		if origin == "*" {
			return true
		}
	}
	return false
}

// matchWildcard 通配符匹配
func (cm *CORSMiddleware) matchWildcard(pattern, str string) bool {
	// 简单的通配符匹配实现
	if pattern == "*" {
		return true
	}

	if strings.HasPrefix(pattern, "*.") {
		domain := strings.TrimPrefix(pattern, "*.")
		return strings.HasSuffix(str, "."+domain) || str == domain
	}

	return false
}
