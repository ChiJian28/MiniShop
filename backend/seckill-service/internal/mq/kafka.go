package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Kafka 生产者
type KafkaProducer struct {
	writer *kafka.Writer
	logger *logrus.Logger
}

// Kafka 配置
type KafkaConfig struct {
	Brokers   []string
	Topic     string
	Partition int
	Timeout   time.Duration
}

// 创建 Kafka 生产者
func NewKafkaProducer(config *KafkaConfig, logger *logrus.Logger) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Topic:        config.Topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}

	return &KafkaProducer{
		writer: writer,
		logger: logger,
	}
}

// 发送消息
func (p *KafkaProducer) SendMessage(ctx context.Context, key string, message []byte) error {
	msg := kafka.Message{
		Key:   []byte(key),
		Value: message,
		Time:  time.Now(),
	}

	err := p.writer.WriteMessages(ctx, msg)
	if err != nil {
		p.logger.Errorf("Failed to write message to Kafka: %v", err)
		return fmt.Errorf("failed to write message: %w", err)
	}

	p.logger.Debugf("Message sent to Kafka topic: %s, key: %s", p.writer.Topic, key)
	return nil
}

// 发送秒杀订单消息
func (p *KafkaProducer) SendSeckillOrderMessage(ctx context.Context, msg *SeckillOrderMessage) error {
	data, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	key := fmt.Sprintf("seckill_order_%d_%d", msg.ProductID, msg.UserID)
	return p.SendMessage(ctx, key, data)
}

// 发送库存更新消息
func (p *KafkaProducer) SendStockUpdateMessage(ctx context.Context, msg *StockUpdateMessage) error {
	data, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	key := fmt.Sprintf("stock_update_%d", msg.ProductID)
	return p.SendMessage(ctx, key, data)
}

// 发送用户通知消息
func (p *KafkaProducer) SendUserNotifyMessage(ctx context.Context, msg *UserNotifyMessage) error {
	data, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	key := fmt.Sprintf("user_notify_%d", msg.UserID)
	return p.SendMessage(ctx, key, data)
}

// 批量发送消息
func (p *KafkaProducer) SendBatchMessages(ctx context.Context, messages []kafka.Message) error {
	err := p.writer.WriteMessages(ctx, messages...)
	if err != nil {
		p.logger.Errorf("Failed to write batch messages to Kafka: %v", err)
		return fmt.Errorf("failed to write batch messages: %w", err)
	}

	p.logger.Debugf("Batch messages sent to Kafka topic: %s, count: %d", p.writer.Topic, len(messages))
	return nil
}

// 健康检查
func (p *KafkaProducer) HealthCheck(ctx context.Context) error {
	// 尝试获取 topic 的元数据
	conn, err := kafka.DialLeader(ctx, "tcp", p.writer.Addr.String(), p.writer.Topic, 0)
	if err != nil {
		return fmt.Errorf("failed to connect to Kafka: %w", err)
	}
	defer conn.Close()

	return nil
}

// 关闭生产者
func (p *KafkaProducer) Close() error {
	if p.writer != nil {
		err := p.writer.Close()
		if err != nil {
			p.logger.Errorf("Failed to close Kafka writer: %v", err)
			return err
		}
	}
	p.logger.Info("Kafka producer closed")
	return nil
}

// Kafka 消费者
type KafkaConsumer struct {
	reader   *kafka.Reader
	logger   *logrus.Logger
	handlers map[string]MessageHandler
}

// 创建 Kafka 消费者
func NewKafkaConsumer(config *KafkaConfig, groupID string, logger *logrus.Logger) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     config.Brokers,
		Topic:       config.Topic,
		GroupID:     groupID,
		Partition:   config.Partition,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		MaxWait:     1 * time.Second,
		StartOffset: kafka.LastOffset,
	})

	return &KafkaConsumer{
		reader:   reader,
		logger:   logger,
		handlers: make(map[string]MessageHandler),
	}
}

// 注册消息处理器
func (c *KafkaConsumer) RegisterHandler(messageType string, handler MessageHandler) {
	c.handlers[messageType] = handler
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

	c.logger.Info("Started consuming messages from Kafka")
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

	// 查找处理器
	handler, exists := c.handlers[baseMsg.MessageType]
	if !exists {
		c.logger.Warnf("No handler found for message type: %s", baseMsg.MessageType)
		return
	}

	// 处理消息
	if err := handler.Handle(ctx, msg.Value); err != nil {
		c.logger.Errorf("Failed to handle message: %v", err)
		return
	}

	c.logger.Debugf("Message handled successfully: topic=%s, partition=%d, offset=%d",
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

// 消息队列接口
type MessageQueue interface {
	SendSeckillOrderMessage(ctx context.Context, msg *SeckillOrderMessage) error
	SendStockUpdateMessage(ctx context.Context, msg *StockUpdateMessage) error
	SendUserNotifyMessage(ctx context.Context, msg *UserNotifyMessage) error
	Close() error
}

// 确保 RabbitMQ 和 Kafka 生产者都实现了 MessageQueue 接口
var _ MessageQueue = (*RabbitMQProducer)(nil)
var _ MessageQueue = (*KafkaProducer)(nil)
