package seckill

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"cache-service/internal/lock"
)

type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
	IncrBy(ctx context.Context, key string, value int64) (int64, error)
	DecrBy(ctx context.Context, key string, value int64) (int64, error)
	SAdd(ctx context.Context, key string, members ...interface{}) error
	SIsMember(ctx context.Context, key string, member interface{}) (bool, error)
	SRem(ctx context.Context, key string, members ...interface{}) error
	SCard(ctx context.Context, key string) (int64, error)
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) interface{}
	Expire(ctx context.Context, key string, expiration time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)
	HSet(ctx context.Context, key string, values ...interface{}) error
	HGet(ctx context.Context, key, field string) (string, error)
	HGetAll(ctx context.Context, key string) (map[string]string, error)
}

type Config struct {
	StockKeyPrefix string
	UserKeyPrefix  string
	LockKeyPrefix  string
	DefaultTTL     time.Duration
}

type SeckillCache struct {
	client RedisClient
	config *Config
}

type SeckillActivity struct {
	ProductID   int64     `json:"product_id"`
	ProductName string    `json:"product_name"`
	Price       float64   `json:"price"`
	Stock       int64     `json:"stock"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Status      string    `json:"status"` // active, inactive, ended
}

type UserPurchaseInfo struct {
	UserID       int64     `json:"user_id"`
	ProductID    int64     `json:"product_id"`
	Quantity     int64     `json:"quantity"`
	PurchaseTime time.Time `json:"purchase_time"`
	Status       string    `json:"status"` // pending, success, failed
}

func NewSeckillCache(client RedisClient, config *Config) *SeckillCache {
	return &SeckillCache{
		client: client,
		config: config,
	}
}

// 预加载秒杀活动数据
func (sc *SeckillCache) PreloadSeckillActivity(ctx context.Context, activity *SeckillActivity) error {
	// 设置库存
	stockKey := sc.getStockKey(activity.ProductID)
	if err := sc.client.Set(ctx, stockKey, activity.Stock, sc.config.DefaultTTL); err != nil {
		return fmt.Errorf("failed to set stock: %w", err)
	}

	// 设置活动信息
	activityKey := sc.getActivityKey(activity.ProductID)
	activityData, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("failed to marshal activity: %w", err)
	}

	if err := sc.client.Set(ctx, activityKey, string(activityData), sc.config.DefaultTTL); err != nil {
		return fmt.Errorf("failed to set activity: %w", err)
	}

	return nil
}

// 获取库存
func (sc *SeckillCache) GetStock(ctx context.Context, productID int64) (int64, error) {
	stockKey := sc.getStockKey(productID)
	stockStr, err := sc.client.Get(ctx, stockKey)
	if err != nil {
		return 0, fmt.Errorf("failed to get stock: %w", err)
	}

	if stockStr == "" {
		return 0, fmt.Errorf("stock not found for product %d", productID)
	}

	stock, err := strconv.ParseInt(stockStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse stock: %w", err)
	}

	return stock, nil
}

// 扣减库存的 Lua 脚本，确保原子性
const decrStockScript = `
local stock_key = KEYS[1]
local quantity = tonumber(ARGV[1])

local current_stock = redis.call('GET', stock_key)
if not current_stock then
    return -1  -- 库存不存在
end

current_stock = tonumber(current_stock)
if current_stock < quantity then
    return -2  -- 库存不足
end

local new_stock = current_stock - quantity
redis.call('SET', stock_key, new_stock)
return new_stock
`

// 扣减库存
func (sc *SeckillCache) DecrStock(ctx context.Context, productID int64, quantity int64) (int64, error) {
	stockKey := sc.getStockKey(productID)
	result := sc.client.Eval(ctx, decrStockScript, []string{stockKey}, quantity)

	if result == nil {
		return 0, fmt.Errorf("script execution failed")
	}

	switch v := result.(type) {
	case int64:
		if v == -1 {
			return 0, fmt.Errorf("stock not found for product %d", productID)
		}
		if v == -2 {
			return 0, fmt.Errorf("insufficient stock for product %d", productID)
		}
		return v, nil
	default:
		return 0, fmt.Errorf("unexpected result type: %T", result)
	}
}

// 增加库存（用于回滚）
func (sc *SeckillCache) IncrStock(ctx context.Context, productID int64, quantity int64) (int64, error) {
	stockKey := sc.getStockKey(productID)
	return sc.client.IncrBy(ctx, stockKey, quantity)
}

// 检查用户是否已经购买
func (sc *SeckillCache) IsUserPurchased(ctx context.Context, productID int64, userID int64) (bool, error) {
	userKey := sc.getUserKey(productID)
	return sc.client.SIsMember(ctx, userKey, userID)
}

// 添加用户购买记录
func (sc *SeckillCache) AddUserPurchase(ctx context.Context, productID int64, userID int64) error {
	userKey := sc.getUserKey(productID)
	return sc.client.SAdd(ctx, userKey, userID)
}

// 移除用户购买记录（用于回滚）
func (sc *SeckillCache) RemoveUserPurchase(ctx context.Context, productID int64, userID int64) error {
	userKey := sc.getUserKey(productID)
	return sc.client.SRem(ctx, userKey, userID)
}

// 获取购买用户数量
func (sc *SeckillCache) GetPurchaseUserCount(ctx context.Context, productID int64) (int64, error) {
	userKey := sc.getUserKey(productID)
	return sc.client.SCard(ctx, userKey)
}

// 创建分布式锁
func (sc *SeckillCache) CreateLock(productID int64, ttl time.Duration) *lock.DistributedLock {
	lockKey := sc.getLockKey(productID)
	return lock.NewDistributedLock(sc.client, lockKey, ttl)
}

// 秒杀购买（带锁）
func (sc *SeckillCache) SeckillPurchase(ctx context.Context, productID int64, userID int64, quantity int64) error {
	// 创建分布式锁
	lockTTL := 10 * time.Second
	distributedLock := sc.CreateLock(productID, lockTTL)

	// 尝试获取锁
	if err := distributedLock.TryLock(ctx, 100*time.Millisecond, 50); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer distributedLock.Unlock(ctx)

	// 检查用户是否已经购买
	purchased, err := sc.IsUserPurchased(ctx, productID, userID)
	if err != nil {
		return fmt.Errorf("failed to check user purchase status: %w", err)
	}
	if purchased {
		return fmt.Errorf("user %d has already purchased product %d", userID, productID)
	}

	// 扣减库存
	_, err = sc.DecrStock(ctx, productID, quantity)
	if err != nil {
		return fmt.Errorf("failed to decrement stock: %w", err)
	}

	// 添加用户购买记录
	if err := sc.AddUserPurchase(ctx, productID, userID); err != nil {
		// 回滚库存
		sc.IncrStock(ctx, productID, quantity)
		return fmt.Errorf("failed to add user purchase: %w", err)
	}

	// 记录购买信息
	purchaseInfo := &UserPurchaseInfo{
		UserID:       userID,
		ProductID:    productID,
		Quantity:     quantity,
		PurchaseTime: time.Now(),
		Status:       "pending",
	}

	if err := sc.SetUserPurchaseInfo(ctx, productID, userID, purchaseInfo); err != nil {
		// 这里可能需要记录日志，但不回滚，因为主要操作已经成功
		return fmt.Errorf("failed to set purchase info: %w", err)
	}

	return nil
}

// 设置用户购买信息
func (sc *SeckillCache) SetUserPurchaseInfo(ctx context.Context, productID int64, userID int64, info *UserPurchaseInfo) error {
	key := sc.getUserPurchaseInfoKey(productID, userID)
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal purchase info: %w", err)
	}

	return sc.client.Set(ctx, key, string(data), sc.config.DefaultTTL)
}

// 获取用户购买信息
func (sc *SeckillCache) GetUserPurchaseInfo(ctx context.Context, productID int64, userID int64) (*UserPurchaseInfo, error) {
	key := sc.getUserPurchaseInfoKey(productID, userID)
	data, err := sc.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get purchase info: %w", err)
	}

	if data == "" {
		return nil, fmt.Errorf("purchase info not found")
	}

	var info UserPurchaseInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal purchase info: %w", err)
	}

	return &info, nil
}

// 获取秒杀活动信息
func (sc *SeckillCache) GetSeckillActivity(ctx context.Context, productID int64) (*SeckillActivity, error) {
	key := sc.getActivityKey(productID)
	data, err := sc.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity: %w", err)
	}

	if data == "" {
		return nil, fmt.Errorf("activity not found")
	}

	var activity SeckillActivity
	if err := json.Unmarshal([]byte(data), &activity); err != nil {
		return nil, fmt.Errorf("failed to unmarshal activity: %w", err)
	}

	return &activity, nil
}

// 清理秒杀数据
func (sc *SeckillCache) CleanupSeckillData(ctx context.Context, productID int64) error {
	keys := []string{
		sc.getStockKey(productID),
		sc.getUserKey(productID),
		sc.getActivityKey(productID),
		sc.getLockKey(productID),
	}

	return sc.client.Del(ctx, keys...)
}

// 生成各种 key
func (sc *SeckillCache) getStockKey(productID int64) string {
	return fmt.Sprintf("%s%d", sc.config.StockKeyPrefix, productID)
}

func (sc *SeckillCache) getUserKey(productID int64) string {
	return fmt.Sprintf("%s%d", sc.config.UserKeyPrefix, productID)
}

func (sc *SeckillCache) getLockKey(productID int64) string {
	return fmt.Sprintf("%s%d", sc.config.LockKeyPrefix, productID)
}

func (sc *SeckillCache) getActivityKey(productID int64) string {
	return fmt.Sprintf("seckill:activity:%d", productID)
}

func (sc *SeckillCache) getUserPurchaseInfoKey(productID int64, userID int64) string {
	return fmt.Sprintf("seckill:purchase:%d:%d", productID, userID)
}
