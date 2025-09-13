package notification

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"

	"github.com/aliskhannn/delayed-notifier/internal/rabbitmq/queue"
)

type notifService interface {
	Send(to, message, channel string) error
	SetStatus(ctx context.Context, id uuid.UUID, status string) error
}

type Handler struct {
	service notifService
	queue   *queue.NotificationQueue
}

func NewHandler(svc notifService) *Handler {
	return &Handler{
		service: svc,
	}
}

func (h *Handler) HandleMessage(ctx context.Context, msg queue.NotificationMessage, strategy retry.Strategy) {
	delay := time.Until(msg.SendAt)
	if delay > 0 {
		time.Sleep(delay)
	}

	attempt := 0
	currentDelay := strategy.Delay

	for attempt < strategy.Attempts {
		err := h.service.Send(msg.To, msg.Message, msg.Channel)
		if err == nil {
			zlog.Logger.Printf("notification %s sent", msg.ID)
			_ = h.service.SetStatus(ctx, msg.ID, "sent")
			return
		}

		attempt++
		zlog.Logger.Printf("failed to send notification %s: %v, retry %d/%d",
			msg.ID, err, attempt, strategy.Attempts,
		)

		time.Sleep(currentDelay)
		currentDelay = time.Duration(float64(currentDelay) * strategy.Backoff)
	}

	zlog.Logger.Printf("Notification %s failed after %d attempts, moving to DLQ", msg.ID, attempt)
	_ = h.service.SetStatus(ctx, msg.ID, "failed")
}
