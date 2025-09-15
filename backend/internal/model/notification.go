package model

import (
	"time"

	"github.com/google/uuid"
)

// Notification represents a notification entity in the system.
type Notification struct {
	ID        uuid.UUID `json:"id"`         // unique identifier for the notification
	Message   string    `json:"message"`    // content of the notification
	SendAt    time.Time `json:"send_at"`    // time when the notification should be sent
	Status    string    `json:"status"`     // current state, e.g., "pending", "sent", "failed", "cancelled"
	Retries   int       `json:"retries"`    // number of retry attempts on failure
	Channel   string    `json:"channel"`    // delivery method, e.g., "email", "telegram"
	To        string    `json:"to"`         // recipient identifier, such as email or chat ID
	CreatedAt time.Time `json:"created_at"` // timestamp when the notification was created
	UpdatedAt time.Time `json:"updated_at"` // timestamp when the notification was last updated
}
