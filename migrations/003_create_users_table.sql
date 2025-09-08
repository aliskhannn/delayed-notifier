-- +goose Up
-- +goose StatementBegin
CREATE TABLE users
(
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT,
    email         TEXT,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMP        DEFAULT now(),
    updated_at    TIMESTAMP        DEFAULT now()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
