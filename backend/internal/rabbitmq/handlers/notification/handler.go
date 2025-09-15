package notification

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"

	"github.com/aliskhannn/delayed-notifier/internal/rabbitmq/queue"
	"github.com/aliskhannn/delayed-notifier/internal/repository/notification"
)

//go:generate mockgen -source=handler.go -destination=../../../mocks/rabbitmq/handlers/notification/mock.go -package=mocks
type notificationService interface {
	Send(to, message, channel string) error
	SetStatus(ctx context.Context, strategy retry.Strategy, id uuid.UUID, status string) error
}

type Handler struct {
	service notificationService
}

func NewHandler(svc notificationService) *Handler {
	return &Handler{
		service: svc,
	}
}

func (h *Handler) HandleMessage(ctx context.Context, msg queue.NotificationMessage, strategy retry.Strategy) {
	zlog.Logger.Info().Msgf("Handle Message: Got notification %s, will be sent at %v", msg.ID, msg.SendAt)

	err := retry.Do(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			zlog.Logger.Printf("Handle Message: Sending notification %s via %s", msg.ID, msg.Channel)
			return h.service.Send(msg.To, msg.Message, msg.Channel)
		}
	}, strategy)

	if err != nil {
		zlog.Logger.Printf("Handle Message: Notification %s failed, moving to DLQ: %v", msg.ID, err)
		if setErr := h.service.SetStatus(ctx, strategy, msg.ID, "failed"); setErr != nil {
			if errors.Is(setErr, notification.ErrNotificationNotFound) {
				zlog.Logger.Warn().Interface("id", msg.ID).Err(err).Msg("notification not found")
			}

			zlog.Logger.Error().Err(setErr).Msgf("failed to set status=failed for %s", msg.ID)
		}
		return
	}

	zlog.Logger.Info().Msgf("Handle Message: Notification %s sent successfully", msg.ID)
	if setErr := h.service.SetStatus(ctx, strategy, msg.ID, "sent"); setErr != nil {
		if errors.Is(setErr, notification.ErrNotificationNotFound) {
			zlog.Logger.Warn().Interface("id", msg.ID).Err(err).Msg("notification not found")
		}

		zlog.Logger.Error().Err(setErr).Msgf("failed to set status=sent for %s", msg.ID)
	}
}
