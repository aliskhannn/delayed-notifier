package notification

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/wb-go/wbf/retry"

	"github.com/aliskhannn/delayed-notifier/internal/config"
	"github.com/aliskhannn/delayed-notifier/internal/mocks/api/handlers/notification"
	"github.com/aliskhannn/delayed-notifier/internal/model"
)

func setupHandler(t *testing.T) (*Handler, *mocks.MocknotificationService, *config.Config) {
	ctrl := gomock.NewController(t)
	mockService := mocks.NewMocknotificationService(ctrl)
	cfg := &config.Config{Retry: retry.Strategy{}}
	validate := validator.New()
	handler := NewHandler(mockService, validate, cfg)
	return handler, mockService, cfg
}

func TestHandler_Create_Success(t *testing.T) {
	handler, mockService, cfg := setupHandler(t)

	reqBody := CreateRequest{
		Message: "Hello",
		SendAt:  "2025-09-15 10:00:00",
		Retries: 3,
		To:      "test@example.com",
		Channel: "email",
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/notifications", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	mockService.EXPECT().
		CreateNotification(
			gomock.Any(),
			cfg.Retry,
			gomock.AssignableToTypeOf(model.Notification{}),
		).Return(uuid.New(), nil)

	handler.Create(c)

	assert.Equal(t, http.StatusCreated, w.Result().StatusCode)
}

func TestHandler_GetStatus_Success(t *testing.T) {
	handler, mockService, cfg := setupHandler(t)
	id := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/notifications/"+id.String(), nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	mockService.EXPECT().
		GetNotificationStatusByID(gomock.Any(), cfg.Retry, id).
		Return("pending", nil)

	handler.GetStatus(c)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestHandler_GetAll_Success(t *testing.T) {
	handler, mockService, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	mockService.EXPECT().
		GetAllNotifications(gomock.Any()).
		Return([]model.Notification{{Message: "msg"}}, nil)

	handler.GetAll(c)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestHandler_Cancel_Success(t *testing.T) {
	handler, mockService, cfg := setupHandler(t)
	id := uuid.New()

	req := httptest.NewRequest(http.MethodPost, "/notifications/"+id.String()+"/cancel", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = gin.Params{{Key: "id", Value: id.String()}}

	mockService.EXPECT().
		SetStatus(gomock.Any(), cfg.Retry, id, "cancelled").
		Return(nil)

	handler.Cancel(c)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}
