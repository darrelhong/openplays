-- +goose Up
ALTER TABLE venues ADD COLUMN google_place_id TEXT;
CREATE UNIQUE INDEX idx_venues_google_place_id ON venues(google_place_id);

-- +goose Down
DROP INDEX IF EXISTS idx_venues_google_place_id;
ALTER TABLE venues DROP COLUMN google_place_id;
