-- +goose Up
-- Dedup should not split rows by level range.
DROP INDEX IF EXISTS idx_plays_dedupe;
CREATE UNIQUE INDEX idx_plays_dedupe ON plays(
    host_name, starts_at, sport,
    COALESCE(venue_id, 0)
);

-- +goose Down
DROP INDEX IF EXISTS idx_plays_dedupe;
CREATE UNIQUE INDEX idx_plays_dedupe ON plays(
    host_name, starts_at, sport,
    COALESCE(level_min, ''), COALESCE(level_max, ''),
    COALESCE(venue_id, 0)
);
