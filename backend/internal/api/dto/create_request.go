package dto

type CreateRequest struct {
	Message string `json:"message" validate:"required"`
	SendAt  string `json:"send_at" validate:"required"`
	Retries int    `json:"retries" validate:"required"`
	To      string `json:"to" validate:"required"`
	Channel string `json:"channel" validate:"required"`
}
