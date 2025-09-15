-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS notifications
(
    id         UUID PRIMARY KEY             DEFAULT gen_random_uuid(),
    message    TEXT                NOT NULL,
    send_at    TIMESTAMP           NOT NULL,
    status     notification_status NOT NULL DEFAULT 'pending',
    retries    INT                 NOT NULL DEFAULT 0,
    "to"       TEXT                NOT NULL,
    channel    TEXT                NOT NULL,
    created_at TIMESTAMP                    DEFAULT NOW(),
    updated_at TIMESTAMP                    DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS notifications;
-- +goose StatementEnd
