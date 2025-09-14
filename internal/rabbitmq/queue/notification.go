package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"

	"github.com/aliskhannn/delayed-notifier/internal/config"
)

type NotificationMessage struct {
	ID      uuid.UUID `json:"id"`
	SendAt  time.Time `json:"send_at"`
	Message string    `json:"message"`
	To      string    `json:"user_id"`
	Channel string    `json:"channel"`
}

type NotificationQueue struct {
	Publisher *rabbitmq.Publisher
	Consumer  *rabbitmq.Consumer
	cfg       *config.Config
}

func NewNotificationQueue(ch *rabbitmq.Channel, cfg *config.Config) (*NotificationQueue, error) {
	args := amqp091.Table{
		"x-delayed-type": "direct",
	}
	if err := ch.ExchangeDeclare(
		cfg.RabbitMQ.Exchange,
		"x-delayed-message",
		true,
		false,
		false,
		false,
		args,
	); err != nil {
		return nil, fmt.Errorf("failed to declare delayed exchange: %w", err)
	}

	qm := rabbitmq.NewQueueManager(ch)

	_, err := qm.DeclareQueue(cfg.RabbitMQ.DLQ, rabbitmq.QueueConfig{Durable: true})
	if err != nil {
		return nil, fmt.Errorf("failed to declare DLQ queue: %w", err)
	}

	retryArgs := map[string]interface{}{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": cfg.RabbitMQ.Queue,
		"x-message-ttl":             int32(5000),
	}

	_, err = qm.DeclareQueue(cfg.RabbitMQ.RetryQueue, rabbitmq.QueueConfig{
		Durable: true,
		Args:    retryArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to declare retry queue: %w", err)
	}

	mainArgs := map[string]interface{}{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": cfg.RabbitMQ.DLQ,
	}

	mainQ, err := qm.DeclareQueue(cfg.RabbitMQ.Queue, rabbitmq.QueueConfig{
		Durable: true,
		Args:    mainArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to declare main queue: %w", err)
	}

	if err := ch.QueueBind(mainQ.Name, cfg.RabbitMQ.RoutingKey, cfg.RabbitMQ.Exchange, false, nil); err != nil {
		return nil, fmt.Errorf("failed to bind the exchange to the main queue: %w", err)
	}

	pub := rabbitmq.NewPublisher(ch, cfg.RabbitMQ.Exchange)
	cons := rabbitmq.NewConsumer(ch, rabbitmq.NewConsumerConfig(mainQ.Name))

	return &NotificationQueue{Publisher: pub, Consumer: cons, cfg: cfg}, nil
}

func (q *NotificationQueue) Publish(msg NotificationMessage, strategy retry.Strategy) error {
	zlog.Logger.Printf("Publishing message %v", msg)
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	var delay time.Duration

	if !msg.SendAt.IsZero() {
		delay = time.Until(msg.SendAt)
		if delay < 0 {
			delay = 0
		}
	}

	zlog.Logger.Printf("delay %v", delay)

	headers := amqp091.Table{
		"x-delay": delay.Milliseconds(),
	}

	return q.Publisher.PublishWithRetry(
		body,
		q.cfg.RabbitMQ.RoutingKey,
		"application/json",
		strategy,
		rabbitmq.PublishingOptions{Headers: headers},
	)
}

func (q *NotificationQueue) Consume(ctx context.Context, out chan<- NotificationMessage, strategy retry.Strategy) error {
	msgChan := make(chan []byte)

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				zlog.Logger.Printf("Stopped consuming messages")
				return
			case m, ok := <-msgChan:
				if !ok {
					return
				}

				var msg NotificationMessage
				if err := json.Unmarshal(m, &msg); err != nil {
					zlog.Logger.Error().Err(err).Msg("failed to unmarshal message")
					continue
				}

				out <- msg
			}
		}
	}()

	return q.Consumer.ConsumeWithRetry(msgChan, strategy)
}
