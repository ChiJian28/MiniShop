package mq

import (
	"encoding/json"
	"time"
)

// 秒杀订单消息
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

// 消息类型常量
const (
	MessageTypeSeckillOrder = "seckill_order"
	MessageTypeStockUpdate  = "stock_update"
	MessageTypeUserNotify   = "user_notify"
)

// 序列化消息
func (m *SeckillOrderMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// 反序列化消息
func (m *SeckillOrderMessage) Unmarshal(data []byte) error {
	return json.Unmarshal(data, m)
}

// 创建秒杀订单消息
func NewSeckillOrderMessage(orderID string, productID, userID, quantity int64, price float64, traceID string) *SeckillOrderMessage {
	return &SeckillOrderMessage{
		OrderID:     orderID,
		ProductID:   productID,
		UserID:      userID,
		Quantity:    quantity,
		Price:       price,
		CreateTime:  time.Now(),
		MessageType: MessageTypeSeckillOrder,
		TraceID:     traceID,
	}
}

// 库存更新消息
type StockUpdateMessage struct {
	ProductID      int64     `json:"product_id"`
	RemainingStock int64     `json:"remaining_stock"`
	UpdateTime     time.Time `json:"update_time"`
	MessageType    string    `json:"message_type"`
	TraceID        string    `json:"trace_id"`
}

// 创建库存更新消息
func NewStockUpdateMessage(productID, remainingStock int64, traceID string) *StockUpdateMessage {
	return &StockUpdateMessage{
		ProductID:      productID,
		RemainingStock: remainingStock,
		UpdateTime:     time.Now(),
		MessageType:    MessageTypeStockUpdate,
		TraceID:        traceID,
	}
}

// 序列化库存更新消息
func (m *StockUpdateMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
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

// 通知类型常量
const (
	NotifyTypeSeckillSuccess = "seckill_success"
	NotifyTypeSeckillFailed  = "seckill_failed"
	NotifyTypeStockAlert     = "stock_alert"
)

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

// 序列化用户通知消息
func (m *UserNotifyMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}
