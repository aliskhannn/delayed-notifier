package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/retry"
	"github.com/wb-go/wbf/zlog"

	"github.com/aliskhannn/delayed-notifier/internal/api/dto"
	"github.com/aliskhannn/delayed-notifier/internal/api/respond"
	"github.com/aliskhannn/delayed-notifier/internal/config"
	"github.com/aliskhannn/delayed-notifier/internal/model"
	"github.com/aliskhannn/delayed-notifier/internal/repository/notification"
)

type notifService interface {
	CreateNotification(context.Context, retry.Strategy, model.Notification) (uuid.UUID, error)
	GetNotificationStatusByID(context.Context, retry.Strategy, uuid.UUID) (string, error)
	DeleteNotification(context.Context, uuid.UUID) error
}

type Handler struct {
	service   notifService
	validator *validator.Validate
	cfg       config.Config
}

func NewHandler(s notifService, v *validator.Validate) *Handler {
	return &Handler{service: s, validator: v}
}

func (h *Handler) Create(c *ginext.Context) {
	var req dto.CreateRequest

	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		zlog.Logger.Error().Err(err).Msg("failed to decode request body")
		respond.Fail(c.Writer, http.StatusBadRequest, fmt.Errorf("invalid request body"))
		return
	}

	if err := h.validator.Struct(req); err != nil {
		zlog.Logger.Warn().Err(err).Msg("failed to validate request body")
		respond.Fail(c.Writer, http.StatusBadRequest, fmt.Errorf("validation error: %s", err.Error()))
		return
	}

	notif := model.Notification{
		Message: req.Message,
		SendAt:  req.SendAt,
		Retries: req.Retries,
	}

	id, err := h.service.CreateNotification(c, h.cfg.Retry, notif)
	if err != nil {
		zlog.Logger.Error().Err(err).Interface("message", notif.Message).Msg("failed to create notification")
		respond.Fail(c.Writer, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return
	}

	respond.Created(c.Writer, id)
}

func (h *Handler) GetStatus(c *ginext.Context) {
	idStr := chi.URLParam(c.Request, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		zlog.Logger.Error().Err(err).Interface("idStr", idStr).Msg("failed to parse id")
		respond.Fail(c.Writer, http.StatusBadRequest, fmt.Errorf("invalid id"))
		return
	}

	if id == uuid.Nil {
		zlog.Logger.Warn().Msg("missing id")
		respond.Fail(c.Writer, http.StatusBadRequest, fmt.Errorf("missing id"))
		return
	}

	status, err := h.service.GetNotificationStatusByID(c, h.cfg.Retry, id)
	if err != nil {
		if errors.Is(err, notification.ErrNotificationNotFound) {
			zlog.Logger.Warn().Interface("id", id).Err(err).Msg("notification not found")
			respond.Fail(c.Writer, http.StatusNotFound, fmt.Errorf("notification not found"))
		}

		zlog.Logger.Error().Err(err).Interface("id", id).Msg("failed to get notification status")
		respond.Fail(c.Writer, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return
	}

	respond.OK(c.Writer, status)
}

func (h *Handler) Delete(c *ginext.Context) {
	idStr := chi.URLParam(c.Request, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		zlog.Logger.Error().Err(err).Interface("idStr", idStr).Msg("failed to parse id")
		respond.Fail(c.Writer, http.StatusBadRequest, fmt.Errorf("invalid id"))
		return
	}

	if id == uuid.Nil {
		zlog.Logger.Warn().Msg("missing id")
		respond.Fail(c.Writer, http.StatusBadRequest, fmt.Errorf("missing id"))
		return
	}

	err = h.service.DeleteNotification(c, id)
	if err != nil {
		if errors.Is(err, notification.ErrNotificationNotFound) {
			zlog.Logger.Warn().Interface("id", id).Err(err).Msg("notification not found")
			respond.Fail(c.Writer, http.StatusNotFound, fmt.Errorf("notification not found"))
		}

		zlog.Logger.Error().Err(err).Interface("id", id).Msg("failed to delete notification")
		respond.Fail(c.Writer, http.StatusInternalServerError, fmt.Errorf("internal server error"))
		return
	}

	respond.OK(c.Writer, "event deleted")
}
