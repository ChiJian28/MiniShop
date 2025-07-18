package flowcontrol

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// 熔断器状态
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// 熔断器错误
var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
	ErrTooManyRequests    = errors.New("too many requests")
)

// 熔断器配置
type CircuitBreakerConfig struct {
	MaxRequests   uint32                                                              // 半开状态下的最大请求数
	Interval      time.Duration                                                       // 统计时间窗口
	Timeout       time.Duration                                                       // 熔断超时时间
	ReadyToTrip   func(counts Counts) bool                                            // 判断是否应该熔断
	OnStateChange func(name string, from CircuitBreakerState, to CircuitBreakerState) // 状态变化回调
	IsSuccessful  func(err error) bool                                                // 判断请求是否成功
}

// 统计信息
type Counts struct {
	Requests             uint32 // 总请求数
	TotalSuccesses       uint32 // 总成功数
	TotalFailures        uint32 // 总失败数
	ConsecutiveSuccesses uint32 // 连续成功数
	ConsecutiveFailures  uint32 // 连续失败数
}

// 请求是否成功
func (c Counts) IsSuccessful() bool {
	return c.TotalFailures == 0 || c.Requests < 3
}

// 熔断器
type CircuitBreaker struct {
	name          string
	maxRequests   uint32
	interval      time.Duration
	timeout       time.Duration
	readyToTrip   func(counts Counts) bool
	isSuccessful  func(err error) bool
	onStateChange func(name string, from CircuitBreakerState, to CircuitBreakerState)

	mutex      sync.Mutex
	state      CircuitBreakerState
	generation uint64
	counts     Counts
	expiry     time.Time

	logger *logrus.Logger
}

// 创建熔断器
func NewCircuitBreaker(name string, config CircuitBreakerConfig, logger *logrus.Logger) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:          name,
		maxRequests:   config.MaxRequests,
		interval:      config.Interval,
		timeout:       config.Timeout,
		readyToTrip:   config.ReadyToTrip,
		isSuccessful:  config.IsSuccessful,
		onStateChange: config.OnStateChange,
		logger:        logger,
	}

	if cb.maxRequests == 0 {
		cb.maxRequests = 1
	}

	if cb.interval <= 0 {
		cb.interval = 60 * time.Second
	}

	if cb.timeout <= 0 {
		cb.timeout = 60 * time.Second
	}

	if cb.readyToTrip == nil {
		cb.readyToTrip = defaultReadyToTrip
	}

	if cb.isSuccessful == nil {
		cb.isSuccessful = defaultIsSuccessful
	}

	cb.toNewGeneration(time.Now())

	return cb
}

// 默认熔断判断逻辑
func defaultReadyToTrip(counts Counts) bool {
	return counts.Requests >= 3 && counts.TotalFailures > 0 &&
		float64(counts.TotalFailures)/float64(counts.Requests) >= 0.6
}

// 默认成功判断逻辑
func defaultIsSuccessful(err error) bool {
	return err == nil
}

// 执行函数
func (cb *CircuitBreaker) Execute(req func() (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := req()
	cb.afterRequest(generation, cb.isSuccessful(err))
	return result, err
}

// 执行函数（带上下文）
func (cb *CircuitBreaker) ExecuteWithContext(ctx context.Context, req func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := req(ctx)
	cb.afterRequest(generation, cb.isSuccessful(err))
	return result, err
}

// 请求前检查
func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, ErrCircuitBreakerOpen
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		return generation, ErrTooManyRequests
	}

	cb.counts.onRequest()
	return generation, nil
}

// 请求后处理
func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

// 成功处理
func (cb *CircuitBreaker) onSuccess(state CircuitBreakerState, now time.Time) {
	cb.counts.onSuccess()

	if state == StateHalfOpen {
		cb.setState(StateClosed, now)
	}
}

// 失败处理
func (cb *CircuitBreaker) onFailure(state CircuitBreakerState, now time.Time) {
	cb.counts.onFailure()

	if cb.readyToTrip(cb.counts) {
		cb.setState(StateOpen, now)
	}
}

// 获取当前状态
func (cb *CircuitBreaker) currentState(now time.Time) (CircuitBreakerState, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

// 设置状态
func (cb *CircuitBreaker) setState(state CircuitBreakerState, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}

	cb.logger.Infof("Circuit breaker %s state changed from %s to %s", cb.name, prev, state)
}

// 新的统计周期
func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts.clear()

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}

// 获取状态
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// 获取统计信息
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	return cb.counts
}

// 统计信息方法
func (c *Counts) onRequest() {
	c.Requests++
}

func (c *Counts) onSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

func (c *Counts) onFailure() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

func (c *Counts) clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveSuccesses = 0
	c.ConsecutiveFailures = 0
}

// 熔断器管理器
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
	logger   *logrus.Logger
}

// 创建熔断器管理器
func NewCircuitBreakerManager(logger *logrus.Logger) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
		logger:   logger,
	}
}

// 获取熔断器
func (m *CircuitBreakerManager) GetCircuitBreaker(name string, config CircuitBreakerConfig) *CircuitBreaker {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if cb, exists := m.breakers[name]; exists {
		return cb
	}

	cb := NewCircuitBreaker(name, config, m.logger)
	m.breakers[name] = cb
	return cb
}

// 移除熔断器
func (m *CircuitBreakerManager) RemoveCircuitBreaker(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.breakers, name)
}

// 获取所有熔断器状态
func (m *CircuitBreakerManager) GetAllStates() map[string]CircuitBreakerState {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	states := make(map[string]CircuitBreakerState)
	for name, cb := range m.breakers {
		states[name] = cb.State()
	}
	return states
}

// 重置熔断器
func (m *CircuitBreakerManager) ResetCircuitBreaker(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if cb, exists := m.breakers[name]; exists {
		cb.mutex.Lock()
		cb.toNewGeneration(time.Now())
		cb.state = StateClosed
		cb.mutex.Unlock()
	}
}
