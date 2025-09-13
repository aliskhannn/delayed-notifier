package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"
)

const (
	ExchangeName   = "notify-exchange"
	MainQueueName  = "notify-queue"
	RetryQueueName = "notify-retry"
	DLQName        = "notify-dlq"
	RoutingKey     = "notify"
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
}

func NewNotificationQueue(ch *rabbitmq.Channel) (*NotificationQueue, error) {
	exchange := rabbitmq.NewExchange(ExchangeName, "direct")
	if err := exchange.BindToChannel(ch); err != nil {
		return nil, fmt.Errorf("failed to bind to exchange: %w", err)
	}

	qm := rabbitmq.NewQueueManager(ch)

	_, err := qm.DeclareQueue(DLQName, rabbitmq.QueueConfig{Durable: true})
	if err != nil {
		return nil, fmt.Errorf("failed to declare DLQ queue: %w", err)
	}

	retryArgs := map[string]interface{}{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": MainQueueName,
		"x-message-ttl":             int32(5000),
	}

	_, err = qm.DeclareQueue(RetryQueueName, rabbitmq.QueueConfig{
		Durable: true,
		Args:    retryArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to declare retry queue: %w", err)
	}

	mainArgs := map[string]interface{}{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": DLQName,
	}

	mainQ, err := qm.DeclareQueue(MainQueueName, rabbitmq.QueueConfig{
		Durable: true,
		Args:    mainArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to declare main queue: %w", err)
	}

	if err := ch.QueueBind(mainQ.Name, RoutingKey, exchange.Name(), false, nil); err != nil {
		return nil, fmt.Errorf("failed to bind the exchange to the main queue: %w", err)
	}

	pub := rabbitmq.NewPublisher(ch, exchange.Name())
	cons := rabbitmq.NewConsumer(ch, rabbitmq.NewConsumerConfig(mainQ.Name))

	return &NotificationQueue{Publisher: pub, Consumer: cons}, nil
}

func (q *NotificationQueue) Publish(msg NotificationMessage, strategy retry.Strategy) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return q.Publisher.PublishWithRetry(body, RoutingKey, "application/json", strategy)
}

func (q *NotificationQueue) Consume(out chan<- NotificationMessage, strategy retry.Strategy) error {
	msgChan := make(chan []byte)

	go func() {
		for m := range msgChan {
			var msg NotificationMessage
			if err := json.Unmarshal(m, &msg); err != nil {
				zlog.Logger.Error().Err(err).Msg("failed to unmarshal message")
				continue
			}

			out <- msg
		}
	}()

	return q.Consumer.ConsumeWithRetry(msgChan, strategy)
}
