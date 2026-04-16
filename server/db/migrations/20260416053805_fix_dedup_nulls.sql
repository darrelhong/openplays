-- +goose Up
-- SQLite treats NULLs as distinct in unique indexes (SQL standard behavior).
-- Plays with NULL level_min/level_max (e.g. sell_booking) bypass the dedup index.
-- Fix: use COALESCE to replace NULLs with sentinel values in the index expression.
-- Also remove ends_at from the index — LLM may parse end times inconsistently.
DROP INDEX IF EXISTS idx_plays_dedupe;
CREATE UNIQUE INDEX idx_plays_dedupe ON plays(
    host_name, starts_at, sport,
    COALESCE(level_min, ''), COALESCE(level_max, ''),
    COALESCE(venue_id, 0)
);

-- +goose Down
DROP INDEX IF EXISTS idx_plays_dedupe;
CREATE UNIQUE INDEX idx_plays_dedupe ON plays(
    host_name, starts_at, ends_at, sport,
    level_min, level_max, venue_id
);
