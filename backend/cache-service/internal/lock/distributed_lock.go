package lock

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrLockFailed  = errors.New("failed to acquire lock")
	ErrLockNotHeld = errors.New("lock not held")
	ErrLockExpired = errors.New("lock expired")
)

type RedisClient interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) interface{}
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, keys ...string) error
}

type DistributedLock struct {
	client RedisClient
	key    string
	value  string
	ttl    time.Duration
}

func NewDistributedLock(client RedisClient, key string, ttl time.Duration) *DistributedLock {
	return &DistributedLock{
		client: client,
		key:    key,
		value:  fmt.Sprintf("%d", time.Now().UnixNano()),
		ttl:    ttl,
	}
}

// 获取锁
func (dl *DistributedLock) Lock(ctx context.Context) error {
	acquired, err := dl.client.SetNX(ctx, dl.key, dl.value, dl.ttl)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !acquired {
		return ErrLockFailed
	}

	return nil
}

// 尝试获取锁（带重试）
func (dl *DistributedLock) TryLock(ctx context.Context, retryInterval time.Duration, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		if err := dl.Lock(ctx); err == nil {
			return nil
		}

		if i < maxRetries-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryInterval):
				continue
			}
		}
	}

	return ErrLockFailed
}

// 释放锁的 Lua 脚本
const unlockScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end
`

// 释放锁
func (dl *DistributedLock) Unlock(ctx context.Context) error {
	result := dl.client.Eval(ctx, unlockScript, []string{dl.key}, dl.value)

	if result == nil {
		return ErrLockNotHeld
	}

	switch v := result.(type) {
	case int64:
		if v == 1 {
			return nil
		}
		return ErrLockNotHeld
	default:
		return ErrLockNotHeld
	}
}

// 续期锁的 Lua 脚本
const renewScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("expire", KEYS[1], ARGV[2])
else
    return 0
end
`

// 续期锁
func (dl *DistributedLock) Renew(ctx context.Context) error {
	result := dl.client.Eval(ctx, renewScript, []string{dl.key}, dl.value, int64(dl.ttl.Seconds()))

	if result == nil {
		return ErrLockNotHeld
	}

	switch v := result.(type) {
	case int64:
		if v == 1 {
			return nil
		}
		return ErrLockNotHeld
	default:
		return ErrLockNotHeld
	}
}

// 检查锁是否还持有
func (dl *DistributedLock) IsHeld(ctx context.Context) (bool, error) {
	value, err := dl.client.Get(ctx, dl.key)
	if err != nil {
		return false, err
	}

	return value == dl.value, nil
}

// 带自动续期的锁
func (dl *DistributedLock) LockWithAutoRenew(ctx context.Context, renewInterval time.Duration) (func(), error) {
	if err := dl.Lock(ctx); err != nil {
		return nil, err
	}

	renewCtx, cancel := context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(renewInterval)
		defer ticker.Stop()

		for {
			select {
			case <-renewCtx.Done():
				return
			case <-ticker.C:
				if err := dl.Renew(renewCtx); err != nil {
					// 续期失败，锁可能已过期
					return
				}
			}
		}
	}()

	// 返回释放锁的函数
	return func() {
		cancel()
		_ = dl.Unlock(context.Background())
	}, nil
}
