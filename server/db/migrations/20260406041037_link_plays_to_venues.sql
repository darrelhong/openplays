-- +goose Up
ALTER TABLE plays ADD COLUMN venue_id INTEGER REFERENCES venues(id);

DROP INDEX IF EXISTS idx_plays_dedup;
CREATE UNIQUE INDEX idx_plays_dedupe ON plays(host_name, starts_at, ends_at, sport, venue_id);

-- +goose Down
DROP INDEX IF EXISTS idx_plays_dedupe;
ALTER TABLE plays DROP COLUMN venue_id;
CREATE UNIQUE INDEX idx_plays_dedup ON plays(host_name, starts_at, venue);
