package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"order-service/internal/config"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// 消息处理器接口
type MessageHandler interface {
	HandleSeckillOrder(ctx context.Context, message *SeckillOrderMessage) error
	HandleStockUpdate(ctx context.Context, message *StockUpdateMessage) error
	HandleUserNotify(ctx context.Context, message *UserNotifyMessage) error
}

// RabbitMQ 消费者
type RabbitMQConsumer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	queue    string
	handler  MessageHandler
	logger   *logrus.Logger
	prefetch int
}

// 创建 RabbitMQ 消费者
func NewRabbitMQConsumer(cfg *config.RabbitMQConfig, handler MessageHandler, logger *logrus.Logger) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(cfg.URL)
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
		queue:    cfg.Queue,
		handler:  handler,
		logger:   logger,
		prefetch: cfg.PrefetchCount,
	}

	// 声明交换机
	err = channel.ExchangeDeclare(
		cfg.Exchange,   // name
		"topic",        // type
		cfg.Durable,    // durable
		cfg.AutoDelete, // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		consumer.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// 声明队列
	_, err = channel.QueueDeclare(
		cfg.Queue,      // name
		cfg.Durable,    // durable
		cfg.AutoDelete, // delete when unused
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		consumer.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// 绑定队列到交换机
	err = channel.QueueBind(
		cfg.Queue,      // queue name
		cfg.RoutingKey, // routing key
		cfg.Exchange,   // exchange
		false,
		nil,
	)
	if err != nil {
		consumer.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	logger.Info("RabbitMQ consumer initialized successfully")
	return consumer, nil
}

// 开始消费消息
func (c *RabbitMQConsumer) StartConsume(ctx context.Context) error {
	// 设置 QoS
	err := c.channel.Qos(
		c.prefetch, // prefetch count
		0,          // prefetch size
		false,      // global
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

	c.logger.Info("Started consuming RabbitMQ messages")
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

	// 根据消息类型处理
	var err error
	switch baseMsg.MessageType {
	case MessageTypeSeckillOrder:
		var orderMsg SeckillOrderMessage
		if err = json.Unmarshal(msg.Body, &orderMsg); err == nil {
			err = c.handler.HandleSeckillOrder(ctx, &orderMsg)
		}
	case MessageTypeStockUpdate:
		var stockMsg StockUpdateMessage
		if err = json.Unmarshal(msg.Body, &stockMsg); err == nil {
			err = c.handler.HandleStockUpdate(ctx, &stockMsg)
		}
	case MessageTypeUserNotify:
		var notifyMsg UserNotifyMessage
		if err = json.Unmarshal(msg.Body, &notifyMsg); err == nil {
			err = c.handler.HandleUserNotify(ctx, &notifyMsg)
		}
	default:
		c.logger.Warnf("Unknown message type: %s", baseMsg.MessageType)
		msg.Ack(false)
		return
	}

	if err != nil {
		c.logger.Errorf("Failed to handle message: %v", err)
		msg.Nack(false, true) // 重新入队
		return
	}

	// 确认消息
	msg.Ack(false)
	c.logger.Debugf("Successfully processed message of type: %s", baseMsg.MessageType)
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

// Kafka 消费者
type KafkaConsumer struct {
	reader  *kafka.Reader
	handler MessageHandler
	logger  *logrus.Logger
}

// 创建 Kafka 消费者
func NewKafkaConsumer(cfg *config.KafkaConfig, handler MessageHandler, logger *logrus.Logger) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		Topic:       cfg.Topic,
		GroupID:     cfg.GroupID,
		Partition:   cfg.Partition,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		MaxWait:     1 * time.Second,
		StartOffset: kafka.LastOffset,
	})

	return &KafkaConsumer{
		reader:  reader,
		handler: handler,
		logger:  logger,
	}
}

// 开始消费消息
func (c *KafkaConsumer) StartConsume(ctx context.Context) error {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := c.reader.ReadMessage(ctx)
				if err != nil {
					c.logger.Errorf("Failed to read message from Kafka: %v", err)
					continue
				}
				c.handleMessage(ctx, msg)
			}
		}
	}()

	c.logger.Info("Started consuming Kafka messages")
	return nil
}

// 处理消息
func (c *KafkaConsumer) handleMessage(ctx context.Context, msg kafka.Message) {
	// 解析消息类型
	var baseMsg struct {
		MessageType string `json:"message_type"`
	}

	if err := json.Unmarshal(msg.Value, &baseMsg); err != nil {
		c.logger.Errorf("Failed to parse message type: %v", err)
		return
	}

	// 根据消息类型处理
	var err error
	switch baseMsg.MessageType {
	case MessageTypeSeckillOrder:
		var orderMsg SeckillOrderMessage
		if err = json.Unmarshal(msg.Value, &orderMsg); err == nil {
			err = c.handler.HandleSeckillOrder(ctx, &orderMsg)
		}
	case MessageTypeStockUpdate:
		var stockMsg StockUpdateMessage
		if err = json.Unmarshal(msg.Value, &stockMsg); err == nil {
			err = c.handler.HandleStockUpdate(ctx, &stockMsg)
		}
	case MessageTypeUserNotify:
		var notifyMsg UserNotifyMessage
		if err = json.Unmarshal(msg.Value, &notifyMsg); err == nil {
			err = c.handler.HandleUserNotify(ctx, &notifyMsg)
		}
	default:
		c.logger.Warnf("Unknown message type: %s", baseMsg.MessageType)
		return
	}

	if err != nil {
		c.logger.Errorf("Failed to handle message: %v", err)
		return
	}

	c.logger.Debugf("Successfully processed Kafka message: topic=%s, partition=%d, offset=%d",
		msg.Topic, msg.Partition, msg.Offset)
}

// 关闭消费者
func (c *KafkaConsumer) Close() error {
	if c.reader != nil {
		err := c.reader.Close()
		if err != nil {
			c.logger.Errorf("Failed to close Kafka reader: %v", err)
			return err
		}
	}
	c.logger.Info("Kafka consumer closed")
	return nil
}

// 消费者接口
type Consumer interface {
	StartConsume(ctx context.Context) error
	Close() error
}

// 确保实现了接口
var _ Consumer = (*RabbitMQConsumer)(nil)
var _ Consumer = (*KafkaConsumer)(nil)
