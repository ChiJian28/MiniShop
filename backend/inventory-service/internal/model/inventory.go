package model

import (
	"time"

	"gorm.io/gorm"
)

// 库存状态
const (
	InventoryStatusActive   = "active"   // 正常
	InventoryStatusInactive = "inactive" // 停用
	InventoryStatusLocked   = "locked"   // 锁定
)

// 库存操作类型
const (
	InventoryOpTypeDeduct = "deduct" // 扣减
	InventoryOpTypeAdd    = "add"    // 增加
	InventoryOpTypeSync   = "sync"   // 同步
	InventoryOpTypeInit   = "init"   // 初始化
)

// 库存表
type Inventory struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	ProductID   int64  `gorm:"uniqueIndex;not null" json:"product_id"`
	ProductName string `gorm:"size:255;not null" json:"product_name"`
	Stock       int64  `gorm:"not null;default:0" json:"stock"`      // 当前库存
	Reserved    int64  `gorm:"not null;default:0" json:"reserved"`   // 预留库存
	Available   int64  `gorm:"not null;default:0" json:"available"`  // 可用库存 = stock - reserved
	Version     int64  `gorm:"not null;default:0" json:"version"`    // 乐观锁版本号
	Status      string `gorm:"size:20;not null;index" json:"status"` // 库存状态
	MinStock    int64  `gorm:"not null;default:0" json:"min_stock"`  // 最小库存
	MaxStock    int64  `gorm:"not null;default:0" json:"max_stock"`  // 最大库存

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// 库存操作记录表
type InventoryLog struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	ProductID   int64  `gorm:"index;not null" json:"product_id"`
	OpType      string `gorm:"size:20;not null;index" json:"op_type"` // 操作类型
	Delta       int64  `gorm:"not null" json:"delta"`                 // 变化量（正数为增加，负数为减少）
	BeforeStock int64  `gorm:"not null" json:"before_stock"`          // 操作前库存
	AfterStock  int64  `gorm:"not null" json:"after_stock"`           // 操作后库存
	OrderID     string `gorm:"size:64;index" json:"order_id"`         // 关联订单ID
	Reason      string `gorm:"size:255" json:"reason"`                // 操作原因
	Operator    string `gorm:"size:100" json:"operator"`              // 操作者
	TraceID     string `gorm:"size:64;index" json:"trace_id"`         // 追踪ID

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// 库存差异记录表（健康检查发现的差异）
type InventoryDiff struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	ProductID  int64      `gorm:"index;not null" json:"product_id"`
	DBStock    int64      `gorm:"not null" json:"db_stock"`             // 数据库库存
	RedisStock int64      `gorm:"not null" json:"redis_stock"`          // Redis库存
	Diff       int64      `gorm:"not null" json:"diff"`                 // 差异量 = redis_stock - db_stock
	Status     string     `gorm:"size:20;not null;index" json:"status"` // pending, fixed, ignored
	FixedAt    *time.Time `json:"fixed_at,omitempty"`
	FixedBy    string     `gorm:"size:100" json:"fixed_by"`
	Remark     string     `gorm:"type:text" json:"remark"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// 库存预警记录表
type InventoryAlert struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	ProductID int64      `gorm:"index;not null" json:"product_id"`
	AlertType string     `gorm:"size:20;not null;index" json:"alert_type"` // low_stock, out_of_stock, diff_alert
	Message   string     `gorm:"type:text;not null" json:"message"`
	Level     string     `gorm:"size:10;not null;index" json:"level"`  // info, warning, error
	Status    string     `gorm:"size:20;not null;index" json:"status"` // pending, sent, ignored
	SentAt    *time.Time `json:"sent_at,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// 表名定义
func (Inventory) TableName() string {
	return "inventories"
}

func (InventoryLog) TableName() string {
	return "inventory_logs"
}

func (InventoryDiff) TableName() string {
	return "inventory_diffs"
}

func (InventoryAlert) TableName() string {
	return "inventory_alerts"
}

// 创建索引
func CreateIndexes(db *gorm.DB) error {
	// 商品ID唯一索引
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_inventories_product_id 
		ON inventories (product_id) 
		WHERE deleted_at IS NULL
	`).Error; err != nil {
		return err
	}

	// 库存日志复合索引
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_inventory_logs_product_time 
		ON inventory_logs (product_id, created_at DESC)
	`).Error; err != nil {
		return err
	}

	// 库存差异复合索引
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_inventory_diffs_status_time 
		ON inventory_diffs (status, created_at DESC)
	`).Error; err != nil {
		return err
	}

	return nil
}

// 库存同步请求DTO
type SyncStockRequest struct {
	ProductID int64  `json:"product_id" binding:"required"`
	Delta     int64  `json:"delta" binding:"required"` // 变化量（正数增加，负数减少）
	OrderID   string `json:"order_id"`                 // 关联订单ID
	Reason    string `json:"reason"`                   // 操作原因
	TraceID   string `json:"trace_id"`                 // 追踪ID
}

// 库存同步响应DTO
type SyncStockResponse struct {
	ProductID   int64  `json:"product_id"`
	BeforeStock int64  `json:"before_stock"`
	AfterStock  int64  `json:"after_stock"`
	Delta       int64  `json:"delta"`
	Success     bool   `json:"success"`
	Message     string `json:"message,omitempty"`
}

// 库存查询请求DTO
type GetInventoryRequest struct {
	ProductID int64 `form:"product_id" binding:"required"`
}

// 库存响应DTO
type InventoryResponse struct {
	ProductID   int64     `json:"product_id"`
	ProductName string    `json:"product_name"`
	Stock       int64     `json:"stock"`
	Reserved    int64     `json:"reserved"`
	Available   int64     `json:"available"`
	Status      string    `json:"status"`
	MinStock    int64     `json:"min_stock"`
	MaxStock    int64     `json:"max_stock"`
	Version     int64     `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// 库存批量查询请求DTO
type BatchGetInventoryRequest struct {
	ProductIDs []int64 `json:"product_ids" binding:"required"`
}

// 库存健康检查响应DTO
type InventoryHealthResponse struct {
	ProductID  int64  `json:"product_id"`
	DBStock    int64  `json:"db_stock"`
	RedisStock int64  `json:"redis_stock"`
	Diff       int64  `json:"diff"`
	Status     string `json:"status"`
}

// 库存统计DTO
type InventoryStats struct {
	TotalProducts      int64 `json:"total_products"`
	LowStockProducts   int64 `json:"low_stock_products"`
	OutOfStockProducts int64 `json:"out_of_stock_products"`
	TotalStock         int64 `json:"total_stock"`
	TotalReserved      int64 `json:"total_reserved"`
	TotalAvailable     int64 `json:"total_available"`
}

// 分页结果
type PageResult struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// 库存查询参数
type InventoryQuery struct {
	ProductID int64  `form:"product_id"`
	Status    string `form:"status"`
	MinStock  *int64 `form:"min_stock"`
	MaxStock  *int64 `form:"max_stock"`
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"page_size,default=20"`
}
