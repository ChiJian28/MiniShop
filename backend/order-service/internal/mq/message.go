package mq

import (
	"encoding/json"
	"time"
)

// 秒杀订单消息（从 seckill-service 接收）
type SeckillOrderMessage struct {
	OrderID     string    `json:"order_id"`
	ProductID   int64     `json:"product_id"`
	UserID      int64     `json:"user_id"`
	Quantity    int64     `json:"quantity"`
	Price       float64   `json:"price"`
	CreateTime  time.Time `json:"create_time"`
	MessageType string    `json:"message_type"`
	TraceID     string    `json:"trace_id"`
}

// 库存更新消息
type StockUpdateMessage struct {
	ProductID      int64     `json:"product_id"`
	RemainingStock int64     `json:"remaining_stock"`
	UpdateTime     time.Time `json:"update_time"`
	MessageType    string    `json:"message_type"`
	TraceID        string    `json:"trace_id"`
}

// 用户通知消息
type UserNotifyMessage struct {
	UserID      int64     `json:"user_id"`
	ProductID   int64     `json:"product_id"`
	NotifyType  string    `json:"notify_type"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	CreateTime  time.Time `json:"create_time"`
	MessageType string    `json:"message_type"`
	TraceID     string    `json:"trace_id"`
}

// 订单状态变更消息（发送给其他服务）
type OrderStatusMessage struct {
	OrderID     string    `json:"order_id"`
	UserID      int64     `json:"user_id"`
	ProductID   int64     `json:"product_id"`
	OldStatus   string    `json:"old_status"`
	NewStatus   string    `json:"new_status"`
	UpdateTime  time.Time `json:"update_time"`
	MessageType string    `json:"message_type"`
	TraceID     string    `json:"trace_id"`
}

// 消息类型常量
const (
	MessageTypeSeckillOrder = "seckill_order"
	MessageTypeStockUpdate  = "stock_update"
	MessageTypeUserNotify   = "user_notify"
	MessageTypeOrderStatus  = "order_status"
)

// 通知类型常量
const (
	NotifyTypeSeckillSuccess = "seckill_success"
	NotifyTypeSeckillFailed  = "seckill_failed"
	NotifyTypeOrderCreated   = "order_created"
	NotifyTypeOrderPaid      = "order_paid"
	NotifyTypeOrderCancelled = "order_cancelled"
)

// 序列化消息
func (m *SeckillOrderMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// 反序列化消息
func (m *SeckillOrderMessage) Unmarshal(data []byte) error {
	return json.Unmarshal(data, m)
}

// 序列化库存更新消息
func (m *StockUpdateMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// 序列化用户通知消息
func (m *UserNotifyMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// 序列化订单状态消息
func (m *OrderStatusMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// 创建订单状态变更消息
func NewOrderStatusMessage(orderID string, userID, productID int64, oldStatus, newStatus, traceID string) *OrderStatusMessage {
	return &OrderStatusMessage{
		OrderID:     orderID,
		UserID:      userID,
		ProductID:   productID,
		OldStatus:   oldStatus,
		NewStatus:   newStatus,
		UpdateTime:  time.Now(),
		MessageType: MessageTypeOrderStatus,
		TraceID:     traceID,
	}
}

// 创建用户通知消息
func NewUserNotifyMessage(userID, productID int64, notifyType, title, content, traceID string) *UserNotifyMessage {
	return &UserNotifyMessage{
		UserID:      userID,
		ProductID:   productID,
		NotifyType:  notifyType,
		Title:       title,
		Content:     content,
		CreateTime:  time.Now(),
		MessageType: MessageTypeUserNotify,
		TraceID:     traceID,
	}
}
