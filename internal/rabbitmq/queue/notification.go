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

// NotificationMessage represents a single notification message
// that can be published or consumed from RabbitMQ.
type NotificationMessage struct {
	ID      uuid.UUID `json:"id"`      // unique identifier
	SendAt  time.Time `json:"send_at"` // time to send the notification
	Message string    `json:"message"` // message content
	To      string    `json:"user_id"` // recipient identifier
	Retries int       `json:"retries"` // number of retry attempts
	Channel string    `json:"channel"` // notification channel (email, telegram, etc.)
}

// NotificationQueue wraps RabbitMQ publisher and consumer
// for publishing and consuming notifications.
type NotificationQueue struct {
	Publisher *rabbitmq.Publisher // rabbitmq publisher
	Consumer  *rabbitmq.Consumer  // rabbitmq consumer
	cfg       *config.Config      // application configuration
}

// NewNotificationQueue creates a new NotificationQueue.
//
// It declares the main queue, DLQ, and retry queue, sets up the delayed exchange,
// and returns a NotificationQueue instance.
func NewNotificationQueue(ch *rabbitmq.Channel, cfg *config.Config) (*NotificationQueue, error) {
	args := amqp091.Table{
		"x-delayed-type": "direct", // enable a delayed message type
	}

	// Declare the delayed exchange.
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

	// Declare DLQ.
	_, err := qm.DeclareQueue(cfg.RabbitMQ.DLQ, rabbitmq.QueueConfig{Durable: true})
	if err != nil {
		return nil, fmt.Errorf("failed to declare DLQ queue: %w", err)
	}

	// Declare retry queue with dead-letter pointing to the main queue.
	retryArgs := map[string]interface{}{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": cfg.RabbitMQ.Queue,
		"x-message-ttl":             int32(5000), // 5s delay
	}

	_, err = qm.DeclareQueue(cfg.RabbitMQ.RetryQueue, rabbitmq.QueueConfig{
		Durable: true,
		Args:    retryArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to declare retry queue: %w", err)
	}

	// Declare the main queue with dead-letter pointing to DLQ.
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

	// Bind main queue to delayed exchange.
	if err := ch.QueueBind(mainQ.Name, cfg.RabbitMQ.RoutingKey, cfg.RabbitMQ.Exchange, false, nil); err != nil {
		return nil, fmt.Errorf("failed to bind the exchange to the main queue: %w", err)
	}

	pub := rabbitmq.NewPublisher(ch, cfg.RabbitMQ.Exchange)
	cons := rabbitmq.NewConsumer(ch, rabbitmq.NewConsumerConfig(mainQ.Name))

	return &NotificationQueue{Publisher: pub, Consumer: cons, cfg: cfg}, nil
}

// Publish sends a notification message to RabbitMQ with optional delay.
//
// Delay is calculated based on msg.SendAt and is applied using the x-delay header.
func (q *NotificationQueue) Publish(msg NotificationMessage, strategy retry.Strategy) error {
	zlog.Logger.Printf("Publishing message %v", msg)

	// Marshal the message to JSON.
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Calculate delay until the message should be sent.
	var delay time.Duration
	if !msg.SendAt.IsZero() {
		delay = time.Until(msg.SendAt)
		if delay < 0 {
			delay = 0
		}
	}

	zlog.Logger.Printf("delay %v", delay)

	// Set RabbitMQ headers for delayed publishing.
	headers := amqp091.Table{
		"x-delay": delay.Milliseconds(),
	}

	// Publish the message with retry strategy.
	return q.Publisher.PublishWithRetry(
		body,
		q.cfg.RabbitMQ.RoutingKey,
		"application/json",
		strategy,
		rabbitmq.PublishingOptions{Headers: headers},
	)
}

// Consume receives messages from RabbitMQ, unmarshals them, and sends to the output channel.
// It runs a goroutine to process messages and stops when the context is done.
func (q *NotificationQueue) Consume(ctx context.Context, out chan<- NotificationMessage, strategy retry.Strategy) error {
	msgChan := make(chan []byte) // internal channel to receive raw messages

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				zlog.Logger.Printf("Stopped consuming messages")
				return
			case m, ok := <-msgChan:
				if !ok {
					return // exit if internal channel is closed
				}

				var msg NotificationMessage
				if err := json.Unmarshal(m, &msg); err != nil {
					zlog.Logger.Error().Err(err).Msg("failed to unmarshal message")
					continue
				}

				out <- msg // send processed message to output channel
			}
		}
	}()

	// Start consuming messages from RabbitMQ with retry.
	return q.Consumer.ConsumeWithRetry(msgChan, strategy)
}
