-- +goose Up

ALTER TABLE play_participants ADD COLUMN invited_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL;

-- +goose Down

ALTER TABLE play_participants DROP COLUMN invited_by_user_id;
