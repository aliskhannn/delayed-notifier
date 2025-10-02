package notification

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/dbpg"

	"github.com/aliskhannn/delayed-notifier/internal/model"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrNoNotificationsFound = errors.New("no notifications found")
)

// Repository provides methods to interact with notifications table.
type Repository struct {
	db *dbpg.DB
}

// NewRepository creates a new notification repository.
func NewRepository(db *dbpg.DB) *Repository {
	return &Repository{db: db}
}

// CreateNotification inserts a new notification into the database and returns its ID.
func (r *Repository) CreateNotification(ctx context.Context, notification model.Notification) (uuid.UUID, error) {
	query := `
		INSERT INTO notifications (
		    message, send_at, retries, "to", channel
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id;
    `

	err := r.db.QueryRowContext(
		ctx, query, notification.Message, notification.SendAt, notification.Retries, notification.To, notification.Channel,
	).Scan(&notification.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create notification: %w", err)
	}

	return notification.ID, nil
}

// UpdateStatus updates the status of a notification by its ID.
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE notifications
		SET status = $1
		WHERE id = $2;
    `

	res, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
	}

	rows, _ := res.RowsAffected()

	if rows == 0 {
		return ErrNotificationNotFound
	}

	return nil
}

// GetNotificationStatusByID retrieves the status of a notification by its ID.
func (r *Repository) GetNotificationStatusByID(ctx context.Context, id uuid.UUID) (string, error) {
	query := `
		SELECT message, send_at, "to", channel, status
		FROM notifications
		WHERE id = $1;
    `

	var status string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotificationNotFound
		}

		return "", fmt.Errorf("failed to get notification status: %w", err)
	}

	return status, nil
}

// GetAllNotifications retrieves all notifications ordered by SendAt descending.
func (r *Repository) GetAllNotifications(ctx context.Context) ([]model.Notification, error) {
	query := `
		SELECT id, message, send_at, retries, "to", channel, status
		FROM notifications
		ORDER BY send_at DESC;
    `

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all notifications: %w", err)
	}
	defer rows.Close()

	var notifications []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.Message, &n.SendAt, &n.Retries, &n.To, &n.Channel, &n.Status); err != nil {
			return nil, err
		}

		notifications = append(notifications, n)
	}

	if len(notifications) == 0 {
		return nil, ErrNoNotificationsFound
	}

	return notifications, nil
}
