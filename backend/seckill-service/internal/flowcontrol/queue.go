package flowcontrol

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	ErrQueueFull   = errors.New("queue is full")
	ErrQueueClosed = errors.New("queue is closed")
	ErrTimeout     = errors.New("request timeout")
)

// 请求队列项
type QueueItem struct {
	ID        string
	Request   interface{}
	Response  chan interface{}
	Error     chan error
	Timestamp time.Time
	Timeout   time.Duration
}

// 请求队列
type RequestQueue struct {
	capacity   int
	queue      chan *QueueItem
	processing chan *QueueItem
	workers    int
	processor  func(ctx context.Context, item *QueueItem) (interface{}, error)
	mutex      sync.RWMutex
	closed     bool
	logger     *logrus.Logger

	// 统计信息
	stats QueueStats
}

// 队列统计信息
type QueueStats struct {
	TotalRequests     int64
	ProcessedRequests int64
	FailedRequests    int64
	TimeoutRequests   int64
	QueuedRequests    int64
	AverageWaitTime   time.Duration
	MaxWaitTime       time.Duration
	mutex             sync.RWMutex
}

// 创建请求队列
func NewRequestQueue(capacity, workers int, processor func(ctx context.Context, item *QueueItem) (interface{}, error), logger *logrus.Logger) *RequestQueue {
	return &RequestQueue{
		capacity:   capacity,
		queue:      make(chan *QueueItem, capacity),
		processing: make(chan *QueueItem, workers),
		workers:    workers,
		processor:  processor,
		logger:     logger,
	}
}

// 启动队列处理
func (q *RequestQueue) Start(ctx context.Context) {
	// 启动工作协程
	for i := 0; i < q.workers; i++ {
		go q.worker(ctx, i)
	}

	// 启动调度协程
	go q.scheduler(ctx)

	q.logger.Infof("Request queue started with %d workers, capacity: %d", q.workers, q.capacity)
}

// 调度器
func (q *RequestQueue) scheduler(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			q.close()
			return
		case item := <-q.queue:
			select {
			case q.processing <- item:
				// 成功发送到处理队列
			case <-ctx.Done():
				q.sendError(item, ErrQueueClosed)
				return
			}
		}
	}
}

// 工作协程
func (q *RequestQueue) worker(ctx context.Context, workerID int) {
	q.logger.Debugf("Worker %d started", workerID)

	for {
		select {
		case <-ctx.Done():
			q.logger.Debugf("Worker %d stopped", workerID)
			return
		case item := <-q.processing:
			q.processItem(ctx, item, workerID)
		}
	}
}

// 处理队列项
func (q *RequestQueue) processItem(ctx context.Context, item *QueueItem, workerID int) {
	startTime := time.Now()

	// 检查超时
	if item.Timeout > 0 && time.Since(item.Timestamp) > item.Timeout {
		q.sendError(item, ErrTimeout)
		q.updateStats(false, true, time.Since(item.Timestamp))
		return
	}

	// 创建带超时的上下文
	var cancel context.CancelFunc
	if item.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, item.Timeout)
		defer cancel()
	}

	// 处理请求
	result, err := q.processor(ctx, item)

	processingTime := time.Since(startTime)
	waitTime := startTime.Sub(item.Timestamp)

	if err != nil {
		q.sendError(item, err)
		q.updateStats(false, false, waitTime)
		q.logger.Debugf("Worker %d processed item %s with error: %v, wait time: %v, processing time: %v",
			workerID, item.ID, err, waitTime, processingTime)
	} else {
		q.sendResponse(item, result)
		q.updateStats(true, false, waitTime)
		q.logger.Debugf("Worker %d processed item %s successfully, wait time: %v, processing time: %v",
			workerID, item.ID, waitTime, processingTime)
	}
}

// 发送响应
func (q *RequestQueue) sendResponse(item *QueueItem, result interface{}) {
	select {
	case item.Response <- result:
	default:
		q.logger.Warnf("Failed to send response for item %s", item.ID)
	}
}

// 发送错误
func (q *RequestQueue) sendError(item *QueueItem, err error) {
	select {
	case item.Error <- err:
	default:
		q.logger.Warnf("Failed to send error for item %s: %v", item.ID, err)
	}
}

