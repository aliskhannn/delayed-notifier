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

//go:generate mockgen -source=service.go -destination=../../mocks/service/notification/mock.go -package=mocks

type notificationPublisher interface {
	Publish(msg queue.NotificationMessage, strategy retry.Strategy) error
}

type notificationRepository interface {
	CreateNotification(context.Context, model.Notification) (uuid.UUID, error)
	GetNotificationStatusByID(context.Context, uuid.UUID) (string, error)
	UpdateStatus(context.Context, uuid.UUID, string) error
	GetAllNotifications(context.Context) ([]model.Notification, error)
}

type Notifier interface {
	Send(to string, msg string) error
}

type cache interface {
	SetWithRetry(ctx context.Context, strategy retry.Strategy, key string, value interface{}) error
	GetWithRetry(ctx context.Context, strategy retry.Strategy, key string) (string, error)
}

type Service struct {
	repo      notificationRepository
	queue     notificationPublisher
	notifiers map[string]Notifier
	cache     cache
}

func NewService(
	repo notificationRepository,
	queue notificationPublisher,
	notifiers map[string]Notifier,
	cache cache,
) *Service {
	return &Service{repo: repo, queue: queue, notifiers: notifiers, cache: cache}
}

func (s *Service) CreateNotification(ctx context.Context, strategy retry.Strategy, notification model.Notification) (uuid.UUID, error) {
	id, err := s.repo.CreateNotification(ctx, notification)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create notification: %w", err)
	}

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

	err = s.queue.Publish(msg, strategy)
	if err != nil {
		zlog.Logger.Error().Err(err).Str("id", id.String()).Msg("failed to publish notification")
	}

	return id, nil
}

func (s *Service) GetNotificationStatusByID(ctx context.Context, strategy retry.Strategy, id uuid.UUID) (string, error) {
	status, err := s.cache.GetWithRetry(ctx, strategy, id.String())
	if err != nil && !errors.Is(err, redis.Nil) {
		zlog.Logger.Error().Err(err).Str("id", id.String()).Msg("failed to get notification status from cache")
	}

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

func (s *Service) GetAllNotifications(ctx context.Context) ([]model.Notification, error) {
	notifications, err := s.repo.GetAllNotifications(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all notifications: %w", err)
	}

	return notifications, nil
}

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
