package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/wb-go/wbf/retry"

	mocks "github.com/aliskhannn/delayed-notifier/internal/mocks/rabbitmq/handlers/notification"
	"github.com/aliskhannn/delayed-notifier/internal/rabbitmq/queue"
	"github.com/aliskhannn/delayed-notifier/internal/repository/notification"
)

func TestHandler_HandleMessage_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMocknotificationService(ctrl)
	h := NewHandler(mockService)

	msg := queue.NotificationMessage{
		ID:      uuid.New(),
		To:      "test@example.com",
		Message: "Hello",
		Channel: "email",
		SendAt:  time.Now(),
	}

	strategy := retry.Strategy{Attempts: 1, Delay: time.Millisecond}

	mockService.EXPECT().
		Send(msg.To, msg.Message, msg.Channel).
		Return(nil)
	mockService.EXPECT().
		SetStatus(gomock.Any(), strategy, msg.ID, "sent").
		Return(nil)

	h.HandleMessage(context.Background(), msg, strategy)
}

func TestHandler_HandleMessage_SendFailsThenSetFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMocknotificationService(ctrl)
	h := NewHandler(mockService)

	msg := queue.NotificationMessage{
		ID:      uuid.New(),
		To:      "test@example.com",
		Message: "Hello",
		Channel: "email",
		SendAt:  time.Now(),
	}

	strategy := retry.Strategy{Attempts: 1, Delay: time.Millisecond}
	sendErr := errors.New("send error")

	mockService.EXPECT().
		Send(msg.To, msg.Message, msg.Channel).
		Return(sendErr)
	mockService.EXPECT().
		SetStatus(gomock.Any(), strategy, msg.ID, "failed").
		Return(nil)

	h.HandleMessage(context.Background(), msg, strategy)
}

func TestHandler_HandleMessage_SendFailsThenSetFailedNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMocknotificationService(ctrl)
	h := NewHandler(mockService)

	msg := queue.NotificationMessage{
		ID:      uuid.New(),
		To:      "test@example.com",
		Message: "Hello",
		Channel: "email",
		SendAt:  time.Now(),
	}

	strategy := retry.Strategy{Attempts: 1, Delay: time.Millisecond}
	sendErr := errors.New("send error")

	mockService.EXPECT().
		Send(msg.To, msg.Message, msg.Channel).
		Return(sendErr)
	mockService.EXPECT().
		SetStatus(gomock.Any(), strategy, msg.ID, "failed").
		Return(notification.ErrNotificationNotFound)

	h.HandleMessage(context.Background(), msg, strategy)
}

func TestHandler_HandleMessage_SetStatusSentFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMocknotificationService(ctrl)
	h := NewHandler(mockService)

	msg := queue.NotificationMessage{
		ID:      uuid.New(),
		To:      "test@example.com",
		Message: "Hello",
		Channel: "email",
		SendAt:  time.Now(),
	}

	strategy := retry.Strategy{Attempts: 1, Delay: time.Millisecond}

	mockService.EXPECT().
		Send(msg.To, msg.Message, msg.Channel).
		Return(nil)
	mockService.EXPECT().
		SetStatus(gomock.Any(), strategy, msg.ID, "sent").
		Return(errors.New("set status error"))

	h.HandleMessage(context.Background(), msg, strategy)
}

func TestHandler_HandleMessage_ContextCanceled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMocknotificationService(ctrl)
	h := NewHandler(mockService)

	msg := queue.NotificationMessage{
		ID:      uuid.New(),
		To:      "test@example.com",
		Message: "Hello",
		Channel: "email",
		SendAt:  time.Now(),
	}

	strategy := retry.Strategy{Attempts: 1, Delay: time.Millisecond}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	mockService.EXPECT().
		SetStatus(ctx, strategy, msg.ID, "failed").
		Return(nil)

	h.HandleMessage(ctx, msg, strategy)
}
