package notification

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/wb-go/wbf/dbpg"

	"github.com/aliskhannn/delayed-notifier/internal/model"
)

func setupMockDB(t *testing.T) (*Repository, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock db: %v", err)
	}

	wrappedDB := &dbpg.DB{Master: db}
	repo := NewRepository(wrappedDB)

	return repo, mock
}

func TestCreateNotification(t *testing.T) {
	repo, mock := setupMockDB(t)

	notificationID := uuid.New()
	n := model.Notification{
		Message: "This is a test notification",
		SendAt:  time.Now(),
		Retries: 0,
		To:      "user@example.com",
		Channel: "email",
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO notifications (
		    message, send_at, retries, "to", channel
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id;
    `)).
		WithArgs(n.Message, n.SendAt, n.Retries, n.To, n.Channel).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(notificationID))

	id, err := repo.CreateNotification(context.Background(), n)
	assert.NoError(t, err)
	assert.Equal(t, notificationID, id)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateStatus(t *testing.T) {
	repo, mock := setupMockDB(t)

	id := uuid.New()
	newStatus := "sent"

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE notifications
		SET status = $1
		WHERE id = $2;
    `)).
		WithArgs(newStatus, id).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateStatus(context.Background(), id, newStatus)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE notifications
		SET status = $1
		WHERE id = $2;
    `)).
		WithArgs(newStatus, id).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.UpdateStatus(context.Background(), id, newStatus)
	assert.ErrorIs(t, err, ErrNotificationNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNotificationStatusByID(t *testing.T) {
	repo, mock := setupMockDB(t)

	id := uuid.New()
	status := "pending"

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT message, send_at, "to", channel, status
		FROM notifications
		WHERE id = $1;
    `)).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(status))

	gotStatus, err := repo.GetNotificationStatusByID(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, status, gotStatus)
	assert.NoError(t, mock.ExpectationsWereMet())

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT message, send_at, "to", channel, status
		FROM notifications
		WHERE id = $1;
    `)).
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)

	gotStatus, err = repo.GetNotificationStatusByID(context.Background(), id)
	assert.ErrorIs(t, err, ErrNotificationNotFound)
	assert.Equal(t, "", gotStatus)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAllNotifications(t *testing.T) {
	repo, mock := setupMockDB(t)

	n1 := model.Notification{
		ID:      uuid.New(),
		Message: "msg1",
		SendAt:  time.Now(),
		Retries: 0,
		To:      "a@example.com",
		Channel: "email",
		Status:  "pending",
	}
	n2 := model.Notification{
		ID:      uuid.New(),
		Message: "msg2",
		SendAt:  time.Now(),
		Retries: 1,
		To:      "b@example.com",
		Channel: "telegram",
		Status:  "sent",
	}

	rows := sqlmock.NewRows([]string{"id", "message", "send_at", "retries", "to", "channel", "status"}).
		AddRow(n1.ID, n1.Message, n1.SendAt, n1.Retries, n1.To, n1.Channel, n1.Status).
		AddRow(n2.ID, n2.Message, n2.SendAt, n2.Retries, n2.To, n2.Channel, n2.Status)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, message, send_at, retries, "to", channel, status
		FROM notifications
		ORDER BY send_at DESC;
    `)).WillReturnRows(rows)

	list, err := repo.GetAllNotifications(context.Background())
	assert.NoError(t, err)
	assert.Len(t, list, 2)
	assert.NoError(t, mock.ExpectationsWereMet())

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, message, send_at, retries, "to", channel, status
		FROM notifications
		ORDER BY send_at DESC;
    `)).WillReturnRows(sqlmock.NewRows([]string{"id", "message", "send_at", "retries", "to", "channel", "status"}))

	_, err = repo.GetAllNotifications(context.Background())
	assert.ErrorIs(t, err, ErrNoNotificationsFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}
