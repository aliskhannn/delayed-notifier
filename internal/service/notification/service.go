package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"

	"github.com/aliskhannn/delayed-notifier/internal/model"
	"github.com/aliskhannn/delayed-notifier/internal/rabbitmq/queue"
)

// notificationPublisher defines the interface for publishing notification messages.
type notificationPublisher interface {
	Publish(msg queue.NotificationMessage, strategy retry.Strategy) error
}

// notificationRepository defines the interface for notification persistence operations.
type notificationRepository interface {
	CreateNotification(context.Context, model.Notification) (uuid.UUID, error)
	GetNotificationStatusByID(context.Context, uuid.UUID) (string, error)
	UpdateStatus(context.Context, uuid.UUID, string) error
	GetAllNotifications(context.Context) ([]model.Notification, error)
}

// Notifier defines an interface for sending notifications through a channel.
type Notifier interface {
	Send(to string, msg string) error
}

// cache defines the interface for caching notification statuses.
type cache interface {
	SetWithRetry(ctx context.Context, strategy retry.Strategy, key string, value interface{}) error
	GetWithRetry(ctx context.Context, strategy retry.Strategy, key string) (string, error)
}

// The Service provides methods for creating, retrieving, sending, and updating notifications.
type Service struct {
	repo      notificationRepository
	queue     notificationPublisher
	notifiers map[string]Notifier
	cache     cache
}

// NewService creates a new Service instance with repository, publisher, notifiers, and cache.
func NewService(
	repo notificationRepository,
	queue notificationPublisher,
	notifiers map[string]Notifier,
	cache cache,
) *Service {
	return &Service{repo: repo, queue: queue, notifiers: notifiers, cache: cache}
}


// CreateNotification creates a new notification, caches its status, and publishes it to the queue.
func (s *Service) CreateNotification(ctx context.Context, strategy retry.Strategy, notification model.Notification) (uuid.UUID, error) {
	id, err := s.repo.CreateNotification(ctx, notification)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create notification: %w", err)
	}

	// Cache initial status.
	err = s.cache.SetWithRetry(ctx, strategy, id.String(), notification.Status)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("id", id.String()).Msg("failed to cache notification")
	}

	msg := queue.NotificationMessage{
		ID:      id,
		SendAt:  notification.SendAt,
		Message: notification.Message,
		To:      notification.To,
		Retries: notification.Retries,
		Channel: notification.Channel,
	}

	// Publish a message.
	err = s.queue.Publish(msg, strategy)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("id", id.String()).Msg("failed to publish notification")
	}

	return id, nil
}

// GetNotificationStatusByID retrieves the status of a notification.
// It first tries to get the value from cache, falls back to repository if cache misses.
func (s *Service) GetNotificationStatusByID(ctx context.Context, strategy retry.Strategy, id uuid.UUID) (string, error) {
	status, err := s.cache.GetWithRetry(ctx, strategy, id.String())
	if err != nil && !errors.Is(err, redis.Nil) {
		zlog.Logger.Error().Err(err).Str("id", id.String()).Msg("failed to get notification status from cache")
	}

	// If cache misses, fetch from repo and update cache.
	if errors.Is(err, redis.Nil) {
		status, err = s.repo.GetNotificationStatusByID(ctx, id)
		if err != nil {
			return "", fmt.Errorf("get notification status: %w", err)
		}

		err = s.cache.SetWithRetry(ctx, strategy, id.String(), status)
		if err != nil {
			zlog.Logger.Error().Err(err).Str("id", id.String()).Msg("failed to cache notification")
		}
	}

	return status, nil
}

// GetAllNotifications returns all notifications from the repository.
func (s *Service) GetAllNotifications(ctx context.Context) ([]model.Notification, error) {
	notifications, err := s.repo.GetAllNotifications(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all notifications: %w", err)
	}

	return notifications, nil
}

// Send sends a notification through the appropriate channel (email, telegram, etc.).
func (s *Service) Send(to, message, channel string) error {
	notifier, ok := s.notifiers[channel]
	if !ok {
		return fmt.Errorf("unknown channel %s", channel)
	}

	err := notifier.Send(to, message)
	if err != nil {
		return fmt.Errorf("send notification: %w", err)
	}

	return nil
}

// SetStatus updates the notification status in the repository and updates the cache.
func (s *Service) SetStatus(ctx context.Context, strategy retry.Strategy, id uuid.UUID, status string) error {
	err := s.repo.UpdateStatus(ctx, id, status)
	if err != nil {
		return fmt.Errorf("update notification status: %w", err)
	}

	err = s.cache.SetWithRetry(ctx, strategy, id.String(), status)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("id", id.String()).Msg("failed to cache notification")
	}

	return nil
}
