package dto

import "time"

type CreateRequest struct {
	Message string    `json:"message" validate:"required"`
	SendAt  time.Time `json:"send_at" validate:"required"`
	Retries int       `json:"retries" validate:"required"`
}
