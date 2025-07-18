package seckill

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// 秒杀结果码
const (
	ResultSuccess            = 1
	ResultStockNotFound      = -1
	ResultInsufficientStock  = -2
	ResultUserAlreadyBought  = -3
	ResultActivityNotFound   = -4
	ResultActivityNotStarted = -5
	ResultActivityEnded      = -6
	ResultInvalidQuantity    = -7
	ResultSystemError        = -8
	ResultSystemBusy         = -9
	ResultRequestTimeout     = -10
)

// 秒杀请求
type SeckillRequest struct {
	ProductID int64 `json:"product_id"`
	UserID    int64 `json:"user_id"`
	Quantity  int64 `json:"quantity"`
}

// 秒杀结果
type SeckillResult struct {
	Code           int    `json:"code"`
	Message        string `json:"message"`
	Success        bool   `json:"success"`
	RemainingStock int64  `json:"remaining_stock"`
	OrderID        string `json:"order_id,omitempty"`
}

// 秒杀活动信息
type SeckillActivity struct {
	ProductID   int64     `json:"product_id"`
	ProductName string    `json:"product_name"`
	Price       float64   `json:"price"`
	Stock       int64     `json:"stock"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Status      string    `json:"status"`
}

// 秒杀统计信息
type SeckillStats struct {
	ProductID    int64           `json:"product_id"`
	CurrentStock int64           `json:"current_stock"`
	UserCount    int64           `json:"user_count"`
	ActivityInfo SeckillActivity `json:"activity_info"`
}

// 用户购买信息
type UserPurchaseInfo struct {
	UserID       int64     `json:"user_id"`
	ProductID    int64     `json:"product_id"`
	Quantity     int64     `json:"quantity"`
	PurchaseTime time.Time `json:"purchase_time"`
	Status       string    `json:"status"`
}

// Redis 客户端接口
type RedisClient interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptLoad(ctx context.Context, script string) *redis.StringCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

// 秒杀核心服务
type SeckillCore struct {
	redisClient RedisClient
	logger      *logrus.Logger
	scriptSHA   map[string]string // 预加载的脚本 SHA
}

// 创建秒杀核心服务
func NewSeckillCore(redisClient RedisClient, logger *logrus.Logger) *SeckillCore {
	return &SeckillCore{
		redisClient: redisClient,
		logger:      logger,
		scriptSHA:   make(map[string]string),
	}
}

// 初始化脚本
func (sc *SeckillCore) InitScripts(ctx context.Context) error {
	scripts := map[string]string{
		"seckill":     SeckillSimpleLuaScript,
		"rollback":    StockRollbackLuaScript,
		"batch_check": BatchCheckUserScript,
		"stats":       SeckillStatsScript,
	}

	for name, script := range scripts {
		result := sc.redisClient.ScriptLoad(ctx, script)
		if result.Err() != nil {
			return fmt.Errorf("failed to load script %s: %w", name, result.Err())
		}
		sc.scriptSHA[name] = result.Val()
		sc.logger.Infof("Loaded script %s with SHA: %s", name, sc.scriptSHA[name])
	}

	return nil
}

// 执行秒杀
func (sc *SeckillCore) ExecuteSeckill(ctx context.Context, req *SeckillRequest) (*SeckillResult, error) {
	// 参数验证
	if req.ProductID <= 0 || req.UserID <= 0 || req.Quantity <= 0 {
		return &SeckillResult{
			Code:    ResultInvalidQuantity,
			Message: "无效的参数",
			Success: false,
		}, nil
	}

	// 构建 Redis 键
	stockKey := fmt.Sprintf("seckill:stock:%d", req.ProductID)
	usersKey := fmt.Sprintf("seckill:users:%d", req.ProductID)

	// 执行 Lua 脚本
	keys := []string{stockKey, usersKey}
	args := []interface{}{req.UserID, req.Quantity}

	var result *redis.Cmd
	var err error

	// 优先使用预加载的脚本
	if sha, exists := sc.scriptSHA["seckill"]; exists {
		result = sc.redisClient.EvalSha(ctx, sha, keys, args...)
	} else {
		result = sc.redisClient.Eval(ctx, SeckillSimpleLuaScript, keys, args...)
	}

	if err = result.Err(); err != nil {
		sc.logger.Errorf("Failed to execute seckill script: %v", err)
		return &SeckillResult{
			Code:    ResultSystemError,
			Message: "系统错误",
			Success: false,
		}, err
	}

	// 解析结果
	return sc.parseSeckillResult(result.Val())
}

// 解析秒杀结果
func (sc *SeckillCore) parseSeckillResult(result interface{}) (*SeckillResult, error) {
	switch v := result.(type) {
	case []interface{}:
		if len(v) >= 2 {
			code, _ := v[0].(int64)
			remainingStock, _ := v[1].(int64)

			return &SeckillResult{
				Code:           int(code),
				Message:        sc.getResultMessage(int(code)),
				Success:        code == ResultSuccess,
				RemainingStock: remainingStock,
			}, nil
		}
	case int64:
		code := int(v)
		return &SeckillResult{
			Code:    code,
			Message: sc.getResultMessage(code),
			Success: code == ResultSuccess,
		}, nil
	}

	return &SeckillResult{
		Code:    ResultSystemError,
		Message: "解析结果失败",
		Success: false,
	}, nil
}

// 获取结果消息
func (sc *SeckillCore) getResultMessage(code int) string {
	messages := map[int]string{
		ResultSuccess:            "秒杀成功",
		ResultStockNotFound:      "商品不存在",
		ResultInsufficientStock:  "库存不足",
		ResultUserAlreadyBought:  "您已经购买过了",
		ResultActivityNotFound:   "活动不存在",
		ResultActivityNotStarted: "活动尚未开始",
		ResultActivityEnded:      "活动已结束",
		ResultInvalidQuantity:    "购买数量无效",
		ResultSystemError:        "系统错误",
		ResultSystemBusy:         "系统繁忙，请稍后重试",
		ResultRequestTimeout:     "请求超时",
	}

	if msg, exists := messages[code]; exists {
		return msg
	}
	return "未知错误"
}

// 库存回滚
func (sc *SeckillCore) RollbackStock(ctx context.Context, productID, userID, quantity int64) error {
	stockKey := fmt.Sprintf("seckill:stock:%d", productID)
	usersKey := fmt.Sprintf("seckill:users:%d", productID)

	keys := []string{stockKey, usersKey}
	args := []interface{}{userID, quantity}

	var result *redis.Cmd
	var err error

	if sha, exists := sc.scriptSHA["rollback"]; exists {
		result = sc.redisClient.EvalSha(ctx, sha, keys, args...)
	} else {
		result = sc.redisClient.Eval(ctx, StockRollbackLuaScript, keys, args...)
	}

	if err = result.Err(); err != nil {
		sc.logger.Errorf("Failed to rollback stock: %v", err)
		return err
	}

	newStock := result.Val()
	sc.logger.Infof("Rollback stock for product %d, user %d, quantity %d, new stock: %v",
		productID, userID, quantity, newStock)

	return nil
}

// 批量检查用户购买状态
func (sc *SeckillCore) BatchCheckUserStatus(ctx context.Context, productID int64, userIDs []int64) ([]bool, error) {
	usersKey := fmt.Sprintf("seckill:users:%d", productID)
	keys := []string{usersKey}

	args := make([]interface{}, len(userIDs))
	for i, userID := range userIDs {
		args[i] = userID
	}

	var result *redis.Cmd
	var err error

	if sha, exists := sc.scriptSHA["batch_check"]; exists {
		result = sc.redisClient.EvalSha(ctx, sha, keys, args...)
	} else {
		result = sc.redisClient.Eval(ctx, BatchCheckUserScript, keys, args...)
	}

	if err = result.Err(); err != nil {
		sc.logger.Errorf("Failed to batch check user status: %v", err)
		return nil, err
	}

	// 解析结果
	resultSlice, ok := result.Val().([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type")
	}

	statuses := make([]bool, len(resultSlice))
	for i, v := range resultSlice {
		if status, ok := v.(int64); ok {
			statuses[i] = status == 1
		}
	}

	return statuses, nil
}

// 获取秒杀统计信息
func (sc *SeckillCore) GetSeckillStats(ctx context.Context, productID int64) (*SeckillStats, error) {
	stockKey := fmt.Sprintf("seckill:stock:%d", productID)
	usersKey := fmt.Sprintf("seckill:users:%d", productID)
	activityKey := fmt.Sprintf("seckill:activity:%d", productID)

	keys := []string{stockKey, usersKey, activityKey}

	var result *redis.Cmd
	var err error

	if sha, exists := sc.scriptSHA["stats"]; exists {
		result = sc.redisClient.EvalSha(ctx, sha, keys)
	} else {
		result = sc.redisClient.Eval(ctx, SeckillStatsScript, keys)
	}

	if err = result.Err(); err != nil {
		sc.logger.Errorf("Failed to get seckill stats: %v", err)
		return nil, err
	}

	// 解析结果
	resultSlice, ok := result.Val().([]interface{})
	if !ok || len(resultSlice) < 3 {
		return nil, fmt.Errorf("unexpected result format")
	}

	stats := &SeckillStats{
		ProductID: productID,
	}

	if currentStock, ok := resultSlice[0].(int64); ok {
		stats.CurrentStock = currentStock
	}

	if userCount, ok := resultSlice[1].(int64); ok {
		stats.UserCount = userCount
	}

	if activityInfo, ok := resultSlice[2].(string); ok && activityInfo != "" {
		var activity SeckillActivity
		if err := json.Unmarshal([]byte(activityInfo), &activity); err == nil {
			stats.ActivityInfo = activity
		}
	}

	return stats, nil
}

// 检查用户是否已购买
func (sc *SeckillCore) IsUserPurchased(ctx context.Context, productID, userID int64) (bool, error) {
	statuses, err := sc.BatchCheckUserStatus(ctx, productID, []int64{userID})
	if err != nil {
		return false, err
	}

	if len(statuses) > 0 {
		return statuses[0], nil
	}

	return false, nil
}

// 预热活动数据
func (sc *SeckillCore) PrewarmActivity(ctx context.Context, activity *SeckillActivity) error {
	// 设置库存
	stockKey := fmt.Sprintf("seckill:stock:%d", activity.ProductID)
	err := sc.redisClient.Set(ctx, stockKey, activity.Stock, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to set stock: %w", err)
	}

	// 设置活动信息
	activityKey := fmt.Sprintf("seckill:activity:%d", activity.ProductID)
	activityData, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("failed to marshal activity: %w", err)
	}

	err = sc.redisClient.Set(ctx, activityKey, string(activityData), 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to set activity: %w", err)
	}

	sc.logger.Infof("Prewarmed activity for product %d with stock %d", activity.ProductID, activity.Stock)
	return nil
}

// 清理活动数据
func (sc *SeckillCore) CleanupActivity(ctx context.Context, productID int64) error {
	keys := []string{
		fmt.Sprintf("seckill:stock:%d", productID),
		fmt.Sprintf("seckill:users:%d", productID),
		fmt.Sprintf("seckill:activity:%d", productID),
	}

	err := sc.redisClient.Del(ctx, keys...).Err()
	if err != nil {
		return fmt.Errorf("failed to cleanup activity: %w", err)
	}

	sc.logger.Infof("Cleaned up activity for product %d", productID)
	return nil
}

// 生成订单ID
func (sc *SeckillCore) GenerateOrderID(productID, userID int64) string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("SK%d%d%d", productID, userID, timestamp)
}
