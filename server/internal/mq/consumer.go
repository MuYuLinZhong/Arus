package mq

import (
	"encoding/json"
	"sync"
	"time"

	"promthus/internal/logger"
	"promthus/internal/model"
	"promthus/internal/repository"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type AuditConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	buffer  []model.AuditLog
	mu      sync.Mutex
	done    chan struct{}
}

func NewAuditConsumer(url string, workerCount int) (*AuditConsumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	if err := ch.Qos(10, 0, false); err != nil {
		return nil, err
	}

	consumer := &AuditConsumer{
		conn:    conn,
		channel: ch,
		buffer:  make([]model.AuditLog, 0, 100),
		done:    make(chan struct{}),
	}

	msgs, err := ch.Consume("audit.queue", "", false, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	for i := 0; i < workerCount; i++ {
		go consumer.worker(msgs)
	}

	go consumer.flushLoop()

	return consumer, nil
}

func (c *AuditConsumer) worker(msgs <-chan amqp.Delivery) {
	for msg := range msgs {
		var audit AuditMessage
		if err := json.Unmarshal(msg.Body, &audit); err != nil {
			logger.Error("failed to unmarshal audit message", zap.Error(err))
			_ = msg.Nack(false, false)
			continue
		}

		deviceType := audit.DeviceType
		if deviceType == "" {
			deviceType = model.DeviceTypeLock
		}
		log := model.AuditLog{
			UserID:      audit.UserID,
			DeviceID:    audit.DeviceID,
			DeviceType:  deviceType,
			Action:      audit.Action,
			ResultCode:  audit.ResultCode,
			ClientIP:    audit.ClientIP,
			DeviceModel: audit.DeviceModel,
			OccurredAt:  time.UnixMilli(audit.OccurredAt),
		}
		if audit.Extra != nil {
			log.Extra = model.JSON(audit.Extra)
		}

		c.mu.Lock()
		c.buffer = append(c.buffer, log)
		shouldFlush := len(c.buffer) >= 100
		c.mu.Unlock()

		if shouldFlush {
			c.flush()
		}

		_ = msg.Ack(false)
	}
}

func (c *AuditConsumer) flushLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.flush()
		case <-c.done:
			c.flush()
			return
		}
	}
}

func (c *AuditConsumer) flush() {
	c.mu.Lock()
	if len(c.buffer) == 0 {
		c.mu.Unlock()
		return
	}
	batch := c.buffer
	c.buffer = make([]model.AuditLog, 0, 100)
	c.mu.Unlock()

	if err := repository.DB.CreateInBatches(batch, len(batch)).Error; err != nil {
		logger.Error("failed to batch insert audit logs",
			zap.Error(err),
			zap.Int("count", len(batch)))
	} else {
		logger.Debug("flushed audit logs", zap.Int("count", len(batch)))
	}
}

func (c *AuditConsumer) Close() {
	close(c.done)
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
