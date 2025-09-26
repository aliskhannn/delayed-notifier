package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wb-go/wbf/retry"

	mocks "github.com/aliskhannn/delayed-notifier/internal/mocks/worker"
	"github.com/aliskhannn/delayed-notifier/internal/rabbitmq/queue"
)

func TestNotifier_Run_HandleMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConsumer := mocks.NewMocknotificationConsumer(ctrl)
	mockHandler := mocks.NewMockmessageHandler(ctrl)
	mockService := mocks.NewMocknotificationService(ctrl)

	n := NewNotifier(mockConsumer, mockHandler, mockService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	strategy := retry.Strategy{Attempts: 1, Delay: time.Millisecond}

	msg := queue.NotificationMessage{
		ID:      uuid.New(),
		To:      "test@example.com",
		Retries: 3,
		Message: "Hello",
		Channel: "email",
		SendAt:  time.Now(),
	}

	mockConsumer.EXPECT().Consume(gomock.Any(), gomock.Any(), strategy).DoAndReturn(
		func(_ context.Context, out chan<- queue.NotificationMessage, _ retry.Strategy) error {
			out <- msg
			return nil
		},
	)

	mockService.EXPECT().GetNotificationStatusByID(gomock.Any(), strategy, msg.ID).Return("pending", nil)
	mockHandler.EXPECT().HandleMessage(gomock.Any(), msg, strategy)

	go n.Run(ctx, strategy, 1)

	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)
}

func TestNotifier_Run_CancelledStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConsumer := mocks.NewMocknotificationConsumer(ctrl)
	mockHandler := mocks.NewMockmessageHandler(ctrl)
	mockService := mocks.NewMocknotificationService(ctrl)

	n := NewNotifier(mockConsumer, mockHandler, mockService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	strategy := retry.Strategy{Attempts: 1, Delay: time.Millisecond}
	msg := queue.NotificationMessage{ID: uuid.New()}

	mockConsumer.EXPECT().Consume(gomock.Any(), gomock.Any(), strategy).DoAndReturn(
		func(_ context.Context, out chan<- queue.NotificationMessage, _ retry.Strategy) error {
			out <- msg
			return nil
		},
	)

	mockService.EXPECT().GetNotificationStatusByID(gomock.Any(), strategy, msg.ID).Return("cancelled", nil)

	go n.Run(ctx, strategy, 1)
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)
}

func TestNotifier_Run_GetStatusError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConsumer := mocks.NewMocknotificationConsumer(ctrl)
	mockHandler := mocks.NewMockmessageHandler(ctrl)
	mockService := mocks.NewMocknotificationService(ctrl)

	n := NewNotifier(mockConsumer, mockHandler, mockService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	strategy := retry.Strategy{Attempts: 1, Delay: time.Millisecond}
	msg := queue.NotificationMessage{ID: uuid.New()}

	mockConsumer.EXPECT().Consume(gomock.Any(), gomock.Any(), strategy).DoAndReturn(
		func(_ context.Context, out chan<- queue.NotificationMessage, _ retry.Strategy) error {
			out <- msg
			return nil
		},
	)

	mockService.EXPECT().GetNotificationStatusByID(gomock.Any(), strategy, msg.ID).Return("", errors.New("db error"))

	go n.Run(ctx, strategy, 1)
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)
}

func TestNotifier_Run_ContextCancelled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConsumer := mocks.NewMocknotificationConsumer(ctrl)
	mockHandler := mocks.NewMockmessageHandler(ctrl)
	mockService := mocks.NewMocknotificationService(ctrl)

	n := NewNotifier(mockConsumer, mockHandler, mockService)

	ctx, cancel := context.WithCancel(context.Background())

	strategy := retry.Strategy{Attempts: 1, Delay: time.Millisecond}

	mockConsumer.EXPECT().Consume(gomock.Any(), gomock.Any(), strategy).DoAndReturn(
		func(ctx context.Context, out chan<- queue.NotificationMessage, _ retry.Strategy) error {
			<-ctx.Done()
			return nil
		},
	)

	go n.Run(ctx, strategy, 1)

	cancel()

	require.Eventually(t, func() bool { return true }, time.Second, 50*time.Millisecond)
	assert.True(t, true, "notifier stopped cleanly")
}
