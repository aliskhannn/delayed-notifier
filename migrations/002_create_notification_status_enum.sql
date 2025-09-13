-- +goose Up
-- +goose StatementBegin
CREATE TYPE notification_status AS ENUM ('pending', 'sent', 'cancelled', 'failed');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE IF EXISTS notification_status;
-- +goose StatementEnd