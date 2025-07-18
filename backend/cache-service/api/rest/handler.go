package rest

import (
	"net/http"
	"strconv"
	"time"

	"cache-service/internal/seckill"
	"cache-service/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	cacheService *service.CacheService
}

func NewHandler(cacheService *service.CacheService) *Handler {
	return &Handler{
		cacheService: cacheService,
	}
}

// 基础缓存操作
func (h *Handler) Get(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
		return
	}

	value, err := h.cacheService.Get(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key":   key,
		"value": value,
	})
}

func (h *Handler) Set(c *gin.Context) {
	var req struct {
		Key        string `json:"key" binding:"required"`
		Value      string `json:"value" binding:"required"`
		Expiration int64  `json:"expiration"` // seconds
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	expiration := time.Duration(req.Expiration) * time.Second
	if req.Expiration == 0 {
		expiration = 0 // 不过期
	}

	err := h.cacheService.Set(c.Request.Context(), req.Key, req.Value, expiration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

func (h *Handler) Del(c *gin.Context) {
	var req struct {
		Keys []string `json:"keys" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.cacheService.Del(c.Request.Context(), req.Keys...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// 秒杀相关操作
func (h *Handler) PreloadSeckillActivity(c *gin.Context) {
	var activity seckill.SeckillActivity
	if err := c.ShouldBindJSON(&activity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.cacheService.PreloadSeckillActivity(c.Request.Context(), &activity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

func (h *Handler) GetSeckillStock(c *gin.Context) {
	productIDStr := c.Param("productId")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	stock, err := h.cacheService.GetSeckillStock(c.Request.Context(), productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"product_id": productID,
		"stock":      stock,
	})
}

func (h *Handler) SeckillPurchase(c *gin.Context) {
	var req struct {
		ProductID int64 `json:"product_id" binding:"required"`
		UserID    int64 `json:"user_id" binding:"required"`
		Quantity  int64 `json:"quantity" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.cacheService.SeckillPurchase(c.Request.Context(), req.ProductID, req.UserID, req.Quantity)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "purchase success"})
}

func (h *Handler) IsUserPurchased(c *gin.Context) {
	productIDStr := c.Param("productId")
	userIDStr := c.Param("userId")

	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	purchased, err := h.cacheService.IsUserPurchased(c.Request.Context(), productID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"product_id": productID,
		"user_id":    userID,
		"purchased":  purchased,
	})
}

func (h *Handler) GetUserPurchaseInfo(c *gin.Context) {
	productIDStr := c.Param("productId")
	userIDStr := c.Param("userId")

	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	info, err := h.cacheService.GetUserPurchaseInfo(c.Request.Context(), productID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
}

func (h *Handler) GetSeckillActivity(c *gin.Context) {
	productIDStr := c.Param("productId")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	activity, err := h.cacheService.GetSeckillActivity(c.Request.Context(), productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, activity)
}

func (h *Handler) GetPurchaseUserCount(c *gin.Context) {
	productIDStr := c.Param("productId")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	count, err := h.cacheService.GetPurchaseUserCount(c.Request.Context(), productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"product_id": productID,
		"count":      count,
	})
}

func (h *Handler) CleanupSeckillData(c *gin.Context) {
	productIDStr := c.Param("productId")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
		return
	}

	err = h.cacheService.CleanupSeckillData(c.Request.Context(), productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cleanup success"})
}

// 健康检查
func (h *Handler) HealthCheck(c *gin.Context) {
	err := h.cacheService.HealthCheck(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}
