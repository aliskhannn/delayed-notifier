package worker

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"

	"github.com/aliskhannn/delayed-notifier/internal/rabbitmq/queue"
)

// notificationConsumer defines an interface for consuming notification messages from a queue.
type notificationConsumer interface {
	Consume(ctx context.Context, out chan<- queue.NotificationMessage, strategy retry.Strategy) error
}

// messageHandler defines an interface for handling notification messages.
type messageHandler interface {
	HandleMessage(ctx context.Context, msg queue.NotificationMessage, strategy retry.Strategy)
}

// notificationService defines an interface for fetching notification status.
type notificationService interface {
	GetNotificationStatusByID(context.Context, retry.Strategy, uuid.UUID) (string, error)
}

// Notifier consumes messages from a queue and delegates handling to a messageHandler.
type Notifier struct {
	queue   notificationConsumer
	handler messageHandler
	service notificationService
}

// NewNotifier creates a new Notifier instance.
func NewNotifier(q notificationConsumer, h messageHandler, s notificationService) *Notifier {
	return &Notifier{
		queue:   q,
		handler: h,
		service: s,
	}
}

// Run starts the Notifier workers to process messages from the queue.
//
// It starts a consumer goroutine to read messages into a buffered channel.
// Then it starts workerCount goroutines that read messages from the channel,
// check the notification status, and pass valid messages to the handler.
//
// Messages with status "cancelled" are skipped.
func (n *Notifier) Run(ctx context.Context, strategy retry.Strategy, workerCount int) {
	var wg sync.WaitGroup
	msgChan := make(chan queue.NotificationMessage, workerCount*10)

	// Start consuming messages from the queue.
	go func() {
		if err := n.queue.Consume(ctx, msgChan, strategy); err != nil {
			zlog.Logger.Error().Err(err).Msg("failed to consume messages")
		}
	}()

	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func(id int) {
			defer wg.Done()

			zlog.Logger.Printf("worker-%d started", id)

			for {
				select {
				case <-ctx.Done():
					zlog.Logger.Printf("worker-%d shutting down", id)
					return
				case msg, ok := <-msgChan:
					if !ok {
						zlog.Logger.Printf("worker-%d channel closed, shutting down", id)
						return
					}

					zlog.Logger.Print("Getting notification status...")
					status, err := n.service.GetNotificationStatusByID(ctx, strategy, msg.ID)
					if err != nil {
						zlog.Logger.Printf("failed to get status for %s: %v", msg.ID, err)
						continue
					}

					zlog.Logger.Printf("Got notification status: %s", status)

					if status == "cancelled" {
						zlog.Logger.Printf("notification %s cancelled, skipping", msg.ID)
						continue
					}

					n.handler.HandleMessage(ctx, msg, strategy) // process the message
				}
			}
		}(i)
	}

	<-ctx.Done() // wait for shutdown signal
	wg.Wait()    // wait for all workers to finish
	zlog.Logger.Print("notifier stopped")
}
