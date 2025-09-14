package worker

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"

	"github.com/aliskhannn/delayed-notifier/internal/rabbitmq/queue"
)

type notifQueue interface {
	Consume(ctx context.Context, out chan<- queue.NotificationMessage, strategy retry.Strategy) error
}

type messageHandler interface {
	HandleMessage(ctx context.Context, msg queue.NotificationMessage, strategy retry.Strategy)
}

type notifService interface {
	GetNotificationStatusByID(context.Context, retry.Strategy, uuid.UUID) (string, error)
}

type Notifier struct {
	queue   notifQueue
	handler messageHandler
	service notifService
}

func NewNotifier(q notifQueue, h messageHandler, s notifService) *Notifier {
	return &Notifier{
		queue:   q,
		handler: h,
		service: s,
	}
}

func (n *Notifier) Run(ctx context.Context, strategy retry.Strategy, workerCount int) {
	var wg sync.WaitGroup
	msgChan := make(chan queue.NotificationMessage, workerCount*10)

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

					n.handler.HandleMessage(ctx, msg, strategy)
				}
			}
		}(i)
	}

	<-ctx.Done()
	wg.Wait()
	zlog.Logger.Print("notifier stopped")
}
