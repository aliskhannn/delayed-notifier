package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"

	"github.com/aliskhannn/delayed-notifier/internal/api/respond"
	"github.com/aliskhannn/delayed-notifier/internal/config"
	"github.com/aliskhannn/delayed-notifier/internal/model"
	"github.com/aliskhannn/delayed-notifier/internal/repository/notification"
)

// notificationService defines the interface that the Handler depends on.
//
// It abstracts the business logic for creating, retrieving, updating,
// and managing the status of notifications.
//
//go:generate mockgen -source=handler.go -destination=../../../mocks/api/handlers/notification/mock.go -package=mocks
type notificationService interface {
	CreateNotification(context.Context, retry.Strategy, model.Notification) (uuid.UUID, error)
	GetNotificationStatusByID(context.Context, retry.Strategy, uuid.UUID) (string, error)
	SetStatus(ctx context.Context, strategy retry.Strategy, id uuid.UUID, status string) error
	GetAllNotifications(context.Context) ([]model.Notification, error)
}

// Handler handles HTTP requests related to notifications.
//
// It provides endpoints for creating notifications, checking their status,
// listing all notifications, and cancelling notifications.
type Handler struct {
	service   notificationService
	validator *validator.Validate
	cfg       *config.Config
}

// NewHandler creates a new Handler instance.
//
// Parameters:
//   - s: implementation of notifService
//   - v: validator instance for request validation
//   - cfg: configuration instance
func NewHandler(
	s notificationService,
	v *validator.Validate,
	cfg *config.Config,
) *Handler {
	return &Handler{service: s, validator: v, cfg: cfg}
}

// CreateRequest represents the JSON body expected in a notification creation request.
type CreateRequest struct {
	Message string `json:"message" validate:"required"`
	SendAt  string `json:"send_at" validate:"required"`
	Retries int    `json:"retries" validate:"required"`
	To      string `json:"to" validate:"required"`
	Channel string `json:"channel" validate:"required"`
}

// Create handles HTTP POST requests to create a new notification.
//
// It validates the request body, parses the send time, creates the notification
// using the service, and returns the created notification ID or an error.
func (h *Handler) Create(c *ginext.Context) {
	var req CreateRequest

	// Decode JSON request body into CreateRequest struct.
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		zlog.Logger.Error().Err(err).Msg("failed to decode request body")
		respond.Fail(c.Writer, http.StatusBadRequest, fmt.Errorf("invalid request body"))
		return
	}

	// Validate request fields using go-playground/validator.
	if err := h.validator.Struct(req); err != nil {
		zlog.Logger.Warn().Err(err).Msg("failed to validate request body")
		respond.Fail(c.Writer, http.StatusBadRequest, fmt.Errorf("validation error: %s", err.Error()))
		return
	}

	// Load Moscow timezone for parsing send_at field.
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		zlog.Logger.Fatal().Err(err).Msg("failed to load Moscow timezone")
	}

	// Parse the SendAt string into a time.Time object.
	parsedTime, err := time.ParseInLocation(time.DateTime, req.SendAt, loc)
	if err != nil {
		zlog.Logger.Error().Err(err).Msg("failed to parse send_at time")
		respond.Fail(c.Writer, http.StatusBadRequest, fmt.Errorf("invalid send_at format"))
		return
	}

	// Construct a Notification model.
	notif := model.Notification{
		Message: req.Message,
		SendAt:  parsedTime,
		Status:  "pending",
		Retries: req.Retries,
		To:      req.To,
		Channel: req.Channel,
	}

	// Create notification using the service layer.
	id, err := h.service.CreateNotification(c.Request.Context(), h.cfg.Retry, notif)
	if err != nil {
		zlog.Logger.Error().Err(err).Interface("message", notif.Message).Msg("failed to create notification")
		respond.Fail(c.Writer, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return
	}

	// Respond with created notification ID.
	respond.Created(c.Writer, id)
}

// GetStatus handles HTTP GET requests to retrieve the status of a notification.
//
// It expects the notification ID as a URL parameter and returns its status.
func (h *Handler) GetStatus(c *ginext.Context) {
	// Extract notification ID from URL parameters.
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		zlog.Logger.Error().Err(err).Interface("idStr", idStr).Msg("failed to parse id")
		respond.Fail(c.Writer, http.StatusBadRequest, fmt.Errorf("invalid id"))
		return
	}

	// Check for missing ID.
	if id == uuid.Nil {
		zlog.Logger.Warn().Msg("missing id")
		respond.Fail(c.Writer, http.StatusBadRequest, fmt.Errorf("missing id"))
		return
	}

	// Fetch notification status from service.
	status, err := h.service.GetNotificationStatusByID(c.Request.Context(), h.cfg.Retry, id)
	if err != nil {
		if errors.Is(err, notification.ErrNotificationNotFound) {
			zlog.Logger.Warn().Interface("id", id).Err(err).Msg("notification not found")
			respond.Fail(c.Writer, http.StatusNotFound, fmt.Errorf("notification not found"))
			return
		}

		// Internal server error.
		zlog.Logger.Error().Err(err).Interface("id", id).Msg("failed to get notification status")
		respond.Fail(c.Writer, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return
	}

	// Return notification status.
	respond.OK(c.Writer, status)
}

// GetAll handles HTTP GET requests to retrieve all notifications.
//
// It returns a list of all notifications or an error if retrieval fails.
func (h *Handler) GetAll(c *ginext.Context) {
	// Fetch all notifications from the service layer.
	notifications, err := h.service.GetAllNotifications(c.Request.Context())
	if err != nil {
		// Check if no notifications found and return 404.
		if errors.Is(err, notification.ErrNotificationNotFound) {
			zlog.Logger.Warn().Err(err).Msg("notification not found")
			respond.Fail(c.Writer, http.StatusNotFound, fmt.Errorf("notification not found"))
			return
		}

		// Log unexpected errors and return 500.
		zlog.Logger.Error().Err(err).Msg("failed to get notifications")
		respond.Fail(c.Writer, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return
	}

	// Respond with the list of notifications.
	respond.OK(c.Writer, notifications)
}

// Cancel handles HTTP POST or PUT requests to cancel a notification.
//
// It expects the notification ID as a URL parameter and updates its status
// to "cancelled". Returns success or error response.
func (h *Handler) Cancel(c *ginext.Context) {
	// Extract notification ID from URL parameters.
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		// Invalid UUID format.
		zlog.Logger.Error().Err(err).Interface("idStr", idStr).Msg("failed to parse id")
		respond.Fail(c.Writer, http.StatusBadRequest, fmt.Errorf("invalid id"))
		return
	}

	// Check for missing ID.
	if id == uuid.Nil {
		zlog.Logger.Warn().Msg("missing id")
		respond.Fail(c.Writer, http.StatusBadRequest, fmt.Errorf("missing id"))
		return
	}

	// Update the notification status to "cancelled".
	err = h.service.SetStatus(c.Request.Context(), h.cfg.Retry, id, "cancelled")
	if err != nil {
		// If notification does not exist, return 404.
		if errors.Is(err, notification.ErrNotificationNotFound) {
			zlog.Logger.Warn().Interface("id", id).Err(err).Msg("notification not found")
			respond.Fail(c.Writer, http.StatusNotFound, fmt.Errorf("notification not found"))
		}

		// Any other error is treated as internal server error.
		zlog.Logger.Error().Err(err).Interface("id", id).Msg("failed to cancel notification")
		respond.Fail(c.Writer, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return
	}

	// Return a success message.
	respond.OK(c.Writer, "notification cancelled")
}
