package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	wbfredis "github.com/wb-go/wbf/redis"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"

	"github.com/aliskhannn/delayed-notifier/internal/model"
)

type notifRepo interface {
	CreateNotification(context.Context, model.Notification) (uuid.UUID, error)
	GetNotificationStatusByID(context.Context, uuid.UUID) (string, error)
	DeleteNotification(context.Context, uuid.UUID) error
}

type Service struct {
	repo  notifRepo
	cache wbfredis.Client
}

func NewService(repo notifRepo, cache wbfredis.Client) *Service {
	return &Service{repo: repo, cache: cache}
}

func (s *Service) CreateNotification(ctx context.Context, strategy retry.Strategy, notification model.Notification) (uuid.UUID, error) {
	id, err := s.repo.CreateNotification(ctx, notification)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create notification: %w", err)
	}

	err = s.cache.SetWithRetry(ctx, strategy, id.String(), notification.Status)
	if err != nil {
		zlog.Logger.Printf("failed to cache notification %s: %v", id, err)
	}

	return id, nil
}

func (s *Service) GetNotificationStatusByID(ctx context.Context, strategy retry.Strategy, id uuid.UUID) (string, error) {
	//strategy := retry.Strategy{
	//	Attempts: 3,
	//	Delay:    50 * time.Millisecond,
	//	Backoff:  2,
	//}

	status, err := s.cache.GetWithRetry(ctx, strategy, id.String())
	if err != nil && !errors.Is(err, redis.Nil) {
		zlog.Logger.Printf("failed to get notification status from cache %s: %v", id, err)
	}

	if errors.Is(err, redis.Nil) {
		status, err = s.repo.GetNotificationStatusByID(ctx, id)
		if err != nil {
			return "", fmt.Errorf("get notification status: %w", err)
		}

		err = s.cache.SetWithRetry(ctx, strategy, id.String(), status)
		if err != nil {
			zlog.Logger.Printf("failed to cache notification %s: %v", id, err)
		}
	}

	return status, nil
}

func (s *Service) DeleteNotification(ctx context.Context, id uuid.UUID) error {
	err := s.repo.DeleteNotification(ctx, id)
	if err != nil {
		return fmt.Errorf("delete notification: %w", err)
	}

	return nil
}
