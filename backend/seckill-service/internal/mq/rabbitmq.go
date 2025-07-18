package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// RabbitMQ 生产者
type RabbitMQProducer struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	exchange   string
	queue      string
	routingKey string
	logger     *logrus.Logger
	durable    bool
	autoDelete bool
}

// RabbitMQ 配置
type RabbitMQConfig struct {
	URL        string
	Exchange   string
	Queue      string
	RoutingKey string
	Durable    bool
	AutoDelete bool
}

// 创建 RabbitMQ 生产者
func NewRabbitMQProducer(config *RabbitMQConfig, logger *logrus.Logger) (*RabbitMQProducer, error) {
	conn, err := amqp.Dial(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	producer := &RabbitMQProducer{
		conn:       conn,
		channel:    channel,
		exchange:   config.Exchange,
		queue:      config.Queue,
		routingKey: config.RoutingKey,
		logger:     logger,
		durable:    config.Durable,
		autoDelete: config.AutoDelete,
	}

	// 声明交换机
	err = channel.ExchangeDeclare(
		config.Exchange,   // name
		"topic",           // type
		config.Durable,    // durable
		config.AutoDelete, // auto-deleted
		false,             // internal
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// 声明队列
	_, err = channel.QueueDeclare(
		config.Queue,      // name
		config.Durable,    // durable
		config.AutoDelete, // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// 绑定队列到交换机
	err = channel.QueueBind(
		config.Queue,      // queue name
		config.RoutingKey, // routing key
		config.Exchange,   // exchange
		false,
		nil,
	)
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	logger.Info("RabbitMQ producer initialized successfully")
	return producer, nil
}

// 发送消息
func (p *RabbitMQProducer) SendMessage(ctx context.Context, message []byte) error {
	return p.SendMessageWithRoutingKey(ctx, message, p.routingKey)
}

// 发送消息（指定路由键）
func (p *RabbitMQProducer) SendMessageWithRoutingKey(ctx context.Context, message []byte, routingKey string) error {
	// 设置消息属性
	msg := amqp.Publishing{
		ContentType:  "application/json",
		Body:         message,
		DeliveryMode: amqp.Persistent, // 持久化消息
		Timestamp:    time.Now(),
		MessageId:    fmt.Sprintf("%d", time.Now().UnixNano()),
	}

	// 发送消息
	err := p.channel.Publish(
		p.exchange, // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		msg,        // message
	)

	if err != nil {
		p.logger.Errorf("Failed to publish message: %v", err)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	p.logger.Debugf("Message sent to exchange: %s, routing key: %s", p.exchange, routingKey)
	return nil
}

// 发送秒杀订单消息
func (p *RabbitMQProducer) SendSeckillOrderMessage(ctx context.Context, msg *SeckillOrderMessage) error {
	data, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return p.SendMessageWithRoutingKey(ctx, data, "seckill.order.create")
}

// 发送库存更新消息
func (p *RabbitMQProducer) SendStockUpdateMessage(ctx context.Context, msg *StockUpdateMessage) error {
	data, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return p.SendMessageWithRoutingKey(ctx, data, "seckill.stock.update")
}

// 发送用户通知消息
func (p *RabbitMQProducer) SendUserNotifyMessage(ctx context.Context, msg *UserNotifyMessage) error {
	data, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return p.SendMessageWithRoutingKey(ctx, data, "seckill.user.notify")
}

// 批量发送消息
func (p *RabbitMQProducer) SendBatchMessages(ctx context.Context, messages [][]byte, routingKey string) error {
	for _, message := range messages {
		if err := p.SendMessageWithRoutingKey(ctx, message, routingKey); err != nil {
			return err
		}
	}
	return nil
}

// 健康检查
func (p *RabbitMQProducer) HealthCheck() error {
	if p.conn == nil || p.conn.IsClosed() {
		return fmt.Errorf("connection is closed")
	}
	if p.channel == nil {
		return fmt.Errorf("channel is nil")
	}
	return nil
}

// 关闭连接
func (p *RabbitMQProducer) Close() error {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
	p.logger.Info("RabbitMQ producer closed")
	return nil
}

// 重连
func (p *RabbitMQProducer) Reconnect(config *RabbitMQConfig) error {
	p.Close()

	conn, err := amqp.Dial(config.URL)
	if err != nil {
		return fmt.Errorf("failed to reconnect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	p.conn = conn
	p.channel = channel

	p.logger.Info("RabbitMQ producer reconnected successfully")
	return nil
}

// RabbitMQ 消费者
type RabbitMQConsumer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	queue    string
	logger   *logrus.Logger
	handlers map[string]MessageHandler
}

// 消息处理器接口
type MessageHandler interface {
	Handle(ctx context.Context, message []byte) error
}

// 创建 RabbitMQ 消费者
func NewRabbitMQConsumer(config *RabbitMQConfig, logger *logrus.Logger) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	consumer := &RabbitMQConsumer{
		conn:     conn,
		channel:  channel,
		queue:    config.Queue,
		logger:   logger,
		handlers: make(map[string]MessageHandler),
	}

	logger.Info("RabbitMQ consumer initialized successfully")
	return consumer, nil
}

// 注册消息处理器
func (c *RabbitMQConsumer) RegisterHandler(messageType string, handler MessageHandler) {
	c.handlers[messageType] = handler
}

// 开始消费消息
func (c *RabbitMQConsumer) StartConsume(ctx context.Context) error {
	// 设置 QoS
	err := c.channel.Qos(
		10,    // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// 开始消费
	msgs, err := c.channel.Consume(
		c.queue, // queue
		"",      // consumer
		false,   // auto-ack
		false,   // exclusive
		false,   // no-local
		false,   // no-wait
		nil,     // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				c.handleMessage(ctx, msg)
			}
		}
	}()

	c.logger.Info("Started consuming messages")
	return nil
}

// 处理消息
func (c *RabbitMQConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) {
	// 解析消息类型
	var baseMsg struct {
		MessageType string `json:"message_type"`
	}

	if err := json.Unmarshal(msg.Body, &baseMsg); err != nil {
		c.logger.Errorf("Failed to parse message type: %v", err)
		msg.Nack(false, false)
		return
	}

	// 查找处理器
	handler, exists := c.handlers[baseMsg.MessageType]
	if !exists {
		c.logger.Warnf("No handler found for message type: %s", baseMsg.MessageType)
		msg.Ack(false)
		return
	}

	// 处理消息
	if err := handler.Handle(ctx, msg.Body); err != nil {
		c.logger.Errorf("Failed to handle message: %v", err)
		msg.Nack(false, true) // 重新入队
		return
	}

	// 确认消息
	msg.Ack(false)
}

// 关闭消费者
func (c *RabbitMQConsumer) Close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	c.logger.Info("RabbitMQ consumer closed")
	return nil
}
