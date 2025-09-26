package notification

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/wb-go/wbf/retry"

	mocks "github.com/aliskhannn/delayed-notifier/internal/mocks/service/notification"
	"github.com/aliskhannn/delayed-notifier/internal/model"
)

func TestService_CreateNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMocknotificationRepository(ctrl)
	queueMock := mocks.NewMocknotificationPublisher(ctrl)
	cacheMock := mocks.NewMockcache(ctrl)

	svc := NewService(repoMock, queueMock, map[string]Notifier{}, cacheMock)

	notificationID := uuid.New()
	n := model.Notification{
		Message: "Hello",
		SendAt:  time.Now(),
		To:      "user@example.com",
		Retries: 3,
		Channel: "email",
		Status:  "pending",
	}
	strategy := retry.Strategy{}

	repoMock.EXPECT().CreateNotification(gomock.Any(), n).Return(notificationID, nil)
	cacheMock.EXPECT().SetWithRetry(gomock.Any(), strategy, notificationID.String(), n.Status).Return(nil)
	queueMock.EXPECT().Publish(gomock.Any(), strategy).Return(nil)

	id, err := svc.CreateNotification(context.Background(), strategy, n)
	assert.NoError(t, err)
	assert.Equal(t, notificationID, id)
}

func TestService_GetNotificationStatusByID_CacheHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cacheMock := mocks.NewMockcache(ctrl)
	svc := NewService(nil, nil, nil, cacheMock)

	id := uuid.New()
	strategy := retry.Strategy{}

	cacheMock.EXPECT().GetWithRetry(gomock.Any(), strategy, id.String()).Return("pending", nil)

	status, err := svc.GetNotificationStatusByID(context.Background(), strategy, id)
	assert.NoError(t, err)
	assert.Equal(t, "pending", status)
}

func TestService_GetNotificationStatusByID_CacheMiss(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMocknotificationRepository(ctrl)
	cacheMock := mocks.NewMockcache(ctrl)
	svc := NewService(repoMock, nil, nil, cacheMock)

	id := uuid.New()
	strategy := retry.Strategy{}

	cacheMock.EXPECT().GetWithRetry(gomock.Any(), strategy, id.String()).Return("", redis.Nil)
	repoMock.EXPECT().GetNotificationStatusByID(gomock.Any(), id).Return("sent", nil)
	cacheMock.EXPECT().SetWithRetry(gomock.Any(), strategy, id.String(), "sent").Return(nil)

	status, err := svc.GetNotificationStatusByID(context.Background(), strategy, id)
	assert.NoError(t, err)
	assert.Equal(t, "sent", status)
}

func TestService_SetStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMocknotificationRepository(ctrl)
	cacheMock := mocks.NewMockcache(ctrl)
	svc := NewService(repoMock, nil, nil, cacheMock)

	id := uuid.New()
	strategy := retry.Strategy{}

	repoMock.EXPECT().UpdateStatus(gomock.Any(), id, "sent").Return(nil)
	cacheMock.EXPECT().SetWithRetry(gomock.Any(), strategy, id.String(), "sent").Return(nil)

	err := svc.SetStatus(context.Background(), strategy, id, "sent")
	assert.NoError(t, err)
}

func TestService_Send_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifierMock := mocks.NewMockNotifier(ctrl)
	notifiers := map[string]Notifier{"email": notifierMock}
	svc := NewService(nil, nil, notifiers, nil)

	notifierMock.EXPECT().Send("user@example.com", "Hello").Return(nil)

	err := svc.Send("user@example.com", "Hello", "email")
	assert.NoError(t, err)
}

func TestService_Send_UnknownChannel(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)
	err := svc.Send("user@example.com", "Hello", "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown channel")
}

func TestService_GetAllNotifications(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoMock := mocks.NewMocknotificationRepository(ctrl)
	svc := NewService(repoMock, nil, nil, nil)

	notifications := []model.Notification{
		{ID: uuid.New(), Message: "test1"},
		{ID: uuid.New(), Message: "test2"},
	}

	repoMock.EXPECT().GetAllNotifications(gomock.Any()).Return(notifications, nil)

	result, err := svc.GetAllNotifications(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, notifications, result)
}
