package model

import (
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID        uuid.UUID `json:"id"`
	Message   string    `json:"message"`
	SendAt    time.Time `json:"send_at"`
	Status    string    `json:"status"`
	Retries   int       `json:"retries"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
