-- +goose Up
ALTER TABLE plays
ADD COLUMN require_waitlist BOOLEAN NOT NULL DEFAULT 0; -- joiners request a spot; a host adds each player to the game or waitlist

-- +goose Down
ALTER TABLE plays DROP COLUMN require_waitlist;
