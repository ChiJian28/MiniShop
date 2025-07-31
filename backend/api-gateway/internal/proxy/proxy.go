package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"api-gateway/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ServiceProxy 服务代理
type ServiceProxy struct {
	config   *config.Config
	logger   *logrus.Logger
	services map[string]*ServiceClient
}

// ServiceClient 服务客户端
type ServiceClient struct {
	name       string
	baseURL    *url.URL
	httpClient *http.Client
	config     config.ServiceConfig
}

// NewServiceProxy 创建服务代理
func NewServiceProxy(cfg *config.Config, logger *logrus.Logger) *ServiceProxy {
	sp := &ServiceProxy{
		config:   cfg,
		logger:   logger,
		services: make(map[string]*ServiceClient),
	}

	// 初始化服务客户端
	sp.initServiceClients()

	return sp
}

// initServiceClients 初始化服务客户端
func (sp *ServiceProxy) initServiceClients() {
	services := map[string]config.ServiceConfig{
		"cache-service":     sp.config.Services.CacheService,
		"seckill-service":   sp.config.Services.SeckillService,
		"order-service":     sp.config.Services.OrderService,
		"inventory-service": sp.config.Services.InventoryService,
	}

	for name, cfg := range services {
		baseURL, err := url.Parse(cfg.URL)
		if err != nil {
			sp.logger.WithError(err).Errorf("解析服务URL失败: %s", name)
			continue
		}

		client := &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:       cfg.MaxIdleConns,
				MaxConnsPerHost:    cfg.MaxConnsPerHost,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: false,
				DisableKeepAlives:  false,
			},
		}

		sp.services[name] = &ServiceClient{
			name:       name,
			baseURL:    baseURL,
			httpClient: client,
			config:     cfg,
		}

		sp.logger.Infof("初始化服务客户端: %s -> %s", name, cfg.URL)
	}
}

// ProxyHandler 代理处理器
func (sp *ServiceProxy) ProxyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 根据路径前缀确定目标服务
		serviceName := sp.getServiceByPath(c.Request.URL.Path)
		if serviceName == "" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "服务未找到",
				"code":  404,
			})
			return
		}

		// 获取服务客户端
		client, exists := sp.services[serviceName]
		if !exists {
			sp.logger.Errorf("服务客户端未找到: %s", serviceName)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "服务不可用",
				"code":  503,
			})
			return
		}

		// 执行代理请求
		sp.proxyRequest(c, client)
	}
}

// getServiceByPath 根据路径获取服务名
func (sp *ServiceProxy) getServiceByPath(path string) string {
	for prefix, serviceName := range sp.config.Routing.PrefixMapping {
		if strings.HasPrefix(path, prefix) {
			return serviceName
		}
	}
	return ""
}

// proxyRequest 执行代理请求
func (sp *ServiceProxy) proxyRequest(c *gin.Context, client *ServiceClient) {
	// 构建目标URL
	targetURL := sp.buildTargetURL(c, client)

	// 读取请求体
	var body []byte
	if c.Request.Body != nil {
		var err error
		body, err = io.ReadAll(c.Request.Body)
		if err != nil {
			sp.logger.WithError(err).Error("读取请求体失败")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "读取请求体失败",
				"code":  500,
			})
			return
		}
		c.Request.Body.Close()
	}

	// 创建新请求
	req, err := http.NewRequestWithContext(
		c.Request.Context(),
		c.Request.Method,
		targetURL,
		bytes.NewReader(body),
	)
	if err != nil {
		sp.logger.WithError(err).Error("创建代理请求失败")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建代理请求失败",
			"code":  500,
		})
		return
	}

	// 复制请求头
	sp.copyHeaders(c.Request.Header, req.Header)

	// 添加代理头
	req.Header.Set("X-Forwarded-For", c.ClientIP())
	req.Header.Set("X-Forwarded-Proto", sp.getScheme(c))
	req.Header.Set("X-Forwarded-Host", c.Request.Host)
	req.Header.Set("X-Real-IP", c.ClientIP())

	// 添加追踪头
	if traceID := c.GetHeader("X-Trace-ID"); traceID == "" {
		req.Header.Set("X-Trace-ID", sp.generateTraceID())
	} else {
		req.Header.Set("X-Trace-ID", traceID)
	}

	// 记录请求开始时间
	startTime := time.Now()

	// 执行请求
	resp, err := client.httpClient.Do(req)
	if err != nil {
		duration := time.Since(startTime)
		sp.logger.WithFields(logrus.Fields{
			"service":  client.name,
			"method":   c.Request.Method,
			"path":     c.Request.URL.Path,
			"duration": duration,
			"error":    err.Error(),
		}).Error("代理请求失败")

		c.JSON(http.StatusBadGateway, gin.H{
			"error": "服务请求失败",
			"code":  502,
		})
		return
	}
	defer resp.Body.Close()

	// 记录请求完成
	duration := time.Since(startTime)
	sp.logger.WithFields(logrus.Fields{
		"service":    client.name,
		"method":     c.Request.Method,
		"path":       c.Request.URL.Path,
		"status":     resp.StatusCode,
		"duration":   duration,
		"target_url": targetURL,
	}).Info("代理请求完成")

	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// 设置状态码
	c.Status(resp.StatusCode)

	// 复制响应体
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		sp.logger.WithError(err).Error("复制响应体失败")
	}
}

