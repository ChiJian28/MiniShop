package flowcontrol

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// 限流器接口
type Limiter interface {
	Allow() bool
	AllowN(n int) bool
	Wait(ctx context.Context) error
	WaitN(ctx context.Context, n int) error
}

// 令牌桶限流器
type TokenBucketLimiter struct {
	rate       float64   // 令牌生成速率（每秒）
	capacity   int       // 桶容量
	tokens     int       // 当前令牌数
	lastRefill time.Time // 上次补充令牌时间
	mu         sync.Mutex
	logger     *logrus.Logger
}

// 创建令牌桶限流器
func NewTokenBucketLimiter(rate float64, capacity int, logger *logrus.Logger) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity,
		lastRefill: time.Now(),
		logger:     logger,
	}
}

// 检查是否允许请求
func (l *TokenBucketLimiter) Allow() bool {
	return l.AllowN(1)
}

// 检查是否允许 n 个请求
func (l *TokenBucketLimiter) AllowN(n int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refill()

	if l.tokens >= n {
		l.tokens -= n
		return true
	}

	return false
}

// 等待直到可以处理请求
func (l *TokenBucketLimiter) Wait(ctx context.Context) error {
	return l.WaitN(ctx, 1)
}

// 等待直到可以处理 n 个请求
func (l *TokenBucketLimiter) WaitN(ctx context.Context, n int) error {
	if n > l.capacity {
		return fmt.Errorf("requested tokens %d exceeds capacity %d", n, l.capacity)
	}

	for {
		if l.AllowN(n) {
			return nil
		}

		// 计算等待时间
		waitTime := time.Duration(float64(n-l.tokens)/l.rate) * time.Second

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			continue
		}
	}
}

// 补充令牌
func (l *TokenBucketLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(l.lastRefill).Seconds()

	if elapsed > 0 {
		tokensToAdd := int(elapsed * l.rate)
		if tokensToAdd > 0 {
			l.tokens += tokensToAdd
			if l.tokens > l.capacity {
				l.tokens = l.capacity
			}
			l.lastRefill = now
		}
	}
}

// 获取当前令牌数
func (l *TokenBucketLimiter) GetTokens() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refill()
	return l.tokens
}

// 滑动窗口限流器
type SlidingWindowLimiter struct {
	limit    int           // 限制数量
	window   time.Duration // 窗口大小
	requests []time.Time   // 请求时间记录
	mu       sync.Mutex
	logger   *logrus.Logger
}

// 创建滑动窗口限流器
func NewSlidingWindowLimiter(limit int, window time.Duration, logger *logrus.Logger) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		limit:    limit,
		window:   window,
		requests: make([]time.Time, 0),
		logger:   logger,
	}
}

// 检查是否允许请求
func (l *SlidingWindowLimiter) Allow() bool {
	return l.AllowN(1)
}

// 检查是否允许 n 个请求
func (l *SlidingWindowLimiter) AllowN(n int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	// 移除过期的请求记录
	var validRequests []time.Time
	for _, req := range l.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	l.requests = validRequests

	// 检查是否超过限制
	if len(l.requests)+n > l.limit {
		return false
	}

	// 添加新的请求记录
	for i := 0; i < n; i++ {
		l.requests = append(l.requests, now)
	}

	return true
}

// 等待直到可以处理请求
func (l *SlidingWindowLimiter) Wait(ctx context.Context) error {
	return l.WaitN(ctx, 1)
}

// 等待直到可以处理 n 个请求
func (l *SlidingWindowLimiter) WaitN(ctx context.Context, n int) error {
	for {
		if l.AllowN(n) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}
}

// 获取当前请求数
func (l *SlidingWindowLimiter) GetCurrentRequests() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	count := 0
	for _, req := range l.requests {
		if req.After(cutoff) {
			count++
		}
	}

	return count
}

// 固定窗口限流器
type FixedWindowLimiter struct {
	limit       int           // 限制数量
	window      time.Duration // 窗口大小
	count       int           // 当前计数
	windowStart time.Time     // 窗口开始时间
	mu          sync.Mutex
	logger      *logrus.Logger
}

