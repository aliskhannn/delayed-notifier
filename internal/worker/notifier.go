package worker

import (
	"context"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"

	"github.com/aliskhannn/delayed-notifier/internal/rabbitmq/queue"
)

type notifQueue interface {
	Consume(out chan<- queue.NotificationMessage, strategy retry.Strategy) error
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
	msgChan := make(chan queue.NotificationMessage)

	go func() {
		if err := n.queue.Consume(msgChan, strategy); err != nil {
			zlog.Logger.Fatal().Err(err).Msg("failed to consume messages")
		}
	}()

	for i := 0; i < workerCount; i++ {
		go func(id int) {
			zlog.Logger.Printf("worker-%d started", id)

			for {
				select {
				case <-ctx.Done():
					zlog.Logger.Printf("worker-%d shutting down", id)
					return
				case msg := <-msgChan:
					status, err := n.service.GetNotificationStatusByID(ctx, strategy, msg.ID)
					if err != nil {
						zlog.Logger.Printf("failed to get status for %s: %v", msg.ID, err)
						continue
					}

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
	zlog.Logger.Print("notifier stopped")
}
