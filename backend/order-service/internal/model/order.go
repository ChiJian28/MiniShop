package model

import (
	"time"

	"gorm.io/gorm"
)

// 订单状态
const (
	OrderStatusPending   = "pending"   // 待支付
	OrderStatusPaid      = "paid"      // 已支付
	OrderStatusCancelled = "cancelled" // 已取消
	OrderStatusExpired   = "expired"   // 已过期
	OrderStatusRefunded  = "refunded"  // 已退款
)

// 订单类型
const (
	OrderTypeSeckill = "seckill" // 秒杀订单
	OrderTypeNormal  = "normal"  // 普通订单
)

// 失败类型常量
const (
	FailureTypeInventoryLock = "inventory_lock" // 库存锁定失败
	FailureTypePayment       = "payment"        // 支付失败
	FailureTypeOrderCreation = "order_creation" // 订单创建失败
)

// 订单表
type Order struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	OrderID     string  `gorm:"uniqueIndex;size:64;not null" json:"order_id"`
	UserID      int64   `gorm:"index;not null" json:"user_id"`
	ProductID   int64   `gorm:"index;not null" json:"product_id"`
	ProductName string  `gorm:"size:255;not null" json:"product_name"`
	Quantity    int64   `gorm:"not null" json:"quantity"`
	Price       float64 `gorm:"type:decimal(10,2);not null" json:"price"`
	TotalAmount float64 `gorm:"type:decimal(10,2);not null" json:"total_amount"`
	Status      string  `gorm:"size:20;not null;index" json:"status"`
	OrderType   string  `gorm:"size:20;not null;index" json:"order_type"`
	TraceID     string  `gorm:"size:64;index" json:"trace_id"`

	// 时间字段
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	ExpiredAt   *time.Time     `gorm:"index" json:"expired_at,omitempty"`
	PaidAt      *time.Time     `json:"paid_at,omitempty"`
	CancelledAt *time.Time     `json:"cancelled_at,omitempty"`

	// 关联字段
	OrderItems    []OrderItem    `gorm:"foreignKey:OrderID;references:OrderID" json:"order_items,omitempty"`
	OrderFailures []OrderFailure `gorm:"foreignKey:OrderID;references:OrderID" json:"order_failures,omitempty"`
}

// 订单明细表
type OrderItem struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	OrderID     string  `gorm:"size:64;not null;index" json:"order_id"`
	ProductID   int64   `gorm:"not null" json:"product_id"`
	ProductName string  `gorm:"size:255;not null" json:"product_name"`
	Quantity    int64   `gorm:"not null" json:"quantity"`
	Price       float64 `gorm:"type:decimal(10,2);not null" json:"price"`
	TotalAmount float64 `gorm:"type:decimal(10,2);not null" json:"total_amount"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// 订单失败记录表（用于补偿）
type OrderFailure struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	OrderID     string     `gorm:"size:64;not null;index" json:"order_id"`
	UserID      int64      `gorm:"not null" json:"user_id"`
	ProductID   int64      `gorm:"not null" json:"product_id"`
	FailureType string     `gorm:"size:50;not null;index" json:"failure_type"` // 失败类型
	MessageData string     `gorm:"type:text" json:"message_data"`              // 原始消息数据
	ErrorMsg    string     `gorm:"type:text" json:"error_msg"`
	RetryCount  int        `gorm:"default:0" json:"retry_count"`
	MaxRetries  int        `gorm:"default:3" json:"max_retries"`
	NextRetryAt *time.Time `gorm:"index" json:"next_retry_at,omitempty"`
	Status      string     `gorm:"size:20;not null;index" json:"status"` // pending, processing, failed, success

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// 幂等性记录表
type OrderIdempotency struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	UserID    int64  `gorm:"not null" json:"user_id"`
	ProductID int64  `gorm:"not null" json:"product_id"`
	OrderID   string `gorm:"size:64;not null" json:"order_id"`
	TraceID   string `gorm:"size:64" json:"trace_id"`
	Status    string `gorm:"size:20;not null" json:"status"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// 订单统计表
type OrderStats struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Date            time.Time `gorm:"type:date;uniqueIndex" json:"date"`
	TotalOrders     int64     `gorm:"default:0" json:"total_orders"`
	SeckillOrders   int64     `gorm:"default:0" json:"seckill_orders"`
	PaidOrders      int64     `gorm:"default:0" json:"paid_orders"`
	CancelledOrders int64     `gorm:"default:0" json:"cancelled_orders"`
	TotalAmount     float64   `gorm:"type:decimal(15,2);default:0" json:"total_amount"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// 表初始化
func (Order) TableName() string {
	return "orders"
}

func (OrderItem) TableName() string {
	return "order_items"
}

func (OrderFailure) TableName() string {
	return "order_failures"
}

func (OrderIdempotency) TableName() string {
	return "order_idempotency"
}

func (OrderStats) TableName() string {
	return "order_stats"
}

// 创建唯一索引（防重复下单）
func CreateUniqueIndexes(db *gorm.DB) error {
	// 用户-商品唯一索引（防止重复下单）
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_user_product_unique 
		ON order_idempotency (user_id, product_id) 
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return err
	}

	// 订单ID唯一索引
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_order_id_unique 
		ON orders (order_id) 
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return err
	}

	return nil
}

// 订单请求DTO
type CreateOrderRequest struct {
	OrderID     string  `json:"order_id" binding:"required"`
	UserID      int64   `json:"user_id" binding:"required"`
	ProductID   int64   `json:"product_id" binding:"required"`
	ProductName string  `json:"product_name" binding:"required"`
	Quantity    int64   `json:"quantity" binding:"required,min=1"`
	Price       float64 `json:"price" binding:"required,min=0"`
	OrderType   string  `json:"order_type" binding:"required"`
	TraceID     string  `json:"trace_id"`
}

// 订单响应DTO
type OrderResponse struct {
	OrderID     string     `json:"order_id"`
	UserID      int64      `json:"user_id"`
	ProductID   int64      `json:"product_id"`
	ProductName string     `json:"product_name"`
	Quantity    int64      `json:"quantity"`
	Price       float64    `json:"price"`
	TotalAmount float64    `json:"total_amount"`
	Status      string     `json:"status"`
	OrderType   string     `json:"order_type"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiredAt   *time.Time `json:"expired_at,omitempty"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
}

// 订单查询参数
type OrderQuery struct {
	UserID    int64  `form:"user_id"`
	ProductID int64  `form:"product_id"`
	Status    string `form:"status"`
	OrderType string `form:"order_type"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"page_size,default=20"`
}

// 分页结果
type PageResult struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}
