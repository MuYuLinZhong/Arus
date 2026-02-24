package mq

import (
	"context"
	"encoding/json"
	"time"

	"promthus/internal/logger"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

type AuditMessage struct {
	MessageID   string                 `json:"message_id"`
	Version     string                 `json:"version"`
	Source      string                 `json:"source"`
	OccurredAt  int64                  `json:"occurred_at"`
	UserID      int64                  `json:"user_id"`
	DeviceID    string                 `json:"device_id"`
	DeviceType  string                 `json:"device_type"` // lock | sensor | ...
	Action      string                 `json:"action"`
	ResultCode  int16                  `json:"result_code"`
	ClientIP    string                 `json:"client_ip"`
	DeviceModel string                 `json:"device_model"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

type NotifyMessage struct {
	MessageID string                 `json:"message_id"`
	Version   string                 `json:"version"`
	Source    string                 `json:"source"`
	OccurredAt int64                 `json:"occurred_at"`
	AlertType string                 `json:"alert_type"`
	DeviceID  string                 `json:"device_id"`
	Severity  int16                  `json:"severity"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

func NewPublisher(url string) (*Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	queues := []string{"audit.queue", "notify.queue", "audit.dlq", "notify.dlq"}
	for _, q := range queues {
		_, err := ch.QueueDeclare(q, true, false, false, false, nil)
		if err != nil {
			logger.Warn("failed to declare queue", zap.String("queue", q), zap.Error(err))
		}
	}

	return &Publisher{conn: conn, channel: ch}, nil
}

func (p *Publisher) PublishAudit(msg *AuditMessage) error {
	msg.MessageID = uuid.New().String()
	msg.Version = "1.0"
	msg.Source = "lock-service"
	msg.OccurredAt = time.Now().UnixMilli()

	return p.publish("audit.queue", msg)
}

func (p *Publisher) PublishNotify(msg *NotifyMessage) error {
	msg.MessageID = uuid.New().String()
	msg.Version = "1.0"
	msg.Source = "lock-service"
	msg.OccurredAt = time.Now().UnixMilli()

	return p.publish("notify.queue", msg)
}

func (p *Publisher) publish(queue string, msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return p.channel.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *Publisher) Close() {
	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
}
