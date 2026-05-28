-- +goose Up

ALTER TABLE plays ADD COLUMN cancelled_at TIMESTAMP;
ALTER TABLE plays ADD COLUMN cancelled_by TEXT REFERENCES users(id);

CREATE INDEX idx_plays_active_list_order
    ON plays(listing_type, cancelled_at, starts_at, sport, id);

-- +goose Down

DROP INDEX IF EXISTS idx_plays_active_list_order;
ALTER TABLE plays DROP COLUMN cancelled_by;
ALTER TABLE plays DROP COLUMN cancelled_at;
