-- +goose Up
CREATE TABLE venues (
    postal_code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    address TEXT NOT NULL,
    latitude REAL NOT NULL,
    longitude REAL NOT NULL,
    source TEXT NOT NULL, -- e.g 'onemap' | 'manual'
    search_term TEXT -- the raw query used to find this venue (for auditing)
);

CREATE TABLE venue_aliases (
    alias TEXT PRIMARY KEY,
    venue_postal_code TEXT NOT NULL REFERENCES venues(postal_code)
);

-- +goose Down
DROP TABLE IF EXISTS venue_aliases;
DROP TABLE IF EXISTS venues;
