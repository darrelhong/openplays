-- +goose Up
-- venue_norm was a denormalized copy of venues.name, now redundant since
-- we JOIN on venue_id and COALESCE in queries.
ALTER TABLE plays DROP COLUMN venue_norm;

-- +goose Down
ALTER TABLE plays ADD COLUMN venue_norm TEXT NOT NULL DEFAULT '';