// 创建固定窗口限流器
func NewFixedWindowLimiter(limit int, window time.Duration, logger *logrus.Logger) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		limit:       limit,
		window:      window,
		count:       0,
		windowStart: time.Now(),
		logger:      logger,
	}
}

// 检查是否允许请求
func (l *FixedWindowLimiter) Allow() bool {
	return l.AllowN(1)
}

// 检查是否允许 n 个请求
func (l *FixedWindowLimiter) AllowN(n int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// 检查是否需要重置窗口
	if now.Sub(l.windowStart) >= l.window {
		l.count = 0
		l.windowStart = now
	}

	// 检查是否超过限制
	if l.count+n > l.limit {
		return false
	}

	l.count += n
	return true
}

// 等待直到可以处理请求
func (l *FixedWindowLimiter) Wait(ctx context.Context) error {
	return l.WaitN(ctx, 1)
}

// 等待直到可以处理 n 个请求
func (l *FixedWindowLimiter) WaitN(ctx context.Context, n int) error {
	for {
		if l.AllowN(n) {
			return nil
		}

		// 计算到下一个窗口的等待时间
		l.mu.Lock()
		waitTime := l.window - time.Since(l.windowStart)
		l.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			continue
		}
	}
}

// 获取当前计数
func (l *FixedWindowLimiter) GetCurrentCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if now.Sub(l.windowStart) >= l.window {
		return 0
	}

	return l.count
}

// 分布式限流器（基于 Redis）
type DistributedLimiter struct {
	key         string
	limit       int
	window      time.Duration
	redisClient interface{} // Redis 客户端接口
	logger      *logrus.Logger
}

// 创建分布式限流器
func NewDistributedLimiter(key string, limit int, window time.Duration, redisClient interface{}, logger *logrus.Logger) *DistributedLimiter {
	return &DistributedLimiter{
		key:         key,
		limit:       limit,
		window:      window,
		redisClient: redisClient,
		logger:      logger,
	}
}

// 检查是否允许请求
func (l *DistributedLimiter) Allow() bool {
	return l.AllowN(1)
}

// 检查是否允许 n 个请求
func (l *DistributedLimiter) AllowN(n int) bool {
	// 这里需要实现基于 Redis 的分布式限流逻辑
	// 使用 Redis 的 INCR 和 EXPIRE 命令
	// 或者使用 Lua 脚本实现更复杂的逻辑

	// 简化实现，实际应该使用 Redis
	return true
}

// 等待直到可以处理请求
func (l *DistributedLimiter) Wait(ctx context.Context) error {
	return l.WaitN(ctx, 1)
}

// 等待直到可以处理 n 个请求
func (l *DistributedLimiter) WaitN(ctx context.Context, n int) error {
	for {
		if l.AllowN(n) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}
}

// 多级限流器
type MultiLevelLimiter struct {
	limiters []Limiter
	logger   *logrus.Logger
}

// 创建多级限流器
func NewMultiLevelLimiter(limiters []Limiter, logger *logrus.Logger) *MultiLevelLimiter {
	return &MultiLevelLimiter{
		limiters: limiters,
		logger:   logger,
	}
}

// 检查是否允许请求
func (l *MultiLevelLimiter) Allow() bool {
	return l.AllowN(1)
}

// 检查是否允许 n 个请求
func (l *MultiLevelLimiter) AllowN(n int) bool {
	for _, limiter := range l.limiters {
		if !limiter.AllowN(n) {
			return false
		}
	}
	return true
}

// 等待直到可以处理请求
func (l *MultiLevelLimiter) Wait(ctx context.Context) error {
	return l.WaitN(ctx, 1)
}

// 等待直到可以处理 n 个请求
func (l *MultiLevelLimiter) WaitN(ctx context.Context, n int) error {
	for _, limiter := range l.limiters {
		if err := limiter.WaitN(ctx, n); err != nil {
			return err
		}
	}
	return nil
}
