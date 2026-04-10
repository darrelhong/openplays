-- +goose Up

-- Split sender identity: sender_username is strictly the Telegram @username
-- (nullable when the user has none), sender_name is the display name
-- (first+last or username fallback). Previously sender_username held both.
ALTER TABLE raw_messages ADD COLUMN sender_name TEXT NOT NULL DEFAULT '';
ALTER TABLE plays ADD COLUMN source_sender_name TEXT;

-- +goose Down
ALTER TABLE plays DROP COLUMN source_sender_name;
ALTER TABLE raw_messages DROP COLUMN sender_name;
