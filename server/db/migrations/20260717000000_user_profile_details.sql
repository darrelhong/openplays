-- +goose Up
ALTER TABLE users ADD COLUMN bio TEXT;
ALTER TABLE users ADD COLUMN profile_links TEXT;

-- +goose Down
ALTER TABLE users DROP COLUMN profile_links;
ALTER TABLE users DROP COLUMN bio;
