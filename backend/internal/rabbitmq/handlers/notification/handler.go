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

// notificationService defines the interface for sending notifications
// and updating their status.
type notificationService interface {
	Send(to, message, channel string) error
	SetStatus(ctx context.Context, strategy retry.Strategy, id uuid.UUID, status string) error
}

// Handler handles notifications from RabbitMQ and manages their lifecycle.
type Handler struct {
	service notificationService
}

// NewHandler creates a new Handler with the given notification service.
func NewHandler(svc notificationService) *Handler {
	return &Handler{
		service: svc,
	}
}

// HandleMessage processes a single notification message.
//
// It attempts to send the notification using the service. If sending fails,
// it marks the notification as "failed". If successful, it marks it as "sent".
func (h *Handler) HandleMessage(ctx context.Context, msg queue.NotificationMessage, strategy retry.Strategy) {
	zlog.Logger.Info().Msgf("Handle Message: Got notification %s, will be sent at %v", msg.ID, msg.SendAt)

	// Attempt to send the notification with retry strategy.
	err := retry.Do(func() error {
		select {
		case <-ctx.Done():
			// Stop if the context is canceled.
			return ctx.Err()
		default:
			// Send the notification
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
