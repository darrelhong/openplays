-- +goose Up
ALTER TABLE raw_messages ADD COLUMN source_message_id TEXT;
ALTER TABLE raw_messages ADD COLUMN source_group TEXT;

ALTER TABLE plays ADD COLUMN source_message_id TEXT;
ALTER TABLE plays ADD COLUMN source_group TEXT;

-- +goose Down
ALTER TABLE plays DROP COLUMN source_group;
ALTER TABLE plays DROP COLUMN source_message_id;
ALTER TABLE raw_messages DROP COLUMN source_group;
ALTER TABLE raw_messages DROP COLUMN source_message_id;
