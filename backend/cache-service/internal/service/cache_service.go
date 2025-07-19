package service

import (
	"context"
	"fmt"
	"time"

	"cache-service/internal/lock"
	"cache-service/internal/redis"
	"cache-service/internal/seckill"
	"github.com/sirupsen/logrus"
)

type CacheService struct {
	redisClient  *redis.Client
	seckillCache *seckill.SeckillCache
}

type Config struct {
	Redis   *redis.Config
	Seckill *seckill.Config
}

func NewCacheService(config *Config, logger *logrus.Logger) (*CacheService, error) {
	redisClient, err := redis.NewClient(config.Redis, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create redis client: %w", err)
	}

	seckillCache := seckill.NewSeckillCache(redisClient, config.Seckill)

	return &CacheService{
		redisClient:  redisClient,
		seckillCache: seckillCache,
	}, nil
}


func (cs *CacheService) Close() error {
	return cs.redisClient.Close()
}

// 基础缓存操作
func (cs *CacheService) Get(ctx context.Context, key string) (string, error) {
	return cs.redisClient.Get(ctx, key)
}

func (cs *CacheService) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return cs.redisClient.Set(ctx, key, value, expiration)
}

func (cs *CacheService) Del(ctx context.Context, keys ...string) error {
	return cs.redisClient.Del(ctx, keys...)
}

func (cs *CacheService) Exists(ctx context.Context, keys ...string) (int64, error) {
	return cs.redisClient.Exists(ctx, keys...)
}

func (cs *CacheService) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return cs.redisClient.Expire(ctx, key, expiration)
}

func (cs *CacheService) TTL(ctx context.Context, key string) (time.Duration, error) {
	return cs.redisClient.TTL(ctx, key)
}

// 数值操作
func (cs *CacheService) Incr(ctx context.Context, key string) (int64, error) {
	return cs.redisClient.Incr(ctx, key)
}

func (cs *CacheService) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return cs.redisClient.IncrBy(ctx, key, value)
}

func (cs *CacheService) Decr(ctx context.Context, key string) (int64, error) {
	return cs.redisClient.Decr(ctx, key)
}

func (cs *CacheService) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return cs.redisClient.DecrBy(ctx, key, value)
}

// 列表操作
func (cs *CacheService) LPush(ctx context.Context, key string, values ...interface{}) error {
	return cs.redisClient.LPush(ctx, key, values...)
}

func (cs *CacheService) RPush(ctx context.Context, key string, values ...interface{}) error {
	return cs.redisClient.RPush(ctx, key, values...)
}

func (cs *CacheService) LPop(ctx context.Context, key string) (string, error) {
	return cs.redisClient.LPop(ctx, key)
}

func (cs *CacheService) RPop(ctx context.Context, key string) (string, error) {
	return cs.redisClient.RPop(ctx, key)
}

func (cs *CacheService) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return cs.redisClient.LRange(ctx, key, start, stop)
}

func (cs *CacheService) LLen(ctx context.Context, key string) (int64, error) {
	return cs.redisClient.LLen(ctx, key)
}

// 集合操作
func (cs *CacheService) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return cs.redisClient.SAdd(ctx, key, members...)
}

func (cs *CacheService) SRem(ctx context.Context, key string, members ...interface{}) error {
	return cs.redisClient.SRem(ctx, key, members...)
}

func (cs *CacheService) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return cs.redisClient.SIsMember(ctx, key, member)
}

func (cs *CacheService) SMembers(ctx context.Context, key string) ([]string, error) {
	return cs.redisClient.SMembers(ctx, key)
}

func (cs *CacheService) SCard(ctx context.Context, key string) (int64, error) {
	return cs.redisClient.SCard(ctx, key)
}

// 哈希操作
func (cs *CacheService) HSet(ctx context.Context, key string, values ...interface{}) error {
	return cs.redisClient.HSet(ctx, key, values...)
}

func (cs *CacheService) HGet(ctx context.Context, key, field string) (string, error) {
	return cs.redisClient.HGet(ctx, key, field)
}

func (cs *CacheService) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return cs.redisClient.HGetAll(ctx, key)
}

func (cs *CacheService) HDel(ctx context.Context, key string, fields ...string) error {
	return cs.redisClient.HDel(ctx, key, fields...)
}

func (cs *CacheService) HExists(ctx context.Context, key, field string) (bool, error) {
	return cs.redisClient.HExists(ctx, key, field)
}

// 分布式锁操作
func (cs *CacheService) CreateLock(key string, ttl time.Duration) *lock.DistributedLock {
	return lock.NewDistributedLock(cs.redisClient, key, ttl)
}

func (cs *CacheService) AcquireLock(ctx context.Context, key string, ttl time.Duration) (*lock.DistributedLock, error) {
	distributedLock := cs.CreateLock(key, ttl)
	if err := distributedLock.Lock(ctx); err != nil {
		return nil, err
	}
	return distributedLock, nil
}

func (cs *CacheService) TryAcquireLock(ctx context.Context, key string, ttl time.Duration, retryInterval time.Duration, maxRetries int) (*lock.DistributedLock, error) {
	distributedLock := cs.CreateLock(key, ttl)
	if err := distributedLock.TryLock(ctx, retryInterval, maxRetries); err != nil {
		return nil, err
	}
	return distributedLock, nil
}

// 秒杀相关操作
func (cs *CacheService) PreloadSeckillActivity(ctx context.Context, activity *seckill.SeckillActivity) error {
	return cs.seckillCache.PreloadSeckillActivity(ctx, activity)
}

func (cs *CacheService) GetSeckillStock(ctx context.Context, productID int64) (int64, error) {
	return cs.seckillCache.GetStock(ctx, productID)
}

func (cs *CacheService) SeckillPurchase(ctx context.Context, productID int64, userID int64, quantity int64) error {
	return cs.seckillCache.SeckillPurchase(ctx, productID, userID, quantity)
}

func (cs *CacheService) IsUserPurchased(ctx context.Context, productID int64, userID int64) (bool, error) {
	return cs.seckillCache.IsUserPurchased(ctx, productID, userID)
}

func (cs *CacheService) GetUserPurchaseInfo(ctx context.Context, productID int64, userID int64) (*seckill.UserPurchaseInfo, error) {
	return cs.seckillCache.GetUserPurchaseInfo(ctx, productID, userID)
}

func (cs *CacheService) GetSeckillActivity(ctx context.Context, productID int64) (*seckill.SeckillActivity, error) {
	return cs.seckillCache.GetSeckillActivity(ctx, productID)
}

func (cs *CacheService) GetPurchaseUserCount(ctx context.Context, productID int64) (int64, error) {
	return cs.seckillCache.GetPurchaseUserCount(ctx, productID)
}

func (cs *CacheService) CleanupSeckillData(ctx context.Context, productID int64) error {
	return cs.seckillCache.CleanupSeckillData(ctx, productID)
}

// 批量操作
func (cs *CacheService) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return cs.redisClient.MGet(ctx, keys...)
}

func (cs *CacheService) MSet(ctx context.Context, values ...interface{}) error {
	return cs.redisClient.MSet(ctx, values...)
}

// 健康检查
func (cs *CacheService) HealthCheck(ctx context.Context) error {
	// 尝试执行一个简单的操作来检查 Redis 连接
	_, err := cs.redisClient.Get(ctx, "health_check")
	return err
}