// buildTargetURL 构建目标URL
func (sp *ServiceProxy) buildTargetURL(c *gin.Context, client *ServiceClient) string {
	// 移除路径前缀
	path := c.Request.URL.Path
	for prefix := range sp.config.Routing.PrefixMapping {
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			break
		}
	}

	// 构建完整URL
	targetURL := client.baseURL.Scheme + "://" + client.baseURL.Host + path

	// 添加查询参数
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	return targetURL
}

// copyHeaders 复制请求头
func (sp *ServiceProxy) copyHeaders(src, dst http.Header) {
	// 需要跳过的头部
	skipHeaders := map[string]bool{
		"Connection":          true,
		"Proxy-Connection":    true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"Te":                  true,
		"Trailers":            true,
		"Transfer-Encoding":   true,
		"Upgrade":             true,
	}

	for key, values := range src {
		if skipHeaders[key] {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

// getScheme 获取请求协议
func (sp *ServiceProxy) getScheme(c *gin.Context) string {
	if c.Request.TLS != nil {
		return "https"
	}

	if scheme := c.GetHeader("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}

	return "http"
}

// generateTraceID 生成追踪ID
func (sp *ServiceProxy) generateTraceID() string {
	return fmt.Sprintf("gateway-%d", time.Now().UnixNano())
}

// HealthCheck 健康检查
func (sp *ServiceProxy) HealthCheck(ctx context.Context) map[string]interface{} {
	results := make(map[string]interface{})

	for name, client := range sp.services {
		healthPath := sp.config.Routing.HealthChecks[name]
		if healthPath == "" {
			healthPath = "/health"
		}

		targetURL := client.baseURL.Scheme + "://" + client.baseURL.Host + healthPath

		// 创建健康检查请求
		req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
		if err != nil {
			results[name] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
			continue
		}

		// 设置超时
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
		req = req.WithContext(ctxWithTimeout)

		startTime := time.Now()
		resp, err := client.httpClient.Do(req)
		duration := time.Since(startTime)
		cancel()

		if err != nil {
			results[name] = map[string]interface{}{
				"status":   "error",
				"error":    err.Error(),
				"duration": duration.String(),
			}
			continue
		}
		resp.Body.Close()

		status := "healthy"
		if resp.StatusCode >= 400 {
			status = "unhealthy"
		}

		results[name] = map[string]interface{}{
			"status":      status,
			"status_code": resp.StatusCode,
			"duration":    duration.String(),
			"url":         targetURL,
		}
	}

	return results
}

// GetServiceStats 获取服务统计信息
func (sp *ServiceProxy) GetServiceStats() map[string]interface{} {
	stats := make(map[string]interface{})

	for name, client := range sp.services {
		stats[name] = map[string]interface{}{
			"base_url":           client.baseURL.String(),
			"timeout":            client.config.Timeout.String(),
			"max_idle_conns":     client.config.MaxIdleConns,
			"max_conns_per_host": client.config.MaxConnsPerHost,
		}
	}

	return stats
}
