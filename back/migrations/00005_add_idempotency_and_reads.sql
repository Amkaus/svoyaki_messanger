-- +goose Up
-- +goose StatementBegin

ALTER TABLE messages ADD COLUMN IF NOT EXISTS client_msg_id VARCHAR(36) UNIQUE;

CREATE TABLE IF NOT EXISTS message_reads (
    message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (message_id, user_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS message_reads;
ALTER TABLE messages DROP COLUMN IF EXISTS client_msg_id;
-- +goose StatementEnd