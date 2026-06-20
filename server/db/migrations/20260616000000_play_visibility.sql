-- +goose Up
ALTER TABLE plays
ADD COLUMN visibility TEXT NOT NULL DEFAULT 'public'; -- public | unlisted

CREATE INDEX IF NOT EXISTS idx_plays_public_list_order
    ON plays(visibility, listing_type, starts_at, sport, id)
    WHERE cancelled_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_plays_public_list_order;
ALTER TABLE plays DROP COLUMN visibility;