// 提交请求
func (q *RequestQueue) Submit(ctx context.Context, id string, request interface{}, timeout time.Duration) (interface{}, error) {
	q.mutex.RLock()
	if q.closed {
		q.mutex.RUnlock()
		return nil, ErrQueueClosed
	}
	q.mutex.RUnlock()

	item := &QueueItem{
		ID:        id,
		Request:   request,
		Response:  make(chan interface{}, 1),
		Error:     make(chan error, 1),
		Timestamp: time.Now(),
		Timeout:   timeout,
	}

	// 尝试加入队列
	select {
	case q.queue <- item:
		q.stats.mutex.Lock()
		q.stats.TotalRequests++
		q.stats.QueuedRequests++
		q.stats.mutex.Unlock()
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		q.stats.mutex.Lock()
		q.stats.TotalRequests++
		q.stats.FailedRequests++
		q.stats.mutex.Unlock()
		return nil, ErrQueueFull
	}

	// 等待响应
	select {
	case result := <-item.Response:
		return result, nil
	case err := <-item.Error:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// 异步提交请求
func (q *RequestQueue) SubmitAsync(ctx context.Context, id string, request interface{}, timeout time.Duration, callback func(interface{}, error)) error {
	q.mutex.RLock()
	if q.closed {
		q.mutex.RUnlock()
		return ErrQueueClosed
	}
	q.mutex.RUnlock()

	item := &QueueItem{
		ID:        id,
		Request:   request,
		Response:  make(chan interface{}, 1),
		Error:     make(chan error, 1),
		Timestamp: time.Now(),
		Timeout:   timeout,
	}

	// 尝试加入队列
	select {
	case q.queue <- item:
		q.stats.mutex.Lock()
		q.stats.TotalRequests++
		q.stats.QueuedRequests++
		q.stats.mutex.Unlock()
	case <-ctx.Done():
		return ctx.Err()
	default:
		q.stats.mutex.Lock()
		q.stats.TotalRequests++
		q.stats.FailedRequests++
		q.stats.mutex.Unlock()
		return ErrQueueFull
	}

	// 异步处理响应
	go func() {
		select {
		case result := <-item.Response:
			callback(result, nil)
		case err := <-item.Error:
			callback(nil, err)
		case <-ctx.Done():
			callback(nil, ctx.Err())
		}
	}()

	return nil
}

// 更新统计信息
func (q *RequestQueue) updateStats(success, timeout bool, waitTime time.Duration) {
	q.stats.mutex.Lock()
	defer q.stats.mutex.Unlock()

	q.stats.QueuedRequests--

	if success {
		q.stats.ProcessedRequests++
	} else if timeout {
		q.stats.TimeoutRequests++
	} else {
		q.stats.FailedRequests++
	}

	// 更新等待时间统计
	if waitTime > q.stats.MaxWaitTime {
		q.stats.MaxWaitTime = waitTime
	}

	// 简单的移动平均
	if q.stats.ProcessedRequests > 0 {
		q.stats.AverageWaitTime = time.Duration(
			(int64(q.stats.AverageWaitTime) + int64(waitTime)) / 2,
		)
	} else {
		q.stats.AverageWaitTime = waitTime
	}
}

// 获取统计信息
func (q *RequestQueue) GetStats() QueueStats {
	q.stats.mutex.RLock()
	defer q.stats.mutex.RUnlock()

	return q.stats
}

// 获取队列长度
func (q *RequestQueue) QueueLength() int {
	return len(q.queue)
}

// 获取处理队列长度
func (q *RequestQueue) ProcessingLength() int {
	return len(q.processing)
}

// 关闭队列
func (q *RequestQueue) close() {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.closed {
		return
	}

	q.closed = true
	close(q.queue)
	close(q.processing)

	q.logger.Info("Request queue closed")
}

// 优先级队列项
type PriorityQueueItem struct {
	*QueueItem
	Priority int
}

// 优先级队列
type PriorityRequestQueue struct {
	*RequestQueue
	priorityQueue []*PriorityQueueItem
	mutex         sync.Mutex
}

// 创建优先级队列
func NewPriorityRequestQueue(capacity, workers int, processor func(ctx context.Context, item *QueueItem) (interface{}, error), logger *logrus.Logger) *PriorityRequestQueue {
	return &PriorityRequestQueue{
		RequestQueue:  NewRequestQueue(capacity, workers, processor, logger),
		priorityQueue: make([]*PriorityQueueItem, 0),
	}
}

// 提交优先级请求
func (pq *PriorityRequestQueue) SubmitWithPriority(ctx context.Context, id string, request interface{}, timeout time.Duration, priority int) (interface{}, error) {
	pq.mutex.Lock()
	if pq.closed {
		pq.mutex.Unlock()
		return nil, ErrQueueClosed
	}

	item := &QueueItem{
		ID:        id,
		Request:   request,
		Response:  make(chan interface{}, 1),
		Error:     make(chan error, 1),
		Timestamp: time.Now(),
		Timeout:   timeout,
	}

	priorityItem := &PriorityQueueItem{
		QueueItem: item,
		Priority:  priority,
	}

	// 按优先级插入
	pq.insertByPriority(priorityItem)
	pq.mutex.Unlock()

	pq.stats.mutex.Lock()
	pq.stats.TotalRequests++
	pq.stats.QueuedRequests++
	pq.stats.mutex.Unlock()

	// 等待响应
	select {
	case result := <-item.Response:
		return result, nil
	case err := <-item.Error:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// 按优先级插入
func (pq *PriorityRequestQueue) insertByPriority(item *PriorityQueueItem) {
	// 简单的插入排序，优先级高的在前面
	inserted := false
	for i, existing := range pq.priorityQueue {
		if item.Priority > existing.Priority {
			pq.priorityQueue = append(pq.priorityQueue[:i], append([]*PriorityQueueItem{item}, pq.priorityQueue[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		pq.priorityQueue = append(pq.priorityQueue, item)
	}
}

// 获取下一个优先级项
func (pq *PriorityRequestQueue) getNextPriorityItem() *QueueItem {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	if len(pq.priorityQueue) == 0 {
		return nil
	}

	item := pq.priorityQueue[0]
	pq.priorityQueue = pq.priorityQueue[1:]
	return item.QueueItem
}
