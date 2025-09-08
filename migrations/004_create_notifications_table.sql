-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS notifications
(
    id          UUID PRIMARY KEY             DEFAULT gen_random_uuid(),
    user_id     UUID                NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    message     TEXT                NOT NULL,
    send_at     TIMESTAMP           NOT NULL,
    status      notification_status NOT NULL DEFAULT 'pending',
    retry_count INT                 NOT NULL DEFAULT 0
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS notifications;
-- +goose StatementEnd
