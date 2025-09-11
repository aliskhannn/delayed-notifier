package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/dbpg"

	"github.com/aliskhannn/delayed-notifier/internal/model"
)

var ErrNotificationNotFound = errors.New("notification not found")

type Repository struct {
	db *dbpg.DB
}

func NewRepository(db *dbpg.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateNotification(ctx context.Context, notification model.Notification) (uuid.UUID, error) {
	query := `
		INSERT INTO notifications (
		    message, send_at, retries
		) VALUES ($1, $2, $3)
		RETURNING id;
    `

	err := r.db.Master.QueryRowContext(
		ctx, query, notification.Message, notification.SendAt, notification.Retries,
	).Scan(&notification.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create notification: %w", err)
	}

	return notification.ID, nil
}

func (r *Repository) GetNotificationStatusByID(ctx context.Context, id uuid.UUID) (string, error) {
	query := `
		SELECT status
		FROM notifications
		WHERE id = $1;
    `

	var status string
	// TODO: заменить на встроенный метод QueryRowContext с поддержкой master и slave с round-robin
	err := r.db.Master.QueryRowContext(ctx, query, id).Scan(&status)
	if err != nil {
		return "", fmt.Errorf("failed to get notification status: %w", err)
	}

	if status == "" {
		return "", ErrNotificationNotFound
	}

	return status, nil
}

func (r *Repository) DeleteNotification(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM notifications
		WHERE id = $1;
    `

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return ErrNotificationNotFound
	}

	return nil
}
