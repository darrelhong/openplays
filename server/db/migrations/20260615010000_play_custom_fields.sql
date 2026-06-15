-- +goose Up

ALTER TABLE plays ADD COLUMN name TEXT;
ALTER TABLE plays ADD COLUMN description TEXT;

-- +goose Down

ALTER TABLE plays DROP COLUMN description;
ALTER TABLE plays DROP COLUMN name;
